package seckill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"seckill/internal/model"
	"seckill/internal/repository"
	"seckill/pkg/breaker"
	"seckill/pkg/limiter"
	"seckill/pkg/log"
	"seckill/pkg/queue"
)

// SeckillService seckill service interface
type SeckillService interface {
	// Execute seckill
	DoSeckill(ctx context.Context, req *SeckillRequest) (*SeckillResult, error)

	// Prewarm activity
	PrewarmActivity(ctx context.Context, activityID uint64) error

	// Query seckill result
	QuerySeckillResult(ctx context.Context, requestID string, userID uint64) (*SeckillResult, error)
}

// seckillService seckill service implementation
type seckillService struct {
	activityRepo   repository.ActivityRepository
	inventory      *MultiLevelInventory
	rateLimiter    *limiter.MultiDimensionLimiter
	circuitBreaker *breaker.Manager
	orderQueue     queue.MessageQueue
	redis          *redis.Client
}

// NewSeckillService creates a seckill service
func NewSeckillService(
	activityRepo repository.ActivityRepository,
	inventory *MultiLevelInventory,
	rateLimiter *limiter.MultiDimensionLimiter,
	circuitBreaker *breaker.Manager,
	orderQueue queue.MessageQueue,
	redis *redis.Client,
) SeckillService {
	return &seckillService{
		activityRepo:   activityRepo,
		inventory:      inventory,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		orderQueue:     orderQueue,
		redis:          redis,
	}
}

