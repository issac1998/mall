package stock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"seckill/internal/repository"
	"seckill/internal/service/seckill"
	"seckill/pkg/log"
)

// StockService stock service interface
type StockService interface {
	// Sync stock to Redis from MySQL
	SyncStockToRedis(ctx context.Context, activityID uint64) error

	// Sync stock to MySQL from Redis
	SyncStockToMySQL(ctx context.Context, activityID uint64) error

	// Check stock consistency between Redis and MySQL
	CheckStockConsistency(ctx context.Context, activityID uint64) (*ConsistencyReport, error)

	// Repair stock inconsistency
	RepairStockInconsistency(ctx context.Context, activityID uint64) error

	// Start periodic sync task
	StartPeriodicSync(ctx context.Context, interval time.Duration)
}

// stockService stock service implementation
type stockService struct {
	activityRepo repository.ActivityRepository
	goodsRepo    repository.GoodsRepository
	inventory    *seckill.MultiLevelInventory
	redis        *redis.Client
}

// NewStockService creates a stock service
func NewStockService(
	activityRepo repository.ActivityRepository,
	goodsRepo repository.GoodsRepository,
	inventory *seckill.MultiLevelInventory,
	redis *redis.Client,
) StockService {
	return &stockService{
		activityRepo: activityRepo,
		goodsRepo:    goodsRepo,
		inventory:    inventory,
		redis:        redis,
	}
}

// ConsistencyReport stock consistency report
type ConsistencyReport struct {
	ActivityID    uint64    `json:"activity_id"`
	RedisStock    int       `json:"redis_stock"`
	MySQLStock    int       `json:"mysql_stock"`
	ReservedStock int       `json:"reserved_stock"`
	Difference    int       `json:"difference"`
	IsConsistent  bool      `json:"is_consistent"`
	CheckTime     time.Time `json:"check_time"`
}

// SyncStockToRedis sync stock from MySQL to Redis
func (s *stockService) SyncStockToRedis(ctx context.Context, activityID uint64) error {
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Start syncing stock to Redis")

	// Get activity from MySQL
	activity, err := s.activityRepo.GetByID(ctx, int64(activityID))
	if err != nil {
		return fmt.Errorf("failed to get activity: %w", err)
	}

	// Calculate available stock (total - sold)
	availableStock := activity.Stock - activity.Sold
	if availableStock < 0 {
		availableStock = 0
	}

	// Sync to Redis
	if err := s.inventory.SyncToRedis(ctx, activityID, availableStock); err != nil {
		return fmt.Errorf("failed to sync to Redis: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
		"stock":       availableStock,
	}).Info("Stock synced to Redis successfully")

	return nil
}

// SyncStockToMySQL sync stock from Redis to MySQL
func (s *stockService) SyncStockToMySQL(ctx context.Context, activityID uint64) error {
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Start syncing stock to MySQL")

	// Get stock from Redis
	redisStock, err := s.inventory.GetStockFromRedis(ctx, activityID)
	if err != nil {
		return fmt.Errorf("failed to get stock from Redis: %w", err)
	}

	// Get reserved stock
	reservedKey := fmt.Sprintf("stock:reserved:%d", activityID)
	reservedStock, _ := s.redis.Get(ctx, reservedKey).Int()

	// Get activity from MySQL
	activity, err := s.activityRepo.GetByID(ctx, int64(activityID))
	if err != nil {
		return fmt.Errorf("failed to get activity: %w", err)
	}

	// Calculate sold quantity
	// Total stock = available stock + reserved stock + sold
	// So: sold = total stock - available stock - reserved stock
	totalStock := activity.Stock
	sold := totalStock - redisStock - reservedStock

	// Update sold quantity in MySQL
	activity.Sold = sold
	if err := s.activityRepo.Update(ctx, activity); err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"activity_id":    activityID,
		"redis_stock":    redisStock,
		"reserved_stock": reservedStock,
		"sold":           sold,
	}).Info("Stock synced to MySQL successfully")

	return nil
}

