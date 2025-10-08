package seckill

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/pmylund/go-bloom"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// MultiLevelInventory multi-level inventory manager
type MultiLevelInventory struct {
	// L1: local memory cache
	localCache *bigcache.BigCache

	// L2: Redis
	redisClient redis.Cmdable

	// Bloom filter (prevent cache penetration)
	bloomFilter *bloom.CountingFilter

	mu sync.RWMutex
}

// NewMultiLevelInventory creates a new multi-level inventory manager
func NewMultiLevelInventory(redisClient redis.Cmdable) (*MultiLevelInventory, error) {
	// Initialize local cache (1 minute expiration)
	localCache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(10*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("failed to create local cache: %w", err)
	}

	// Initialize bloom filter (estimate 10000 elements, false positive rate 0.01)
	bloomFilter := bloom.NewCounting(10000, 0.01)

	return &MultiLevelInventory{
		localCache:  localCache,
		redisClient: redisClient,
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
func (m *MultiLevelInventory) LocalCheck(ctx context.Context, activityID uint64) bool {
	// Check bloom filter, if not exists then definitely sold out
	goodsKey := fmt.Sprintf("goods:{%d}", activityID)
	if !m.bloomFilter.Test([]byte(goodsKey)) {
		logrus.WithField("activity_id", activityID).Debug("Activity not found in bloom filter")
		return false
	}

	// Check local cache for sold out status
	soldOutKey := fmt.Sprintf("sold_out:{%d}", activityID)
	_, err := m.localCache.Get(soldOutKey)

	// If found in sold out cache, return false (sold out)
	// If not found in sold out cache, return true (potentially available)
	return err != nil
}

// MarkSoldOut mark as sold out (update bloom filter)
func (m *MultiLevelInventory) MarkSoldOut(activityID uint64) error {
	soldOutKey := fmt.Sprintf("sold_out:{%d}", activityID)
	err := m.localCache.Set(soldOutKey, []byte("1"))
	if err != nil {
		return fmt.Errorf("failed to mark activity as sold out in local cache: %w", err)
	}

	// Remove from bloom filter to prevent unnecessary cache queries
	bloomKey := fmt.Sprintf("goods:{%d}", activityID)
	m.bloomFilter.Remove([]byte(bloomKey))

	return nil
}

// AddToBloomFilter add to bloom filter
func (m *MultiLevelInventory) AddToBloomFilter(activityID uint64) {
	key := []byte(fmt.Sprintf("stock:%d", activityID))
	m.bloomFilter.Add(key)
}

// TryDeduct Try phase: pre-deduct stock
func (m *MultiLevelInventory) TryDeductWithLimit(ctx context.Context, req *DeductRequest, limitPerUser int) (*DeductResult, error) {
	if !m.LocalCheck(ctx, req.ActivityID) {
		return &DeductResult{
			Success:     false,
			DeductID:    "",
			Message:     "活动已售罄",
			RemainStock: 0,
		}, nil
	}

	// Generate deduction ID (ensure idempotency)
	deductID := fmt.Sprintf("deduct:%s:%d", req.RequestID, time.Now().UnixNano())

	// Check if already processed
	existKey := fmt.Sprintf("deduct_result:%s", req.RequestID)
	if exists, _ := m.redisClient.Exists(ctx, existKey).Result(); exists > 0 {
		// Return existing result
		data, _ := m.redisClient.Get(ctx, existKey).Bytes()
		var result DeductResult
		json.Unmarshal(data, &result)
		return &result, nil
	}

	// Execute Lua script for atomic deduction with purchase limit check
	// Use hash tag to ensure all keys are in the same slot for Redis cluster
	stockKey := fmt.Sprintf("stock:{%d}", req.ActivityID)
	reserveKey := fmt.Sprintf("stock:reserved:{%d}", req.ActivityID)
	logKey := fmt.Sprintf("stock:deduct_log:{%d}", req.ActivityID)
	purchaseCountKey := fmt.Sprintf("purchase_count:{%d}:%d", req.ActivityID, req.UserID)

	script := `
		local stock_key = KEYS[1]
		local reserve_key = KEYS[2]
		local deduct_log_key = KEYS[3]
		local purchase_count_key = KEYS[4]
		local deduct_id = ARGV[1]
		local quantity = tonumber(ARGV[2])
		local expire_time = tonumber(ARGV[3])
		local limit_per_user = tonumber(ARGV[4])

		-- Debug: Log the parameters
		redis.log(redis.LOG_NOTICE, "TryDeductWithLimit: purchase_count_key=" .. purchase_count_key .. ", limit_per_user=" .. limit_per_user)

		-- Atomically increment purchase count and check limit
		local new_purchase_count = redis.call('INCRBY', purchase_count_key, quantity)
		redis.call('EXPIRE', purchase_count_key, 86400) -- 24 hours expiration
		
		-- Debug: Log the purchase count
		redis.log(redis.LOG_NOTICE, "TryDeductWithLimit: new_purchase_count=" .. new_purchase_count .. ", limit=" .. limit_per_user)
		
		-- Check if the new count exceeds the limit
		if new_purchase_count > limit_per_user then
			-- Rollback the increment
			redis.call('DECRBY', purchase_count_key, quantity)
			redis.log(redis.LOG_NOTICE, "TryDeductWithLimit: Purchase limit exceeded, rolling back")
			return {0, 'purchase_limit_exceeded', 0}
		end

		-- Get current stock
		local current_stock = tonumber(redis.call('GET', stock_key) or 0)

		-- Check stock availability
		if current_stock < quantity then
			-- Rollback the purchase count increment
			redis.call('DECRBY', purchase_count_key, quantity)
			redis.log(redis.LOG_NOTICE, "TryDeductWithLimit: Insufficient stock, rolling back")
			return {0, 'insufficient_stock', current_stock}
		end

		-- Pre-deduct stock (transfer to reserved stock)
		redis.call('DECRBY', stock_key, quantity)
		redis.call('INCRBY', reserve_key, quantity)

		redis.log(redis.LOG_NOTICE, "TryDeductWithLimit: Success, stock deducted")

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
		redis.call('SETEX', 'deduct_record:{' .. ARGV[5] .. '}:' .. deduct_id, expire_time, log_data)

		return {1, 'success', current_stock - quantity}
	`

	result, err := m.redisClient.Eval(ctx, script,
		[]string{stockKey, reserveKey, logKey, purchaseCountKey},
		deductID, req.Quantity, 900, limitPerUser, req.ActivityID).Result()

	if err != nil {
		logrus.WithField("error", err.Error()).Error("Redis eval failed")
		return nil, err
	}

	// Parse result
	resultSlice := result.([]interface{})
	success := resultSlice[0].(int64) == 1
	message := resultSlice[1].(string)
	remainStock := int(resultSlice[2].(int64))

	deductResult := &DeductResult{
		Success:     success,
		DeductID:    deductID,
		Message:     message,
		RemainStock: remainStock,
	}

	// Cache result for idempotency (5 minutes)
	if data, _ := json.Marshal(deductResult); data != nil {
		m.redisClient.SetEx(ctx, existKey, data, 5*time.Minute)
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

	recordKey := fmt.Sprintf("deduct_record:{%d}:%s", activityID, deductID)
	reserveKey := fmt.Sprintf("stock:reserved:{%d}", activityID)

	_, err := m.redisClient.Eval(ctx, script,
		[]string{recordKey, reserveKey},
		deductID).Result()

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"deduct_id": deductID,
			"error":     err.Error(),
		}).Error("Confirm deduct failed")
		return err
	}

	logrus.WithField("deduct_id", deductID).Info("Stock deduction confirmed successfully")
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

	stockKey := fmt.Sprintf("stock:{%d}", activityID)
	reserveKey := fmt.Sprintf("stock:reserved:{%d}", activityID)
	recordKey := fmt.Sprintf("deduct_record:{%d}:%s", activityID, deductID)

	_, err := m.redisClient.Eval(ctx, script,
		[]string{stockKey, reserveKey, recordKey},
		deductID).Result()

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"deduct_id": deductID,
			"error":     err.Error(),
		}).Error("Cancel deduct failed")
		return err
	}

	logrus.WithField("deduct_id", deductID).Info("Stock deduction cancelled successfully")
	return nil
}

// SyncToRedis sync stock to Redis
func (m *MultiLevelInventory) SyncToRedis(ctx context.Context, activityID uint64, stock int) error {
	stockKey := fmt.Sprintf("stock:{%d}", activityID)

	// Set stock (24 hours expiration)
	err := m.redisClient.Set(ctx, stockKey, stock, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	// Add to bloom filter
	m.AddToBloomFilter(activityID)

	logrus.WithFields(logrus.Fields{
		"activity_id": activityID,
		"stock":       stock,
	}).Info("Stock synced to Redis")
	return nil
}

// GetStockFromRedis get stock from Redis
func (m *MultiLevelInventory) GetStockFromRedis(ctx context.Context, activityID uint64) (int, error) {
	stockKey := fmt.Sprintf("stock:{%d}", activityID)
	stock, err := m.redisClient.Get(ctx, stockKey).Int()
	if err != nil {
		return 0, err
	}
	return stock, nil
}
