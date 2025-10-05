package seckill

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/redis/go-redis/v9"
	"seckill/pkg/log"
)

// MultiLevelInventory multi-level inventory manager
type MultiLevelInventory struct {
	// L1: local memory cache
	localCache *bigcache.BigCache

	// L2: Redis
	redis *redis.Client

	// Bloom filter (prevent cache penetration)
	bloomFilter *bloom.BloomFilter
}

// NewMultiLevelInventory creates a new multi-level inventory manager
func NewMultiLevelInventory(redis *redis.Client) (*MultiLevelInventory, error) {
	// Initialize local cache (1 minute expiration)
	localCache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(1*time.Minute))
	if err != nil {
		return nil, err
	}

	// Initialize bloom filter (estimate 10000 elements, false positive rate 0.01)
	bloomFilter := bloom.NewWithEstimates(10000, 0.01)

	return &MultiLevelInventory{
		localCache:  localCache,
		redis:       redis,
		bloomFilter: bloomFilter,
	}, nil
}

// DeductRequest stock deduction request
type DeductRequest struct {
	RequestID  string `json:"request_id"`
	ActivityID uint64 `json:"activity_id"`
	UserID     uint64 `json:"user_id"`
	Quantity   int    `json:"quantity"`
}

// DeductResult stock deduction result
type DeductResult struct {
	Success     bool   `json:"success"`
	DeductID    string `json:"deduct_id"`
	Message     string `json:"message"`
	RemainStock int    `json:"remain_stock"`
}

// LocalCheck local cache quick check (bloom filter)
func (m *MultiLevelInventory) LocalCheck(activityID uint64) bool {
	// Check bloom filter, if not exists then definitely sold out
	key := []byte(fmt.Sprintf("stock:%d", activityID))
	return m.bloomFilter.Test(key)
}

// MarkSoldOut mark as sold out (update bloom filter)
func (m *MultiLevelInventory) MarkSoldOut(activityID uint64) {
	// Note: Bloom filter doesn't support deletion
	// In production, use counting bloom filter or other methods
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Activity sold out")
}

// AddToBloomFilter add to bloom filter
func (m *MultiLevelInventory) AddToBloomFilter(activityID uint64) {
	key := []byte(fmt.Sprintf("stock:%d", activityID))
	m.bloomFilter.Add(key)
}

// TryDeduct Try phase: pre-deduct stock
func (m *MultiLevelInventory) TryDeduct(ctx context.Context, req *DeductRequest) (*DeductResult, error) {
	// Generate deduction ID (ensure idempotency)
	deductID := fmt.Sprintf("deduct:%s:%d", req.RequestID, time.Now().UnixNano())

	// Check if already processed
	existKey := fmt.Sprintf("deduct_result:%s", req.RequestID)
	if exists, _ := m.redis.Exists(ctx, existKey).Result(); exists > 0 {
		// Return existing result
		data, _ := m.redis.Get(ctx, existKey).Bytes()
		var result DeductResult
		json.Unmarshal(data, &result)
		return &result, nil
	}

	// Execute Lua script for atomic deduction
	stockKey := fmt.Sprintf("stock:%d", req.ActivityID)
	reserveKey := fmt.Sprintf("stock:reserved:%d", req.ActivityID)
	logKey := fmt.Sprintf("stock:deduct_log:%d", req.ActivityID)

	script := `
		local stock_key = KEYS[1]
		local reserve_key = KEYS[2]
		local deduct_log_key = KEYS[3]
		local deduct_id = ARGV[1]
		local quantity = tonumber(ARGV[2])
		local expire_time = tonumber(ARGV[3])

		-- Get current stock
		local current_stock = tonumber(redis.call('GET', stock_key) or 0)

		-- Check stock availability
		if current_stock < quantity then
			return {0, 'insufficient_stock', current_stock}
		end

		-- Pre-deduct stock (transfer to reserved stock)
		redis.call('DECRBY', stock_key, quantity)
		redis.call('INCRBY', reserve_key, quantity)

		-- Record deduction log
		local log_data = cjson.encode({
			deduct_id = deduct_id,
			quantity = quantity,
			timestamp = redis.call('TIME')[1],
			status = 'try'
		})
		redis.call('HSET', deduct_log_key, deduct_id, log_data)
		redis.call('EXPIRE', deduct_log_key, expire_time)

		-- Set deduction record expiration (15 minutes)
		redis.call('SETEX', 'deduct_record:' .. deduct_id, expire_time, log_data)

		return {1, 'success', current_stock - quantity}
	`

	result, err := m.redis.Eval(ctx, script,
		[]string{stockKey, reserveKey, logKey},
		deductID, req.Quantity, 900).Result()

	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Redis eval failed")
		return nil, err
	}

	// Parse result
	resultArray := result.([]interface{})
	success := resultArray[0].(int64)
	message := resultArray[1].(string)
	remainStock := int(resultArray[2].(int64))

	deductResult := &DeductResult{
		Success:     success == 1,
		DeductID:    deductID,
		Message:     message,
		RemainStock: remainStock,
	}

	// Cache result (15 minutes)
	resultData, _ := json.Marshal(deductResult)
	m.redis.SetEx(ctx, existKey, resultData, 15*time.Minute)

	// If stock is 0, mark as sold out
	if success == 1 && remainStock == 0 {
		m.MarkSoldOut(req.ActivityID)
	}

	return deductResult, nil
}

