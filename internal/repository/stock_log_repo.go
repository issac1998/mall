package repository

import (
	"context"

	"gorm.io/gorm"
	"seckill/internal/model"
)

// StockLogRepository stock log repository interface
type StockLogRepository interface {
	// Create creates a stock log
	Create(ctx context.Context, log *model.StockLog) error

	// GetByActivityID gets stock logs by activity ID
	GetByActivityID(ctx context.Context, activityID uint64, limit int) ([]model.StockLog, error)

	// GetByOrderNo gets stock log by order number
	GetByOrderNo(ctx context.Context, orderNo string) (*model.StockLog, error)

	// List lists stock logs with pagination
	List(ctx context.Context, page, pageSize int) ([]model.StockLog, int64, error)
}

// stockLogRepository stock log repository implementation
type stockLogRepository struct {
	db *gorm.DB
}

// NewStockLogRepository creates a stock log repository
func NewStockLogRepository(db *gorm.DB) StockLogRepository {
	return &stockLogRepository{
		db: db,
	}
}

// Create creates a stock log
func (r *stockLogRepository) Create(ctx context.Context, log *model.StockLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByActivityID gets stock logs by activity ID
func (r *stockLogRepository) GetByActivityID(ctx context.Context, activityID uint64, limit int) ([]model.StockLog, error) {
	var logs []model.StockLog
	err := r.db.WithContext(ctx).
		Where("activity_id = ?", activityID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetByOrderNo gets stock log by order number
func (r *stockLogRepository) GetByOrderNo(ctx context.Context, orderNo string) (*model.StockLog, error) {
	var log model.StockLog
	err := r.db.WithContext(ctx).
		Where("order_no = ?", orderNo).
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List lists stock logs with pagination
func (r *stockLogRepository) List(ctx context.Context, page, pageSize int) ([]model.StockLog, int64, error) {
	var logs []model.StockLog
	var total int64

	offset := (page - 1) * pageSize

	if err := r.db.WithContext(ctx).Model(&model.StockLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

