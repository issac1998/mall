package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"seckill/internal/model"
)

// OrderRepository order repository interface
type OrderRepository interface {
	// Create order
	Create(ctx context.Context, order *model.Order) error

	// Get order by ID
	GetByID(ctx context.Context, id uint64) (*model.Order, error)

	// Get order by order number
	GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)

	// Get order by request ID (for idempotency)
	GetByRequestID(ctx context.Context, requestID string) (*model.Order, error)

	// Update order status
	UpdateStatus(ctx context.Context, id uint64, status int8) error

	// List user orders
	ListUserOrders(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Order, int64, error)

	// List expired orders
	ListExpiredOrders(ctx context.Context, limit int) ([]*model.Order, error)
}

// orderRepository order repository implementation
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates an order repository
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// Create creates an order
func (r *orderRepository) Create(ctx context.Context, order *model.Order) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create order
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// Create order details
		if len(order.Details) > 0 {
			for i := range order.Details {
				order.Details[i].OrderID = order.ID
				order.Details[i].OrderNo = order.OrderNo
			}
			if err := tx.Create(&order.Details).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetByID gets an order by ID
func (r *orderRepository) GetByID(ctx context.Context, id uint64) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("Details").
		Preload("User").
		Preload("Activity").
		Preload("Goods").
		Where("id = ?", id).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

// GetByOrderNo gets an order by order number
func (r *orderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("Details").
		Where("order_no = ?", orderNo).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

// GetByRequestID gets an order by request ID (for idempotency)
func (r *orderRepository) GetByRequestID(ctx context.Context, requestID string) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil for not found (not an error for idempotency check)
		}
		return nil, err
	}
	return &order, nil
}

// UpdateStatus updates order status
func (r *orderRepository) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Set timestamp based on status
	now := time.Now()
	switch status {
	case model.OrderStatusPaid:
		updates["paid_at"] = &now
	case model.OrderStatusCancelled, model.OrderStatusRefunded:
		// Keep original cancel/refund time if already set
	}

	return r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ListUserOrders lists user orders
func (r *orderRepository) ListUserOrders(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	offset := (page - 1) * pageSize

	db := r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("user_id = ?", userID)

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get list
	err := db.Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Preload("Details").
		Find(&orders).Error

	return orders, total, err
}

// ListExpiredOrders lists expired orders
func (r *orderRepository) ListExpiredOrders(ctx context.Context, limit int) ([]*model.Order, error) {
	var orders []*model.Order

	err := r.db.WithContext(ctx).
		Where("status = ?", model.OrderStatusPending).
		Where("expire_at < ?", time.Now()).
		Limit(limit).
		Find(&orders).Error

	return orders, err
}

