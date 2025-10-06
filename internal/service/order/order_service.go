package order

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"seckill/internal/model"
	"seckill/internal/repository"
	"seckill/internal/service/seckill"
	"seckill/pkg/log"
	"seckill/pkg/snowflake"
)

// OrderService order service interface
type OrderService interface {
	// Create order (synchronous)
	CreateOrder(ctx context.Context, msg *model.OrderMessage) error

	// Consume order message (asynchronous)
	ConsumeOrderMessage(ctx context.Context, messageData []byte) error

	// Handle expired orders
	HandleExpiredOrders(ctx context.Context) error

	// Pay order
	PayOrder(ctx context.Context, orderNo string) error

	// Get order by order number
	GetOrderByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)

	// List user orders
	ListUserOrders(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Order, int64, error)
}

// orderService order service implementation
type orderService struct {
	orderRepo   repository.OrderRepository
	goodsRepo   repository.GoodsRepository
	inventory   *seckill.MultiLevelInventory
	idGenerator *snowflake.IDGenerator
}

// NewOrderService creates an order service
func NewOrderService(
	orderRepo repository.OrderRepository,
	goodsRepo repository.GoodsRepository,
	inventory *seckill.MultiLevelInventory,
	idGenerator *snowflake.IDGenerator,
) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		goodsRepo:   goodsRepo,
		inventory:   inventory,
		idGenerator: idGenerator,
	}
}

// CreateOrder creates an order
func (s *orderService) CreateOrder(ctx context.Context, msg *model.OrderMessage) error {
	log.WithFields(map[string]interface{}{
		"request_id": msg.RequestID,
	}).Info("Start creating order")

	// 1. Check if order already exists (idempotency)
	existingOrder, err := s.orderRepo.GetByRequestID(ctx, msg.RequestID)
	if err != nil {
		return err
	}
	if existingOrder != nil {
		log.WithFields(map[string]interface{}{
			"order_no": existingOrder.OrderNo,
		}).Info("Order already exists")
		return nil
	}

	// 2. Generate order ID and order number
	orderID := uint64(s.idGenerator.NextID())
	orderNo := fmt.Sprintf("SK%d", orderID)

	// 3. Calculate order amount (convert to cents)
	priceInCents := int64(msg.Price * 100)
	totalAmount := priceInCents * int64(msg.Quantity)

	// 4. Construct order
	order := &model.Order{
		ID:             orderID,
		OrderNo:        orderNo,
		RequestID:      msg.RequestID,
		ActivityID:     msg.ActivityID,
		UserID:         msg.UserID,
		GoodsID:        msg.GoodsID,
		Quantity:       msg.Quantity,
		Price:          priceInCents,
		TotalAmount:    totalAmount,
		DiscountAmount: 0,
		PaymentAmount:  totalAmount,
		Status:         model.OrderStatusPending,
		DeductID:       msg.DeductID,                     // Store deduct ID for TCC
		ExpireAt:       time.Now().Add(15 * time.Minute), // 15 minutes payment timeout
		Details: []model.OrderDetail{
			{
				ID:        0, // 让数据库自动生成ID
				GoodsID:   msg.GoodsID,
				GoodsName: "Seckill Product",
				Price:     priceInCents,
				Quantity:  msg.Quantity,
				Amount:    totalAmount,
			},
		},
	}

	// 5. Save order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to create order")

		// Creation failed, cancel stock deduction
		s.inventory.CancelDeduct(ctx, msg.DeductID, msg.ActivityID)
		return err
	}

	// 6. Confirm stock deduction (TCC-Confirm phase)
	if err := s.inventory.ConfirmDeduct(ctx, msg.DeductID, msg.ActivityID); err != nil {
		log.WithFields(map[string]interface{}{
			"deduct_id": msg.DeductID,
			"error":     err.Error(),
		}).Error("Failed to confirm stock deduction")
		// TODO: Order is already created, but stock confirmation failed
		// This should be handled by a compensation mechanism
	}

	log.WithFields(map[string]interface{}{
		"order_no":  orderNo,
		"user_id":   msg.UserID,
		"amount":    totalAmount,
		"deduct_id": msg.DeductID,
	}).Info("Order created and stock confirmed successfully")

	return nil
}

// ConsumeOrderMessage consumes order message
func (s *orderService) ConsumeOrderMessage(ctx context.Context, messageData []byte) error {
	var msg model.OrderMessage
	if err := json.Unmarshal(messageData, &msg); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to parse order message")
		return err
	}

	return s.CreateOrder(ctx, &msg)
}

// HandleExpiredOrders handles expired orders
func (s *orderService) HandleExpiredOrders(ctx context.Context) error {
	// Query expired orders (process 100 at a time)
	orders, err := s.orderRepo.ListExpiredOrders(ctx, 100)
	if err != nil {
		return err
	}

	log.WithFields(map[string]interface{}{
		"count": len(orders),
	}).Info("Found expired orders")

	for _, order := range orders {
		// Update order status to timeout cancelled
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusCancelled); err != nil {
			log.WithFields(map[string]interface{}{
				"order_id": order.ID,
				"error":    err.Error(),
			}).Error("Failed to update order status")
			continue
		}

		// Rollback stock (TCC-Cancel)
		if order.DeductID != "" {
			if err := s.inventory.CancelDeduct(ctx, order.DeductID, order.ActivityID); err != nil {
				log.WithFields(map[string]interface{}{
					"order_id":  order.ID,
					"deduct_id": order.DeductID,
					"error":     err.Error(),
				}).Error("Failed to rollback stock")
			}
		}

		log.WithFields(map[string]interface{}{
			"order_no": order.OrderNo,
		}).Info("Expired order processed")
	}

	return nil
}

// PayOrder pays an order
func (s *orderService) PayOrder(ctx context.Context, orderNo string) error {
	order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return err
	}

	if order.Status != model.OrderStatusPending {
		return fmt.Errorf("invalid order status")
	}

	// Confirm stock deduction (TCC-Confirm)
	if order.DeductID != "" {
		if err := s.inventory.ConfirmDeduct(ctx, order.DeductID, order.ActivityID); err != nil {
			log.WithFields(map[string]interface{}{
				"order_id":  order.ID,
				"deduct_id": order.DeductID,
				"error":     err.Error(),
			}).Error("Failed to confirm stock deduction")
			return fmt.Errorf("failed to confirm stock: %w", err)
		}
	}

	// Update order status
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusPaid); err != nil {
		return err
	}

	log.WithFields(map[string]interface{}{
		"order_no": orderNo,
	}).Info("Order paid successfully")
	return nil
}

// GetOrderByOrderNo gets an order by order number
func (s *orderService) GetOrderByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	return s.orderRepo.GetByOrderNo(ctx, orderNo)
}

// ListUserOrders lists user orders
func (s *orderService) ListUserOrders(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Order, int64, error) {
	return s.orderRepo.ListUserOrders(ctx, userID, page, pageSize)
}
