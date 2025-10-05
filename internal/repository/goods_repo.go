package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"seckill/internal/model"
)

// GoodsRepository goods repository interface
type GoodsRepository interface {
	// Create goods
	Create(ctx context.Context, goods *model.Goods) error

	// Get goods by ID
	GetByID(ctx context.Context, id uint64) (*model.Goods, error)

	// Update goods
	Update(ctx context.Context, goods *model.Goods) error

	// Update goods stock
	UpdateStock(ctx context.Context, id uint64, quantity int) error

	// Decrement stock (atomic operation)
	DecrStock(ctx context.Context, id uint64, quantity int) error

	// Increment stock
	IncrStock(ctx context.Context, id uint64, quantity int) error

	// List goods
	List(ctx context.Context, page, pageSize int, status int8) ([]*model.Goods, int64, error)
}

// goodsRepository goods repository implementation
type goodsRepository struct {
	db *gorm.DB
}

// NewGoodsRepository creates a goods repository
func NewGoodsRepository(db *gorm.DB) GoodsRepository {
	return &goodsRepository{db: db}
}

// Create creates goods
func (r *goodsRepository) Create(ctx context.Context, goods *model.Goods) error {
	return r.db.WithContext(ctx).Create(goods).Error
}

// GetByID gets goods by ID
func (r *goodsRepository) GetByID(ctx context.Context, id uint64) (*model.Goods, error) {
	var goods model.Goods
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&goods).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("goods not found")
		}
		return nil, err
	}
	return &goods, nil
}

// Update updates goods
func (r *goodsRepository) Update(ctx context.Context, goods *model.Goods) error {
	return r.db.WithContext(ctx).Save(goods).Error
}

// UpdateStock updates goods stock
func (r *goodsRepository) UpdateStock(ctx context.Context, id uint64, quantity int) error {
	return r.db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ?", id).
		Update("stock", quantity).Error
}

// DecrStock decrements stock (atomic operation)
func (r *goodsRepository) DecrStock(ctx context.Context, id uint64, quantity int) error {
	result := r.db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ? AND stock >= ?", id, quantity).
		Updates(map[string]interface{}{
			"stock": gorm.Expr("stock - ?", quantity),
			"sales": gorm.Expr("sales + ?", quantity),
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
func (r *goodsRepository) IncrStock(ctx context.Context, id uint64, quantity int) error {
	return r.db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"stock": gorm.Expr("stock + ?", quantity),
			"sales": gorm.Expr("sales - ?", quantity),
		}).Error
}

// List lists goods
func (r *goodsRepository) List(ctx context.Context, page, pageSize int, status int8) ([]*model.Goods, int64, error) {
	var goods []*model.Goods
	var total int64

	offset := (page - 1) * pageSize

	db := r.db.WithContext(ctx).Model(&model.Goods{})

	if status > 0 {
		db = db.Where("status = ?", status)
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get list
	err := db.Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&goods).Error

	return goods, total, err
}

