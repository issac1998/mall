package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"seckill/internal/config"
	"seckill/internal/handler"
	"seckill/internal/middleware"
	"seckill/internal/model"
	"seckill/internal/repository"
	"seckill/internal/service/auth"
	"seckill/internal/utils"
)

// AuthTestEnv 认证测试环境
type AuthTestEnv struct {
	DB         *gorm.DB
	Redis      *redis.Client
	Router     *gin.Engine
	Config     *config.Config
	JWTManager *utils.JWTManager
	AuthService auth.AuthService
	AuthHandler *handler.AuthHandler
}

var authTestEnv *AuthTestEnv

// setupAuthTestEnv 设置认证测试环境
func setupAuthTestEnv(t *testing.T) *AuthTestEnv {
	if authTestEnv != nil {
		return authTestEnv
	}

	// 设置测试配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
			Port:     getEnvOrDefaultIntAuth("TEST_DB_PORT", 3306),
			Username: getEnvOrDefault("TEST_DB_USER", "root"),
			Password: getEnvOrDefault("TEST_DB_PASSWORD", ""),
			DBName:   getEnvOrDefault("TEST_DB_NAME", "seckill_test"),
		},
		Redis: config.RedisConfig{
			Host:     getEnvOrDefault("TEST_REDIS_HOST", "localhost"),
			Port:     getEnvOrDefaultIntAuth("TEST_REDIS_PORT", 6379),
			Password: getEnvOrDefault("TEST_REDIS_PASSWORD", ""),
			DB:       0,
		},
		Security: config.SecurityConfig{},
		Server: config.ServerConfig{
			Port: 8080,
		},
	}

	// 设置JWT配置
	cfg.Security.JWT.Secret = "test-secret-key-for-integration-tests"
	cfg.Security.JWT.Expire = 24 * time.Hour
	cfg.Security.JWT.RefreshTTL = 7 * 24 * time.Hour
	cfg.Security.JWT.Issuer = "seckill-test"

	// 初始化数据库
	db, err := initAuthTestDB(cfg)
	require.NoError(t, err, "Failed to initialize test database")

	// 初始化Redis
	rdb, err := initAuthTestRedis(cfg)
	require.NoError(t, err, "Failed to initialize test redis")

	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		cfg.Security.JWT.Secret,
		cfg.Security.JWT.Issuer,
		cfg.Security.JWT.Expire,
		cfg.Security.JWT.RefreshTTL,
	)

	// 初始化仓储层
	userRepo := repository.NewUserRepository(db)

	// 初始化服务层
	authService := auth.NewAuthService(userRepo, jwtManager, rdb)

	// 初始化处理器层
	authHandler := handler.NewAuthHandler(authService)

	// 初始化路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 设置路由
	setupAuthRoutes(router, authHandler, jwtManager)

	authTestEnv = &AuthTestEnv{
		DB:          db,
		Redis:       rdb,
		Router:      router,
		Config:      cfg,
		JWTManager:  jwtManager,
		AuthService: authService,
		AuthHandler: authHandler,
	}

	return authTestEnv
}

// teardownAuthTestEnv 清理认证测试环境
func teardownAuthTestEnv(t *testing.T) {
	if authTestEnv == nil {
		return
	}

	// 清理数据库
	cleanupAuthTestDB(t, authTestEnv.DB)

	// 清理Redis
	cleanupAuthTestRedis(t, authTestEnv.Redis)

	authTestEnv = nil
}

// initAuthTestDB 初始化认证测试数据库
func initAuthTestDB(cfg *config.Config) (*gorm.DB, error) {
	// 创建测试数据库（如果不存在）
	err := createAuthTestDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create test database: %w", err)
	}

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&model.User{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate test database: %w", err)
	}

	return db, nil
}

// createAuthTestDatabase 创建认证测试数据库
func createAuthTestDatabase(cfg *config.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.Database.DBName))
	return err
}

// initAuthTestRedis 初始化认证测试Redis
func initAuthTestRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test redis: %w", err)
	}

	return rdb, nil
}

// setupAuthRoutes 设置认证路由
func setupAuthRoutes(router *gin.Engine, authHandler *handler.AuthHandler, jwtManager *utils.JWTManager) {
	// 认证相关路由
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// 创建token验证函数
	tokenValidator := func(token string) (*middleware.UserInfo, error) {
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			return nil, err
		}
		return &middleware.UserInfo{
			ID:   claims.UserID,
			Role: claims.Role,
		}, nil
	}

	// 需要认证的路由
	api := router.Group("/api")
	api.Use(middleware.Auth(tokenValidator))
	{
		// 测试用的受保护路由
		api.GET("/profile", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(200, gin.H{
				"message": "success",
				"user_id": userID,
			})
		})
	}
}

