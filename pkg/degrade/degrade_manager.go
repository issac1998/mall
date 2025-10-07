package degrade

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DegradeManager manages service degradation
type DegradeManager struct {
	redis *redis.Client
}

// NewDegradeManager creates a new degrade manager
func NewDegradeManager(redis *redis.Client) *DegradeManager {
	return &DegradeManager{
		redis: redis,
	}
}

// DegradeStrategy degradation strategy
type DegradeStrategy struct {
	Type        string `json:"type"`         // queue_only/return_error/lottery
	EstWaitTime int    `json:"est_wait_time"` // Estimated wait time (seconds)
	Ratio       float64 `json:"ratio"`        // Degradation ratio (0-1)
	Message     string `json:"message"`       // Message to return
}

// IsDegrade checks if service is degraded
func (dm *DegradeManager) IsDegrade(ctx context.Context, activityID uint64) bool {
	key := fmt.Sprintf("degrade:status:%d", activityID)
	
	val, err := dm.redis.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	
	return val == "1"
}

// GetStrategy gets degradation strategy
func (dm *DegradeManager) GetStrategy(ctx context.Context, activityID uint64) *DegradeStrategy {
	key := fmt.Sprintf("degrade:strategy:%d", activityID)
	
	data, err := dm.redis.Get(ctx, key).Bytes()
	if err != nil {
		return &DegradeStrategy{
			Type:    "return_error",
			Message: "System busy, please try again later",
		}
	}
	
	var strategy DegradeStrategy
	if err := json.Unmarshal(data, &strategy); err != nil {
		return &DegradeStrategy{
			Type:    "return_error",
			Message: "System busy, please try again later",
		}
	}
	
	return &strategy
}

// EnableDegrade enables degradation
func (dm *DegradeManager) EnableDegrade(ctx context.Context, activityID uint64, strategy *DegradeStrategy) error {
	statusKey := fmt.Sprintf("degrade:status:%d", activityID)
	strategyKey := fmt.Sprintf("degrade:strategy:%d", activityID)
	
	// Set degradation status
	if err := dm.redis.Set(ctx, statusKey, "1", 0).Err(); err != nil {
		return fmt.Errorf("failed to set degrade status: %w", err)
	}
	
	// Set degradation strategy
	data, err := json.Marshal(strategy)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy: %w", err)
	}
	
	if err := dm.redis.Set(ctx, strategyKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to set degrade strategy: %w", err)
	}
	
	return nil
}

// DisableDegrade disables degradation
func (dm *DegradeManager) DisableDegrade(ctx context.Context, activityID uint64) error {
	statusKey := fmt.Sprintf("degrade:status:%d", activityID)
	strategyKey := fmt.Sprintf("degrade:strategy:%d", activityID)
	
	if err := dm.redis.Del(ctx, statusKey, strategyKey).Err(); err != nil {
		return fmt.Errorf("failed to disable degrade: %w", err)
	}
	
	return nil
}

// SetTemporaryDegrade sets temporary degradation with TTL
func (dm *DegradeManager) SetTemporaryDegrade(ctx context.Context, activityID uint64, strategy *DegradeStrategy, ttl time.Duration) error {
	statusKey := fmt.Sprintf("degrade:status:%d", activityID)
	strategyKey := fmt.Sprintf("degrade:strategy:%d", activityID)
	
	// Set degradation status with TTL
	if err := dm.redis.Set(ctx, statusKey, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to set temporary degrade status: %w", err)
	}
	
	// Set degradation strategy with TTL
	data, err := json.Marshal(strategy)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy: %w", err)
	}
	
	if err := dm.redis.Set(ctx, strategyKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set temporary degrade strategy: %w", err)
	}
	
	return nil
}

// GetDegradeStatus gets degradation status for all activities
func (dm *DegradeManager) GetDegradeStatus(ctx context.Context) (map[uint64]*DegradeStrategy, error) {
	result := make(map[uint64]*DegradeStrategy)
	
	// Scan for all degrade status keys
	iter := dm.redis.Scan(ctx, 0, "degrade:status:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		
		// Extract activity ID from key
		var activityID uint64
		if _, err := fmt.Sscanf(key, "degrade:status:%d", &activityID); err != nil {
			continue
		}
		
		// Get strategy
		strategy := dm.GetStrategy(ctx, activityID)
		result[activityID] = strategy
	}
	
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan degrade keys: %w", err)
	}
	
	return result, nil
}

