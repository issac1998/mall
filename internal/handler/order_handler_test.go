package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"seckill/internal/model"
	"seckill/internal/service/seckill"
)

// MockOrderService is a mock implementation of order.OrderService
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, msg *seckill.OrderMessage) error {
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
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.Order), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrderService) PayOrder(ctx context.Context, orderNo string) error {
	args := m.Called(ctx, orderNo)
	return args.Error(0)
}

func TestOrderHandler_GetOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful get order", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		expectedOrder := &model.Order{
			ID:       1,
			OrderNo:  "ORDER123",
			UserID:   1,
			GoodsID:  1,
			Quantity: 2,
			Status:   1,
		}

		mockService.On("GetOrderByOrderNo", mock.Anything, "ORDER123").Return(expectedOrder, nil)

		router := gin.New()
		router.GET("/orders/:order_no", handler.GetOrder)

		req, _ := http.NewRequest("GET", "/orders/ORDER123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		mockService.On("GetOrderByOrderNo", mock.Anything, "NOTFOUND").Return(nil, errors.New("order not found"))

		router := gin.New()
		router.GET("/orders/:order_no", handler.GetOrder)

		req, _ := http.NewRequest("GET", "/orders/NOTFOUND", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "order not found")

		mockService.AssertExpectations(t)
	})
}

func TestOrderHandler_ListOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful list orders", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		expectedOrders := []*model.Order{
			{ID: 1, OrderNo: "ORDER123", UserID: 1, GoodsID: 1, Quantity: 2, Status: 1},
			{ID: 2, OrderNo: "ORDER124", UserID: 1, GoodsID: 2, Quantity: 1, Status: 1},
		}

		mockService.On("ListUserOrders", mock.Anything, uint64(1), 1, 10).Return(expectedOrders, int64(2), nil)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", int64(1))
			c.Next()
		})
		router.GET("/orders", handler.ListOrders)

		req, _ := http.NewRequest("GET", "/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["total"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["size"])

		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized - no user_id", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		router := gin.New()
		router.GET("/orders", handler.ListOrders)

		req, _ := http.NewRequest("GET", "/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", response["message"])
	})

	t.Run("with custom pagination", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		expectedOrders := []*model.Order{
			{ID: 3, OrderNo: "ORDER125", UserID: 1, GoodsID: 3, Quantity: 1, Status: 1},
		}

		mockService.On("ListUserOrders", mock.Anything, uint64(1), 2, 5).Return(expectedOrders, int64(6), nil)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", int64(1))
			c.Next()
		})
		router.GET("/orders", handler.ListOrders)

		req, _ := http.NewRequest("GET", "/orders?page=2&page_size=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(6), data["total"])
		assert.Equal(t, float64(2), data["page"])
		assert.Equal(t, float64(5), data["size"])

		mockService.AssertExpectations(t)
	})
}

func TestOrderHandler_PayOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful payment", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		mockService.On("PayOrder", mock.Anything, "ORDER123").Return(nil)

		router := gin.New()
		router.POST("/orders/:order_no/pay", handler.PayOrder)

		req, _ := http.NewRequest("POST", "/orders/ORDER123/pay", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])
		assert.Equal(t, "Payment successful", response["data"])

		mockService.AssertExpectations(t)
	})

	t.Run("payment failed", func(t *testing.T) {
		mockService := new(MockOrderService)
		handler := NewOrderHandler(mockService)

		mockService.On("PayOrder", mock.Anything, "ORDER123").Return(errors.New("insufficient balance"))

		router := gin.New()
		router.POST("/orders/:order_no/pay", handler.PayOrder)

		req, _ := http.NewRequest("POST", "/orders/ORDER123/pay", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "Payment failed")

		mockService.AssertExpectations(t)
	})
}