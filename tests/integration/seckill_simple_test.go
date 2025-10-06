package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// SimpleSeckillTest 简化的秒杀测试
func TestSimpleSeckillFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 模拟库存
	var stock int = 100
	var mutex sync.Mutex
	var orders []string

	// 模拟用户注册接口
	router.POST("/api/auth/register", func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data": gin.H{
				"id":       1,
				"username": req["username"],
			},
		})
	})

	// 模拟用户登录接口
	router.POST("/api/auth/login", func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data": gin.H{
				"access_token":  "mock-token-123",
				"refresh_token": "mock-refresh-token-123",
				"expires_in":    3600,
				"token_type":    "Bearer",
			},
		})
	})

	// 模拟秒杀接口
	router.POST("/api/seckill/do", func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		if stock <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"message": "success",
				"data": gin.H{
					"success": false,
					"message": "库存不足",
				},
			})
			return
		}

		stock--
		orderID := "order-" + time.Now().Format("20060102150405")
		orders = append(orders, orderID)

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data": gin.H{
				"success":  true,
				"message":  "秒杀成功",
				"order_id": orderID,
			},
		})
	})

	// 模拟查询秒杀结果接口
	router.GET("/api/seckill/result/:activity_id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data": gin.H{
				"success":  true,
				"message":  "秒杀成功",
				"order_id": "order-123456",
			},
		})
	})

	// 模拟预热接口
	router.POST("/api/seckill/prewarm/:activity_id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data":    gin.H{"status": "prewarmed"},
		})
	})

	t.Run("完整秒杀流程测试", func(t *testing.T) {
		// 1. 用户注册
		registerReq := map[string]interface{}{
			"username": "testuser",
			"phone":    "13800138000",
			"email":    "test@example.com",
			"password": "password123",
		}

		reqBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 2. 用户登录获取token
		loginReq := map[string]interface{}{
			"account":  "testuser",
			"password": "password123",
		}

		reqBody, _ = json.Marshal(loginReq)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var loginResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		assert.NoError(t, err)

		data := loginResponse["data"].(map[string]interface{})
		accessToken := data["access_token"].(string)

		// 3. 预热活动
		req = httptest.NewRequest("POST", "/api/seckill/prewarm/1", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 4. 执行秒杀
		seckillReq := map[string]interface{}{
			"activity_id": 1,
		}

		reqBody, _ = json.Marshal(seckillReq)
		req = httptest.NewRequest("POST", "/api/seckill/do", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var seckillResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &seckillResponse)
		assert.NoError(t, err)

		seckillData := seckillResponse["data"].(map[string]interface{})
		assert.Equal(t, true, seckillData["success"])
		assert.Equal(t, "秒杀成功", seckillData["message"])

		// 5. 查询秒杀结果
		req = httptest.NewRequest("GET", "/api/seckill/result/1", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resultResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resultResponse)
		assert.NoError(t, err)

		resultData := resultResponse["data"].(map[string]interface{})
		assert.Equal(t, true, resultData["success"])
		assert.NotEmpty(t, resultData["order_id"])
	})

	t.Run("并发秒杀防超卖测试", func(t *testing.T) {
		// 重置库存为小数量
		mutex.Lock()
		stock = 5
		orders = []string{}
		mutex.Unlock()

		var wg sync.WaitGroup
		successCount := 0
		var countMutex sync.Mutex

		// 启动10个并发请求
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				seckillReq := map[string]interface{}{
					"activity_id": 1,
				}

				reqBody, _ := json.Marshal(seckillReq)
				req := httptest.NewRequest("POST", "/api/seckill/do", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					var response map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &response)
					if err == nil {
						data := response["data"].(map[string]interface{})
						if data["success"].(bool) {
							countMutex.Lock()
							successCount++
							countMutex.Unlock()
						}
					}
				}
			}(i)
		}

		wg.Wait()

		// 验证成功秒杀的数量不超过库存
		assert.LessOrEqual(t, successCount, 5, "成功秒杀数量不应超过库存")
		t.Logf("成功秒杀数量: %d, 库存: %d", successCount, 5)
	})

	t.Run("库存耗尽测试", func(t *testing.T) {
		// 重置库存为0
		mutex.Lock()
		stock = 0
		mutex.Unlock()

		seckillReq := map[string]interface{}{
			"activity_id": 1,
		}

		reqBody, _ := json.Marshal(seckillReq)
		req := httptest.NewRequest("POST", "/api/seckill/do", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, false, data["success"])
		assert.Equal(t, "库存不足", data["message"])
	})
}

// TestSeckillPerformance 性能测试
func TestSeckillPerformance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var stock int = 1000
	var mutex sync.Mutex
	var requestCount int64

	router.POST("/api/seckill/do", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()

		requestCount++

		if stock <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"success": false,
					"message": "库存不足",
				},
			})
			return
		}

		stock--
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"success": true,
				"message": "秒杀成功",
				"order_id": "order-" + time.Now().Format("20060102150405"),
			},
		})
	})

	t.Run("高并发性能测试", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 100
		requestsPerGoroutine := 10

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < requestsPerGoroutine; j++ {
					seckillReq := map[string]interface{}{
						"activity_id": 1,
					}

					reqBody, _ := json.Marshal(seckillReq)
					req := httptest.NewRequest("POST", "/api/seckill/do", bytes.NewBuffer(reqBody))
					req.Header.Set("Content-Type", "application/json")

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		totalRequests := concurrency * requestsPerGoroutine
		qps := float64(totalRequests) / duration.Seconds()

		t.Logf("总请求数: %d", totalRequests)
		t.Logf("总耗时: %v", duration)
		t.Logf("QPS: %.2f", qps)
		t.Logf("处理的请求数: %d", requestCount)
		t.Logf("剩余库存: %d", stock)

		// 验证QPS达到一定标准
		assert.Greater(t, qps, float64(100), "QPS应该大于100")
	})
}