// cleanupAuthTestDB 清理认证测试数据库
func cleanupAuthTestDB(t *testing.T, db *gorm.DB) {
	err := db.Exec("TRUNCATE TABLE users").Error
	if err != nil {
		t.Logf("Warning: failed to truncate users table: %v", err)
	}
}

// cleanupAuthTestRedis 清理认证测试Redis
func cleanupAuthTestRedis(t *testing.T, rdb *redis.Client) {
	ctx := context.Background()
	err := rdb.FlushDB(ctx).Err()
	if err != nil {
		t.Logf("Warning: failed to flush test redis: %v", err)
	}
}

// getEnvOrDefaultIntAuth 获取环境变量或默认值（整数）
func getEnvOrDefaultIntAuth(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// TestAuthIntegration 认证服务集成测试
func TestAuthIntegration(t *testing.T) {
	env := setupAuthTestEnv(t)
	defer teardownAuthTestEnv(t)

	t.Run("用户注册流程", func(t *testing.T) {
		// 准备注册请求
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
		env.Router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])
		assert.NotNil(t, response["data"])

		// 验证用户已创建
		var user model.User
		err = env.DB.Where("username = ?", "testuser").First(&user).Error
		assert.NoError(t, err)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "13800138000", user.Phone)
		assert.Equal(t, "test@example.com", *user.Email)
	})

	t.Run("用户登录流程", func(t *testing.T) {
		// 先注册一个用户
		registerReq := map[string]interface{}{
			"username": "loginuser",
			"phone":    "13800138001",
			"email":    "login@example.com",
			"password": "password123",
		}

		reqBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 用户登录
		loginReq := map[string]interface{}{
			"account":  "loginuser",
			"password": "password123",
		}

		reqBody, _ = json.Marshal(loginReq)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])

		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"])
		assert.NotEmpty(t, data["refresh_token"])
	})

	t.Run("JWT认证中间件测试", func(t *testing.T) {
		// 先注册一个用户
		registerReq := map[string]interface{}{
			"username": "authuser",
			"phone":    "13800138002",
			"email":    "auth@example.com",
			"password": "password123",
		}

		reqBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 登录获取token
		loginReq := map[string]interface{}{
			"account":  "authuser",
			"password": "password123",
		}

		reqBody, _ = json.Marshal(loginReq)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var loginResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		assert.NoError(t, err)

		// 检查登录是否成功
		if !assert.Equal(t, http.StatusOK, w.Code) {
			t.Logf("Login failed with response: %s", w.Body.String())
			return
		}

		data, ok := loginResponse["data"].(map[string]interface{})
		if !assert.True(t, ok, "data field should be a map") {
			t.Logf("Response: %s", w.Body.String())
			return
		}

		accessToken, ok := data["access_token"].(string)
		if !assert.True(t, ok, "access_token should be a string") {
			t.Logf("Data: %+v", data)
			return
		}

		// 使用token访问受保护的路由
		req = httptest.NewRequest("GET", "/api/profile", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])
		assert.NotNil(t, response["user_id"])
	})

	t.Run("无效token测试", func(t *testing.T) {
		// 使用无效token访问受保护的路由
		req := httptest.NewRequest("GET", "/api/profile", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("重复注册测试", func(t *testing.T) {
		// 先注册一个用户
		registerReq := map[string]interface{}{
			"username": "duplicateuser",
			"phone":    "13800138003",
			"email":    "duplicate@example.com",
			"password": "password123",
		}

		reqBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 再次注册相同用户名
		req = httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"].(string), "username already exists")
	})

	t.Run("错误密码登录测试", func(t *testing.T) {
		// 先注册一个用户
		registerReq := map[string]interface{}{
			"username": "wrongpassuser",
			"phone":    "13800138004",
			"email":    "wrongpass@example.com",
			"password": "password123",
		}

		reqBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 使用错误密码登录
		loginReq := map[string]interface{}{
			"account":  "wrongpassuser",
			"password": "wrongpassword",
		}

		reqBody, _ = json.Marshal(loginReq)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"].(string), "username or password incorrect")
	})
}