// CheckStockConsistency check stock consistency between Redis and MySQL
func (s *stockService) CheckStockConsistency(ctx context.Context, activityID uint64) (*ConsistencyReport, error) {
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Start checking stock consistency")

	// Get stock from Redis
	redisStock, err := s.inventory.GetStockFromRedis(ctx, activityID)
	if err != nil {
		redisStock = 0 // If Redis doesn't have the data, treat as 0
	}

	// Get reserved stock from Redis
	reservedKey := fmt.Sprintf("stock:reserved:%d", activityID)
	reservedStock, _ := s.redis.Get(ctx, reservedKey).Int()

	// Get activity from MySQL
	activity, err := s.activityRepo.GetByID(ctx, int64(activityID))
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}

	// Calculate MySQL available stock
	mysqlStock := activity.Stock - activity.Sold

	// Calculate difference
	// Expected: Redis stock + reserved stock should equal MySQL available stock
	expectedTotal := redisStock + reservedStock
	difference := mysqlStock - expectedTotal

	report := &ConsistencyReport{
		ActivityID:    activityID,
		RedisStock:    redisStock,
		MySQLStock:    mysqlStock,
		ReservedStock: reservedStock,
		Difference:    difference,
		IsConsistent:  difference == 0,
		CheckTime:     time.Now(),
	}

	if !report.IsConsistent {
		log.WithFields(map[string]interface{}{
			"activity_id":    activityID,
			"redis_stock":    redisStock,
			"mysql_stock":    mysqlStock,
			"reserved_stock": reservedStock,
			"difference":     difference,
		}).Warn("Stock inconsistency detected")
	} else {
		log.WithFields(map[string]interface{}{
			"activity_id": activityID,
		}).Info("Stock is consistent")
	}

	return report, nil
}

// RepairStockInconsistency repair stock inconsistency
func (s *stockService) RepairStockInconsistency(ctx context.Context, activityID uint64) error {
	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
	}).Info("Start repairing stock inconsistency")

	// Check consistency first
	report, err := s.CheckStockConsistency(ctx, activityID)
	if err != nil {
		return err
	}

	if report.IsConsistent {
		log.WithFields(map[string]interface{}{
			"activity_id": activityID,
		}).Info("Stock is already consistent, no repair needed")
		return nil
	}

	// Strategy: Use MySQL as the source of truth
	// Sync MySQL stock to Redis
	if err := s.SyncStockToRedis(ctx, activityID); err != nil {
		return fmt.Errorf("failed to repair: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"activity_id": activityID,
		"difference":  report.Difference,
	}).Info("Stock inconsistency repaired successfully")

	return nil
}

// StartPeriodicSync start periodic sync task
func (s *stockService) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.WithFields(map[string]interface{}{
		"interval": interval,
	}).Info("Started periodic stock sync task")

	for {
		select {
		case <-ctx.Done():
			log.Info("Periodic stock sync task stopped")
			return
		case <-ticker.C:
			s.performPeriodicSync(ctx)
		}
	}
}

// performPeriodicSync perform periodic sync
func (s *stockService) performPeriodicSync(ctx context.Context) {
	log.Info("Performing periodic stock sync")

	// Get all active activities
	activities, _, err := s.activityRepo.ListActive(ctx, 1, 100)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to list active activities")
		return
	}

	for _, activity := range activities {
		activityID := activity.ID

		// Check consistency
		report, err := s.CheckStockConsistency(ctx, uint64(activityID))
		if err != nil {
			log.WithFields(map[string]interface{}{
				"activity_id": activityID,
				"error":       err.Error(),
			}).Error("Failed to check stock consistency")
			continue
		}

		// Repair if inconsistent
		if !report.IsConsistent {
			if err := s.RepairStockInconsistency(ctx, uint64(activityID)); err != nil {
				log.WithFields(map[string]interface{}{
					"activity_id": activityID,
					"error":       err.Error(),
				}).Error("Failed to repair stock inconsistency")
			}
		}
	}

	log.Info("Periodic stock sync completed")
}

