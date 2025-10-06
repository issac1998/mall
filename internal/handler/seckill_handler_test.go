package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"seckill/internal/service/seckill"
)

// MockSeckillService mock seckill service
type MockSeckillService struct {
	mock.Mock
}

func (m *MockSeckillService) DoSeckill(ctx context.Context, req *seckill.SeckillRequest) (*seckill.SeckillResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*seckill.SeckillResult), args.Error(1)
}

func (m *MockSeckillService) PrewarmActivity(ctx context.Context, activityID uint64) error {
	args := m.Called(ctx, activityID)
	return args.Error(0)
}

func (m *MockSeckillService) QuerySeckillResult(ctx context.Context, requestID string, userID uint64) (*seckill.SeckillResult, error) {
	args := m.Called(ctx, requestID, userID)
	return args.Get(0).(*seckill.SeckillResult), args.Error(1)
}

func TestSeckillHandler_DoSeckill(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful seckill", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.POST("/seckill", func(c *gin.Context) {
			c.Set("user_id", int64(123))
			handler.DoSeckill(c)
		})

		expectedResult := &seckill.SeckillResult{
			Success:   true,
			RequestID: "req123",
			Message:   "Seckill successful",
		}

		mockService.On("DoSeckill", mock.Anything, mock.MatchedBy(func(req *seckill.SeckillRequest) bool {
			return req.ActivityID == 1 && req.UserID == 123 && req.RequestID == "req123" && req.Quantity == 1
		})).Return(expectedResult, nil)

		reqBody := map[string]interface{}{
			"request_id":  "req123",
			"activity_id": 1,
			"user_id":     123,
			"quantity":    1,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/seckill", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, true, data["success"])
		assert.Equal(t, "req123", data["request_id"])

		mockService.AssertExpectations(t)
	})

	t.Run("failed seckill", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.POST("/seckill", func(c *gin.Context) {
			c.Set("user_id", int64(123))
			handler.DoSeckill(c)
		})

		expectedResult := &seckill.SeckillResult{
			Success:   false,
			RequestID: "req123",
			Message:   "Inventory insufficient",
		}

		mockService.On("DoSeckill", mock.Anything, mock.MatchedBy(func(req *seckill.SeckillRequest) bool {
			return req.ActivityID == 1 && req.UserID == 123 && req.RequestID == "req123" && req.Quantity == 1
		})).Return(expectedResult, nil)

		reqBody := map[string]interface{}{
			"request_id":  "req123",
			"activity_id": 1,
			"user_id":     123,
			"quantity":    1,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/seckill", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Inventory insufficient", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, false, data["success"])
		assert.Equal(t, "Inventory insufficient", data["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized - no user_id", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.POST("/seckill", handler.DoSeckill)

		reqBody := map[string]interface{}{
			"request_id":  "req123",
			"activity_id": 1,
			"user_id":     123,
			"quantity":    1,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/seckill", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.POST("/seckill", func(c *gin.Context) {
			c.Set("user_id", int64(123))
			handler.DoSeckill(c)
		})

		req, _ := http.NewRequest("POST", "/seckill", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "Invalid parameters")

		mockService.AssertExpectations(t)
	})
}

func TestSeckillHandler_QueryResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful query", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.GET("/seckill/result", func(c *gin.Context) {
			c.Set("user_id", int64(123))
			handler.QueryResult(c)
		})

		expectedResult := &seckill.SeckillResult{
			Success:   true,
			RequestID: "req123",
			Message:   "Seckill successful",
		}

		mockService.On("QuerySeckillResult", mock.Anything, "req123", uint64(123)).Return(expectedResult, nil)

		req, _ := http.NewRequest("GET", "/seckill/result?request_id=req123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, true, data["success"])
		assert.Equal(t, "req123", data["request_id"])

		mockService.AssertExpectations(t)
	})

	t.Run("missing request_id", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.GET("/seckill/result", func(c *gin.Context) {
			c.Set("user_id", int64(123))
			handler.QueryResult(c)
		})

		req, _ := http.NewRequest("GET", "/seckill/result", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Missing request_id parameter", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized - no user_id", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.GET("/seckill/result", handler.QueryResult)

		req, _ := http.NewRequest("GET", "/seckill/result?request_id=req123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", response["message"])

		mockService.AssertExpectations(t)
	})
}

func TestSeckillHandler_PrewarmActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful prewarm", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.POST("/seckill/prewarm/:activity_id", handler.PrewarmActivity)

		mockService.On("PrewarmActivity", mock.Anything, uint64(123)).Return(nil)

		req, _ := http.NewRequest("POST", "/seckill/prewarm/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])
		assert.Equal(t, "Prewarm successful", response["data"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid activity_id", func(t *testing.T) {
		mockService := new(MockSeckillService)
		handler := NewSeckillHandler(mockService)

		router := gin.New()
		router.POST("/seckill/prewarm/:activity_id", handler.PrewarmActivity)

		req, _ := http.NewRequest("POST", "/seckill/prewarm/invalid", nil)
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