// SeckillRequest seckill request
type SeckillRequest struct {
	RequestID  string `json:"request_id" binding:"required"` // Idempotency ID
	ActivityID uint64 `json:"activity_id" binding:"required"`
	UserID     uint64 `json:"user_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	IP         string `json:"ip"`
	DeviceID   string `json:"device_id"`
	UserAgent  string `json:"user_agent"`
}

// SeckillResult seckill result
type SeckillResult struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	OrderID   string `json:"order_id,omitempty"`
	Message   string `json:"message"`
	QueuePos  int    `json:"queue_pos,omitempty"` // Queue position
}

// DoSeckill execute seckill (16-step complete process)
func (s *seckillService) DoSeckill(ctx context.Context, req *SeckillRequest) (*SeckillResult, error) {
	startTime := time.Now()
	activityID := req.ActivityID
	userID := req.UserID

	log.WithFields(map[string]interface{}{
		"request_id":  req.RequestID,
		"activity_id": activityID,
		"user_id":     userID,
	}).Info("Start processing seckill request")

	// ========== Step 1: Idempotency check ==========
	resultKey := fmt.Sprintf("seckill:result:%s:%d", req.RequestID, userID)
	if existingResult, err := s.redis.Get(ctx, resultKey).Bytes(); err == nil {
		var result SeckillResult
		json.Unmarshal(existingResult, &result)
		log.WithFields(map[string]interface{}{
			"request_id": req.RequestID,
		}).Info("Return idempotent result")
		return &result, nil
	}

	// ========== Step 2: Parameter validation ==========
	if req.Quantity <= 0 || req.Quantity > 5 {
		return s.failResult(req.RequestID, "Invalid purchase quantity"), nil
	}

	// ========== Step 3: Bloom filter check (L1 cache) ==========
	if !s.inventory.LocalCheck(activityID) {
		log.WithFields(map[string]interface{}{
			"activity_id": activityID,
		}).Warn("Bloom filter check failed, activity may be sold out")
		return s.failResult(req.RequestID, "Activity sold out"), nil
	}

	// ========== Step 4: Circuit breaker check ==========
	cbName := fmt.Sprintf("activity:%d", activityID)
	if s.circuitBreaker.State(cbName) == breaker.StateOpen {
		log.WithFields(map[string]interface{}{
			"activity_id": activityID,
		}).Warn("Circuit breaker is open")
		return s.failResult(req.RequestID, "System busy, please try again later"), nil
	}

	// ========== Step 5: Multi-dimension rate limiting ==========
	dimensions := map[string]string{
		"global":   fmt.Sprintf("global:%d", activityID),
		"user":     fmt.Sprintf("%d", userID),
		"ip":       req.IP,
		"activity": fmt.Sprintf("%d", activityID),
	}

	if allowed, err := s.rateLimiter.Allow(ctx, dimensions); err != nil || !allowed {
		log.WithFields(map[string]interface{}{
			"user_id": userID,
			"ip":      req.IP,
		}).Warn("Rate limit exceeded")
		return s.failResult(req.RequestID, "Request too frequent, please try again later"), nil
	}

	// ========== Step 6: Activity validity check ==========
	// First try to get from Redis cache
	var activity *model.SeckillActivity
	configKey := fmt.Sprintf("activity:config:%d", activityID)
	if configData, err := s.redis.Get(ctx, configKey).Bytes(); err == nil {
		// Found in cache, unmarshal
		if err := json.Unmarshal(configData, &activity); err == nil {
			log.WithFields(map[string]interface{}{
				"activity_id": activityID,
			}).Debug("Activity loaded from Redis cache")
		} else {
			activity = nil // Fallback to database
		}
	}
	
	// If not found in cache, query from database
	if activity == nil {
		var err error
		activity, err = s.activityRepo.GetByID(ctx, int64(activityID))
		if err != nil {
			log.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to query activity")
			s.recordCircuitBreakerError(cbName)
			return nil, err
		}
		log.WithFields(map[string]interface{}{
			"activity_id": activityID,
		}).Debug("Activity loaded from database")
	}

	if !activity.IsRunning() {
		return s.failResult(req.RequestID, "Activity not started or ended"), nil
	}

	// ========== Step 7: Gray control ==========
	if !s.checkGrayControl(ctx, activity, int64(userID)) {
		log.WithFields(map[string]interface{}{
			"user_id": userID,
		}).Info("User not in gray whitelist")
		return s.failResult(req.RequestID, "Activity not available for you"), nil
	}

	// ========== Step 8: User eligibility verification (risk control) ==========
	if !s.checkUserEligibility(ctx, activityID, userID) {
		log.WithFields(map[string]interface{}{
			"user_id": userID,
		}).Warn("User eligibility verification failed")
		return s.failResult(req.RequestID, "You are not eligible for this activity"), nil
	}

	// ========== Step 9: TCC-Try phase with purchase limit check ==========
	//  need to check limit and deduct in one step ,otherwise a user can bypass per user limit

	deductReq := &DeductRequest{
		RequestID:  req.RequestID,
		ActivityID: activityID,
		UserID:     userID,
		Quantity:   req.Quantity,
	}

	deductResult, err := s.inventory.TryDeductWithLimit(ctx, deductReq, activity.LimitPerUser)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Stock deduction failed")
		s.recordCircuitBreakerError(cbName)
		return nil, err
	}

	if !deductResult.Success {
		log.WithFields(map[string]interface{}{
			"activity_id": activityID,
			"message":    deductResult.Message,
		}).Info("Deduction failed")
		return s.failResult(req.RequestID, deductResult.Message), nil
	}

	// ========== Step 10: Generate pre-order and send to message queue ==========
	orderMsg := &model.OrderMessage{
		RequestID:  req.RequestID,
		ActivityID: activityID,
		UserID:     userID,
		GoodsID:    activity.GoodsID,
		Quantity:   req.Quantity,
		Price:      activity.Price,
		DeductID:   deductResult.DeductID,
		Timestamp:  time.Now().Unix(),
	}

	orderData, _ := json.Marshal(orderMsg)
	if err := s.orderQueue.Publish(ctx, "seckill_orders", orderData); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to send order message")

		// Rollback stock (TCC-Cancel)
		s.inventory.CancelDeduct(ctx, deductResult.DeductID, activityID)
		s.recordCircuitBreakerError(cbName)
		return nil, err
	}

	// ========== Step 12: Record user purchase count ==========
	// Note: Purchase count is now incremented atomically in TryDeductWithLimit
	// s.incrUserPurchaseCount(ctx, activityID, userID, req.Quantity)

	// ========== Step 13: Record seckill log ==========
	s.recordSeckillLog(ctx, req, deductResult.DeductID, "success")

	// ========== Step 14: Construct success result ==========
	result := &SeckillResult{
		Success:   true,
		RequestID: req.RequestID,
		OrderID:   "", // Order ID will be generated asynchronously by order service
		Message:   "Seckill successful, order processing",
	}

	// ========== Step 15: Cache result (idempotency guarantee) ==========
	resultData, _ := json.Marshal(result)
	s.redis.SetEx(ctx, resultKey, resultData, 30*time.Minute)

	// ========== Step 16: Record success metrics ==========
	s.recordCircuitBreakerSuccess(cbName)
	duration := time.Since(startTime)

	log.WithFields(map[string]interface{}{
		"request_id":  req.RequestID,
		"duration_ms": duration.Milliseconds(),
	}).Info("Seckill request processed successfully")

	return result, nil
}

// checkGrayControl gray control check
func (s *seckillService) checkGrayControl(ctx context.Context, activity *model.SeckillActivity, userID int64) bool {
	// If gray ratio is 0, allow all
	if activity.GrayRatio == 0 {
		return true
	}

	// Check if in whitelist
	if activity.GrayWhitelist != nil {
		for _, id := range activity.GrayWhitelist {
			if uid, ok := id.(float64); ok && int64(uid) == userID {
				return true
			}
		}
	}

	// Random allow based on gray ratio
	if activity.GrayRatio >= 1.0 {
		return true
	}

	// Use user ID hash to determine if in gray range
	hash := userID % 100
	return float64(hash) < activity.GrayRatio*100
}

// checkUserEligibility user eligibility verification
func (s *seckillService) checkUserEligibility(ctx context.Context, activityID, userID uint64) bool {
	// Check blacklist
	blacklistKey := fmt.Sprintf("blacklist:user:%d", userID)
	if exists, _ := s.redis.Exists(ctx, blacklistKey).Result(); exists > 0 {
		return false
	}

	// Check activity blacklist
	activityBlacklistKey := fmt.Sprintf("blacklist:activity:%d:user:%d", activityID, userID)
	if exists, _ := s.redis.Exists(ctx, activityBlacklistKey).Result(); exists > 0 {
		return false
	}

	// Can add more risk control checks...
	return true
}

// getUserPurchaseCount get user purchase count
func (s *seckillService) getUserPurchaseCount(ctx context.Context, activityID, userID uint64) (int, error) {
	key := fmt.Sprintf("purchase_count:%d:%d", activityID, userID)
	count, err := s.redis.Get(ctx, key).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}
	return count, nil
}

// incrUserPurchaseCount increase user purchase count
func (s *seckillService) incrUserPurchaseCount(ctx context.Context, activityID, userID uint64, quantity int) {
	key := fmt.Sprintf("purchase_count:%d:%d", activityID, userID)
	s.redis.IncrBy(ctx, key, int64(quantity))
	s.redis.Expire(ctx, key, 7*24*time.Hour) // 7 days expiration
}

// recordSeckillLog record seckill log
func (s *seckillService) recordSeckillLog(ctx context.Context, req *SeckillRequest, deductID, status string) {
	logKey := fmt.Sprintf("seckill_log:%s", req.RequestID)
	logData := map[string]interface{}{
		"request_id":  req.RequestID,
		"activity_id": req.ActivityID,
		"user_id":     req.UserID,
		"quantity":    req.Quantity,
		"deduct_id":   deductID,
		"status":      status,
		"ip":          req.IP,
		"device_id":   req.DeviceID,
		"timestamp":   time.Now().Unix(),
	}

	data, _ := json.Marshal(logData)
	s.redis.SetEx(ctx, logKey, data, 7*24*time.Hour)
}

// failResult construct failure result
func (s *seckillService) failResult(requestID, message string) *SeckillResult {
	return &SeckillResult{
		Success:   false,
		RequestID: requestID,
		Message:   message,
	}
}

// recordCircuitBreakerSuccess record circuit breaker success
func (s *seckillService) recordCircuitBreakerSuccess(name string) {
	// Circuit breaker will record automatically in Execute method
}

// recordCircuitBreakerError record circuit breaker error
func (s *seckillService) recordCircuitBreakerError(name string) {
	// Circuit breaker will record automatically in Execute method
}

// PrewarmActivity prewarm activity
func (s *seckillService) PrewarmActivity(ctx context.Context, activityID uint64) error {
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Start prewarming activity")

	// 1. Query activity information
	activity, err := s.activityRepo.GetByIDWithGoods(ctx, int64(activityID))
	if err != nil {
		return err
	}
	
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
		"stock":       activity.Stock,
		"name":        activity.Name,
	}).Info("Activity loaded from database")

	// 2. Sync stock to Redis
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
		"stock":       activity.Stock,
	}).Info("Syncing stock to Redis")
	if err := s.inventory.SyncToRedis(ctx, activityID, activity.Stock); err != nil {
		return err
	}

	// 3. Pre-load activity config to cache
	configKey := fmt.Sprintf("activity:config:%d", activityID)
	configData, _ := json.Marshal(activity)
	s.redis.SetEx(ctx, configKey, configData, 24*time.Hour)

	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Activity prewarmed successfully")
	return nil
}

// QuerySeckillResult query seckill result
func (s *seckillService) QuerySeckillResult(ctx context.Context, requestID string, userID uint64) (*SeckillResult, error) {
	resultKey := fmt.Sprintf("seckill:result:%s:%d", requestID, userID)
	data, err := s.redis.Get(ctx, resultKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("seckill result not found")
		}
		return nil, err
	}

	var result SeckillResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// OrderMessage has been moved to internal/model/message.go

