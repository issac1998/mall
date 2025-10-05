package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"seckill/pkg/utils"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// TestLogger 测试日志中间件
func TestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name   string
		path   string
		method string
		status int
	}{
		{
			name:   "GET request",
			path:   "/test",
			method: "GET",
			status: 200,
		},
		{
			name:   "POST request",
			path:   "/test",
			method: "POST",
			status: 201,
		},
		{
			name:   "Error request",
			path:   "/error",
			method: "GET",
			status: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Logger())
			
			r.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})
			r.POST("/test", func(c *gin.Context) {
				c.JSON(201, gin.H{"message": "created"})
			})
			r.GET("/error", func(c *gin.Context) {
				c.JSON(500, gin.H{"error": "internal error"})
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)
		})
	}
}

// TestRecovery 测试恢复中间件
func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.Use(Recovery())
	
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})
	
	r.GET("/normal", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	// 测试panic恢复
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, 500, w.Code)
	
	var response utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1004, response.Code) // CodeInternalError = 1004

	// 测试正常请求
	req = httptest.NewRequest("GET", "/normal", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, 200, w.Code)
}

// TestCORS 测试CORS中间件
func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "Valid origin",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedStatus: 200,
			checkHeaders:   true,
		},
		{
			name:           "OPTIONS request",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedStatus: 204,
			checkHeaders:   true,
		},
		{
			name:           "No origin",
			origin:         "",
			method:         "GET",
			expectedStatus: 200,
			checkHeaders:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(CORS())
			
			r.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.checkHeaders {
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

// TestAuth 测试认证中间件
func TestAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 模拟token验证函数
	validator := func(token string) (*UserInfo, error) {
		if token == "valid_token" {
			return &UserInfo{ID: 1, Role: "user"}, nil
		}
		return nil, assert.AnError
	}

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedUserID int64
	}{
		{
			name:           "Valid token",
			token:          "Bearer valid_token",
			expectedStatus: 200,
			expectedUserID: 1,
		},
		{
			name:           "Invalid token",
			token:          "Bearer invalid_token",
			expectedStatus: 401,
			expectedUserID: 0,
		},
		{
			name:           "No token",
			token:          "",
			expectedStatus: 401,
			expectedUserID: 0,
		},
		{
			name:           "Invalid format",
			token:          "invalid_format",
			expectedStatus: 401,
			expectedUserID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Auth(validator))
			
			r.GET("/test", func(c *gin.Context) {
				userID, exists := GetUserID(c)
				if exists {
					c.JSON(200, gin.H{"user_id": userID})
				} else {
					c.JSON(200, gin.H{"message": "no user"})
				}
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestOptionalAuth 测试可选认证中间件
func TestOptionalAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	validator := func(token string) (*UserInfo, error) {
		if token == "valid_token" {
			return &UserInfo{ID: 1, Role: "user"}, nil
		}
		return nil, assert.AnError
	}

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		hasUser        bool
	}{
		{
			name:           "Valid token",
			token:          "Bearer valid_token",
			expectedStatus: 200,
			hasUser:        true,
		},
		{
			name:           "Invalid token",
			token:          "Bearer invalid_token",
			expectedStatus: 200,
			hasUser:        false,
		},
		{
			name:           "No token",
			token:          "",
			expectedStatus: 200,
			hasUser:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(OptionalAuth(validator))
			
			r.GET("/test", func(c *gin.Context) {
				userID, exists := GetUserID(c)
				c.JSON(200, gin.H{
					"has_user": exists,
					"user_id":  userID,
				})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.hasUser, response["has_user"])
		})
	}
}

// TestTimeout 测试超时中间件
func TestTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		timeout        time.Duration
		handlerDelay   time.Duration
		expectedStatus int
	}{
		{
			name:           "Normal request",
			timeout:        100 * time.Millisecond,
			handlerDelay:   50 * time.Millisecond,
			expectedStatus: 200,
		},
		{
			name:           "Timeout request",
			timeout:        50 * time.Millisecond,
			handlerDelay:   100 * time.Millisecond,
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Timeout(tt.timeout))
			
			r.GET("/test", func(c *gin.Context) {
				select {
				case <-time.After(tt.handlerDelay):
					c.JSON(200, gin.H{"message": "ok"})
				case <-c.Request.Context().Done():
					return
				}
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestGetUserID 测试获取用户ID
func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name     string
		userID   interface{}
		expected int64
		exists   bool
	}{
		{
			name:     "int64 user ID",
			userID:   int64(123),
			expected: 123,
			exists:   true,
		},
		{
			name:     "int user ID",
			userID:   123,
			expected: 123,
			exists:   true,
		},
		{
			name:     "string user ID",
			userID:   "123",
			expected: 123,
			exists:   true,
		},
		{
			name:     "invalid string user ID",
			userID:   "invalid",
			expected: 0,
			exists:   false,
		},
		{
			name:     "no user ID",
			userID:   nil,
			expected: 0,
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			
			if tt.userID != nil {
				c.Set(UserIDKey, tt.userID)
			}

			userID, exists := GetUserID(c)
			assert.Equal(t, tt.expected, userID)
			assert.Equal(t, tt.exists, exists)
		})
	}
}

// TestGetUserRole 测试获取用户角色
func TestGetUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name     string
		role     interface{}
		expected string
		exists   bool
	}{
		{
			name:     "valid role",
			role:     "admin",
			expected: "admin",
			exists:   true,
		},
		{
			name:     "invalid role type",
			role:     123,
			expected: "",
			exists:   false,
		},
		{
			name:     "no role",
			role:     nil,
			expected: "",
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			
			if tt.role != nil {
				c.Set(UserRoleKey, tt.role)
			}

			role, exists := GetUserRole(c)
			assert.Equal(t, tt.expected, role)
			assert.Equal(t, tt.exists, exists)
		})
	}
}