package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
	"seckill/internal/model"
)

// ActivityRepository activity repository interface
type ActivityRepository interface {
	// Create activity
	Create(ctx context.Context, activity *model.SeckillActivity) error

	// Get activity by ID
	GetByID(ctx context.Context, id int64) (*model.SeckillActivity, error)

	// Get activity by ID with goods info
	GetByIDWithGoods(ctx context.Context, id int64) (*model.SeckillActivity, error)

	// Update activity
	Update(ctx context.Context, activity *model.SeckillActivity) error

	// Update activity status
	UpdateStatus(ctx context.Context, id int64, status int8) error

	// List active activities
	ListActive(ctx context.Context, page, pageSize int) ([]*model.SeckillActivity, int64, error)

	// List upcoming activities
	ListUpcoming(ctx context.Context, limit int) ([]*model.SeckillActivity, error)

	// Decrement stock (atomic operation)
	DecrStock(ctx context.Context, id int64, quantity int) error

	// Increment stock
	IncrStock(ctx context.Context, id int64, quantity int) error
}

// activityRepository activity repository implementation
type activityRepository struct {
	db *gorm.DB
}

// NewActivityRepository creates an activity repository
func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

// Create creates an activity
func (r *activityRepository) Create(ctx context.Context, activity *model.SeckillActivity) error {
	return r.db.WithContext(ctx).Create(activity).Error
}

// GetByID gets an activity by ID
func (r *activityRepository) GetByID(ctx context.Context, id int64) (*model.SeckillActivity, error) {
	var activity model.SeckillActivity
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&activity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("activity not found")
		}
		return nil, err
	}
	return &activity, nil
}

// GetByIDWithGoods gets an activity with goods info
func (r *activityRepository) GetByIDWithGoods(ctx context.Context, id int64) (*model.SeckillActivity, error) {
	var activity model.SeckillActivity
	err := r.db.WithContext(ctx).
		Preload("Goods").
		Where("id = ?", id).
		First(&activity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("activity not found")
		}
		return nil, err
	}
	
	log.Printf("DEBUG: Activity loaded from DB - ID: %d, Stock: %d, Price: %.2f, GoodsID: %d, LimitPerUser: %d", 
		activity.ID, activity.Stock, activity.Price, activity.GoodsID, activity.LimitPerUser)
	
	return &activity, nil
}

// Update updates an activity
func (r *activityRepository) Update(ctx context.Context, activity *model.SeckillActivity) error {
	return r.db.WithContext(ctx).Save(activity).Error
}

// UpdateStatus updates activity status
func (r *activityRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).
		Model(&model.SeckillActivity{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// ListActive lists active activities
func (r *activityRepository) ListActive(ctx context.Context, page, pageSize int) ([]*model.SeckillActivity, int64, error) {
	var activities []*model.SeckillActivity
	var total int64

	offset := (page - 1) * pageSize

	db := r.db.WithContext(ctx).
		Model(&model.SeckillActivity{}).
		Where("status = ?", 1).
		Where("start_time <= ?", time.Now()).
		Where("end_time >= ?", time.Now())

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get list
	err := db.Offset(offset).
		Limit(pageSize).
		Order("start_time ASC").
		Find(&activities).Error

	return activities, total, err
}

// ListUpcoming lists upcoming activities
func (r *activityRepository) ListUpcoming(ctx context.Context, limit int) ([]*model.SeckillActivity, error) {
	var activities []*model.SeckillActivity

	err := r.db.WithContext(ctx).
		Where("status = ?", 0).
		Where("start_time > ?", time.Now()).
		Order("start_time ASC").
		Limit(limit).
		Find(&activities).Error

	return activities, err
}

// DecrStock decrements stock (atomic operation)
func (r *activityRepository) DecrStock(ctx context.Context, id int64, quantity int) error {
	result := r.db.WithContext(ctx).
		Model(&model.SeckillActivity{}).
		Where("id = ? AND stock >= ?", id, quantity).
		Updates(map[string]interface{}{
			"stock": gorm.Expr("stock - ?", quantity),
			"sold":  gorm.Expr("sold + ?", quantity),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("insufficient stock")
	}

	return nil
}

// IncrStock increments stock
func (r *activityRepository) IncrStock(ctx context.Context, id int64, quantity int) error {
	return r.db.WithContext(ctx).
		Model(&model.SeckillActivity{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"stock": gorm.Expr("stock + ?", quantity),
			"sold":  gorm.Expr("sold - ?", quantity),
		}).Error
}

