package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"seckill/internal/service/stock"
)

// MockStockService mock stock service
type MockStockService struct {
	mock.Mock
}

func (m *MockStockService) SyncStockToRedis(ctx context.Context, activityID uint64) error {
	args := m.Called(ctx, activityID)
	return args.Error(0)
}

func (m *MockStockService) SyncStockToMySQL(ctx context.Context, activityID uint64) error {
	args := m.Called(ctx, activityID)
	return args.Error(0)
}

func (m *MockStockService) CheckStockConsistency(ctx context.Context, activityID uint64) (*stock.ConsistencyReport, error) {
	args := m.Called(ctx, activityID)
	return args.Get(0).(*stock.ConsistencyReport), args.Error(1)
}

func (m *MockStockService) RepairStockInconsistency(ctx context.Context, activityID uint64) error {
	args := m.Called(ctx, activityID)
	return args.Error(0)
}

func (m *MockStockService) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	m.Called(ctx, interval)
}

func TestStockHandler_SyncToRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful sync to redis", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/sync/redis/:id", handler.SyncToRedis)

		mockService.On("SyncStockToRedis", mock.Anything, uint64(123)).Return(nil)

		req, _ := http.NewRequest("POST", "/stock/sync/redis/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Stock synced to Redis successfully", data["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid activity id", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/sync/redis/:id", handler.SyncToRedis)

		req, _ := http.NewRequest("POST", "/stock/sync/redis/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid activity ID", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/sync/redis/:id", handler.SyncToRedis)

		mockService.On("SyncStockToRedis", mock.Anything, uint64(123)).Return(assert.AnError)

		req, _ := http.NewRequest("POST", "/stock/sync/redis/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "assert.AnError")

		mockService.AssertExpectations(t)
	})
}

func TestStockHandler_SyncToMySQL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful sync to mysql", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/sync/mysql/:id", handler.SyncToMySQL)

		mockService.On("SyncStockToMySQL", mock.Anything, uint64(123)).Return(nil)

		req, _ := http.NewRequest("POST", "/stock/sync/mysql/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Stock synced to MySQL successfully", data["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid activity id", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/sync/mysql/:id", handler.SyncToMySQL)

		req, _ := http.NewRequest("POST", "/stock/sync/mysql/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid activity ID", response["message"])

		mockService.AssertExpectations(t)
	})
}

func TestStockHandler_CheckConsistency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful consistency check", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.GET("/stock/consistency/:id", handler.CheckConsistency)

		expectedReport := &stock.ConsistencyReport{
			ActivityID:    123,
			RedisStock:    100,
			MySQLStock:    100,
			ReservedStock: 0,
			Difference:    0,
			IsConsistent:  true,
			CheckTime:     time.Now(),
		}

		mockService.On("CheckStockConsistency", mock.Anything, uint64(123)).Return(expectedReport, nil)

		req, _ := http.NewRequest("GET", "/stock/consistency/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(123), data["activity_id"])
		assert.Equal(t, float64(100), data["redis_stock"])
		assert.Equal(t, float64(100), data["mysql_stock"])
		assert.Equal(t, true, data["is_consistent"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid activity id", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.GET("/stock/consistency/:id", handler.CheckConsistency)

		req, _ := http.NewRequest("GET", "/stock/consistency/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid activity ID", response["message"])

		mockService.AssertExpectations(t)
	})
}

func TestStockHandler_RepairInconsistency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful repair", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/repair/:id", handler.RepairInconsistency)

		mockService.On("RepairStockInconsistency", mock.Anything, uint64(123)).Return(nil)

		req, _ := http.NewRequest("POST", "/stock/repair/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Stock inconsistency repaired successfully", data["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid activity id", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/repair/:id", handler.RepairInconsistency)

		req, _ := http.NewRequest("POST", "/stock/repair/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid activity ID", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStockService)
		handler := NewStockHandler(mockService)

		router := gin.New()
		router.POST("/stock/repair/:id", handler.RepairInconsistency)

		mockService.On("RepairStockInconsistency", mock.Anything, uint64(123)).Return(assert.AnError)

		req, _ := http.NewRequest("POST", "/stock/repair/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "assert.AnError")

		mockService.AssertExpectations(t)
	})
}