// ConfirmDeduct Confirm phase: confirm deduction
func (m *MultiLevelInventory) ConfirmDeduct(ctx context.Context, deductID string, activityID uint64) error {
	script := `
		local deduct_record_key = KEYS[1]
		local reserve_key = KEYS[2]

		-- Get deduction record
		local log_data = redis.call('GET', deduct_record_key)
		if not log_data then
			return {0, 'deduct_record_not_found'}
		end

		local log = cjson.decode(log_data)

		-- Check status
		if log.status == 'confirmed' then
			return {1, 'already_confirmed'}
		end

		if log.status == 'cancelled' then
			return {0, 'already_cancelled'}
		end

		-- Deduct from reserved stock (confirm deduction)
		local reserve_quantity = tonumber(log.quantity)
		redis.call('DECRBY', reserve_key, reserve_quantity)

		-- Update status to confirmed
		log.status = 'confirmed'
		log.confirm_time = redis.call('TIME')[1]
		redis.call('SET', deduct_record_key, cjson.encode(log))

		return {1, 'success'}
	`

	recordKey := fmt.Sprintf("deduct_record:%s", deductID)
	reserveKey := fmt.Sprintf("stock:reserved:%d", activityID)

	_, err := m.redis.Eval(ctx, script,
		[]string{recordKey, reserveKey},
		deductID).Result()

	if err != nil {
		log.WithFields(map[string]interface{}{
			"deduct_id": deductID,
			"error":     err.Error(),
		}).Error("Confirm deduct failed")
		return err
	}

	log.WithFields(map[string]interface{}{
		"deduct_id": deductID,
	}).Info("Stock deduction confirmed successfully")
	return nil
}

// CancelDeduct Cancel phase: cancel deduction (rollback)
func (m *MultiLevelInventory) CancelDeduct(ctx context.Context, deductID string, activityID uint64) error {
	script := `
		local stock_key = KEYS[1]
		local reserve_key = KEYS[2]
		local deduct_record_key = KEYS[3]

		-- Get deduction record
		local log_data = redis.call('GET', deduct_record_key)
		if not log_data then
			return {0, 'deduct_record_not_found'}
		end

		local log = cjson.decode(log_data)

		-- Check status
		if log.status == 'cancelled' then
			return {1, 'already_cancelled'}
		end

		if log.status == 'confirmed' then
			return {0, 'already_confirmed'}
		end

		-- Rollback stock
		local quantity = tonumber(log.quantity)
		redis.call('INCRBY', stock_key, quantity)
		redis.call('DECRBY', reserve_key, quantity)

		-- Update status to cancelled
		log.status = 'cancelled'
		log.cancel_time = redis.call('TIME')[1]
		redis.call('SET', deduct_record_key, cjson.encode(log))

		return {1, 'success'}
	`

	stockKey := fmt.Sprintf("stock:%d", activityID)
	reserveKey := fmt.Sprintf("stock:reserved:%d", activityID)
	recordKey := fmt.Sprintf("deduct_record:%s", deductID)

	_, err := m.redis.Eval(ctx, script,
		[]string{stockKey, reserveKey, recordKey},
		deductID).Result()

	if err != nil {
		log.WithFields(map[string]interface{}{
			"deduct_id": deductID,
			"error":     err.Error(),
		}).Error("Cancel deduct failed")
		return err
	}

	log.WithFields(map[string]interface{}{
		"deduct_id": deductID,
	}).Info("Stock deduction cancelled successfully")
	return nil
}

// SyncToRedis sync stock to Redis
func (m *MultiLevelInventory) SyncToRedis(ctx context.Context, activityID uint64, stock int) error {
	stockKey := fmt.Sprintf("stock:%d", activityID)

	// Set stock (24 hours expiration)
	err := m.redis.Set(ctx, stockKey, stock, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	// Add to bloom filter
	m.AddToBloomFilter(activityID)

	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
		"stock":       stock,
	}).Info("Stock synced to Redis")
	return nil
}

// GetStockFromRedis get stock from Redis
func (m *MultiLevelInventory) GetStockFromRedis(ctx context.Context, activityID uint64) (int, error) {
	stockKey := fmt.Sprintf("stock:%d", activityID)
	stock, err := m.redis.Get(ctx, stockKey).Int()
	if err != nil {
		return 0, err
	}
	return stock, nil
}

