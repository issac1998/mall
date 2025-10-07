package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"seckill/internal/model"
	"seckill/pkg/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderService mock order service
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, msg *model.OrderMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockOrderService) ConsumeOrderMessage(ctx context.Context, messageData []byte) error {
	args := m.Called(ctx, messageData)
	return args.Error(0)
}

func (m *MockOrderService) HandleExpiredOrders(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockOrderService) PayOrder(ctx context.Context, orderNo string) error {
	args := m.Called(ctx, orderNo)
	return args.Error(0)
}

func (m *MockOrderService) GetOrderByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	args := m.Called(ctx, orderNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderService) ListUserOrders(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Order, int64, error) {
	args := m.Called(ctx, userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.Order), args.Get(1).(int64), args.Error(2)
}

func TestVIPPriorityConsumer(t *testing.T) {
	// Create mock service and queue
	mockService := new(MockOrderService)
	mq, _ := queue.NewMemoryQueue(nil)
	defer mq.Close()

	// Create consumer with 1 VIP worker and 1 normal worker for testing
	consumer := NewVIPPriorityConsumer(mockService, mq, 1, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Setup mock expectations
	mockService.On("ConsumeOrderMessage", mock.Anything, mock.Anything).Return(nil)

	// Start consumer
	consumer.Start(ctx)

	// Send VIP message
	vipMsg := &model.OrderMessage{
		RequestID:  "vip-001",
		UserID:     1,
		ActivityID: 100,
		IsVIP:      true,
	}
	vipData, _ := json.Marshal(vipMsg)
	err := mq.Publish(ctx, "seckill_orders_vip", vipData)
	assert.NoError(t, err)

	// Send normal message
	normalMsg := &model.OrderMessage{
		RequestID:  "normal-001",
		UserID:     2,
		ActivityID: 100,
		IsVIP:      false,
	}
	normalData, _ := json.Marshal(normalMsg)
	err = mq.Publish(ctx, "seckill_orders", normalData)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Stop consumer
	consumer.Stop()

	// Verify messages were processed
	mockService.AssertCalled(t, "ConsumeOrderMessage", mock.Anything, vipData)
	mockService.AssertCalled(t, "ConsumeOrderMessage", mock.Anything, normalData)
}

func TestVIPPriorityConsumer_VIPFirst(t *testing.T) {
	// Create mock service and queue
	mockService := new(MockOrderService)
	mq, _ := queue.NewMemoryQueue(nil)
	defer mq.Close()

	processedOrder := make([]string, 0)
	
	// Mock to track processing order
	mockService.On("ConsumeOrderMessage", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		data := args.Get(1).([]byte)
		var msg model.OrderMessage
		json.Unmarshal(data, &msg)
		processedOrder = append(processedOrder, msg.RequestID)
	}).Return(nil)

	// Create consumer with 1 priority worker
	consumer := NewVIPPriorityConsumer(mockService, mq, 0, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Send normal message first
	normalMsg := &model.OrderMessage{RequestID: "normal-001"}
	normalData, _ := json.Marshal(normalMsg)
	mq.Publish(ctx, "seckill_orders", normalData)

	// Send VIP message
	vipMsg := &model.OrderMessage{RequestID: "vip-001"}
	vipData, _ := json.Marshal(vipMsg)
	mq.Publish(ctx, "seckill_orders_vip", vipData)

	// Start consumer
	consumer.Start(ctx)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Stop consumer
	consumer.Stop()

	// VIP should be processed first even though normal was sent first
	assert.Equal(t, 2, len(processedOrder))
	assert.Equal(t, "vip-001", processedOrder[0], "VIP message should be processed first")
}

