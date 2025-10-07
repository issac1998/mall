package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"seckill/internal/config"
	"seckill/internal/handler"
	"seckill/internal/middleware"
	"seckill/internal/model"
	"seckill/internal/repository"
	"seckill/internal/service/auth"
	"seckill/internal/service/seckill"
	"seckill/internal/utils"
	"seckill/pkg/breaker"
	"seckill/pkg/degrade"
	"seckill/pkg/limiter"
	"seckill/pkg/lock"
	"seckill/pkg/queue"
	"seckill/pkg/snowflake"
)

// IntegrationTestEnv 集成测试环境
type IntegrationTestEnv struct {
	DB          *gorm.DB
	Redis       *redis.Client
	Router      *gin.Engine
	Config      *config.Config
	JWTManager  *utils.JWTManager
	Lock        *lock.RedisLock
	Queue       *queue.MemoryQueue
	IDGenerator *snowflake.IDGenerator

	// Services
	AuthService    auth.AuthService
	SeckillService seckill.SeckillService

	// Handlers
	AuthHandler    *handler.AuthHandler
	SeckillHandler *handler.SeckillHandler
}

// SetupIntegrationTestEnv 设置集成测试环境
func SetupIntegrationTestEnv(t *testing.T) *IntegrationTestEnv {
	// 设置测试配置
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Mode: "test",
		},
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
			Port:     getEnvIntOrDefault("TEST_DB_PORT", 3306),
			Username: getEnvOrDefault("TEST_DB_USER", "root"),
			Password: getEnvOrDefault("TEST_DB_PASSWORD", ""),
			DBName:   getEnvOrDefault("TEST_DB_NAME", "seckill_test"),
		},
		Redis: config.RedisConfig{
			Host:     getEnvOrDefault("TEST_REDIS_HOST", "localhost"),
			Port:     getEnvIntOrDefault("TEST_REDIS_PORT", 6379),
			Password: getEnvOrDefault("TEST_REDIS_PASSWORD", ""),
			DB:       getEnvIntOrDefault("TEST_REDIS_DB", 1),
		},
		Security: config.SecurityConfig{
			JWT: struct {
				Secret     string        `mapstructure:"secret"`
				Expire     time.Duration `mapstructure:"expire"`
				RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
				Issuer     string        `mapstructure:"issuer"`
			}{
				Secret:     "test-jwt-secret-key-for-integration-tests",
				Expire:     time.Hour,
				RefreshTTL: 24 * time.Hour,
				Issuer:     "seckill-test",
			},
		},
	}

	// 初始化数据库
	db := setupTestDB(t, cfg.Database)

	// 初始化Redis
	redisClient := setupTestRedis(t, cfg.Redis)

	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		cfg.Security.JWT.Secret,
		cfg.Security.JWT.Issuer,
		cfg.Security.JWT.Expire,
		cfg.Security.JWT.RefreshTTL,
	)

	// 初始化分布式锁
	distributedLock := lock.NewRedisLock(redisClient, "test", "test-node", 30*time.Second)

	// 初始化消息队列
	messageQueue, err := queue.NewMemoryQueue(&queue.MemoryQueueConfig{
		BufferSize: 1000,
	})
	require.NoError(t, err)

	// 初始化ID生成器
	idGenerator, err := snowflake.NewIDGenerator(1)
	require.NoError(t, err)

	// 初始化Repository
	userRepo := repository.NewUserRepository(db)
	activityRepo := repository.NewActivityRepository(db)

	// 初始化多级库存管理器
	inventory, err := seckill.NewMultiLevelInventory(redisClient)
	require.NoError(t, err)

	// 初始化限流器
	rateLimiter := limiter.NewMultiDimensionLimiter(redisClient)

	// 初始化熔断器
	circuitBreaker := breaker.NewManager(breaker.Config{
		MaxRequests: 10,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts breaker.Counts) bool {
			return counts.TotalFailures >= 5
		},
	})

	// 初始化降级管理器
	degradeManager := degrade.NewDegradeManager(redisClient)
	
	// 初始化Service
	authService := auth.NewAuthService(userRepo, jwtManager, redisClient)
	seckillService := seckill.NewSeckillService(
		activityRepo,
		inventory,
		rateLimiter,
		circuitBreaker,
		degradeManager,
		messageQueue,
		redisClient,
	)

	// 初始化Handler
	authHandler := handler.NewAuthHandler(authService)
	seckillHandler := handler.NewSeckillHandler(seckillService)

	// 设置路由
	router := setupTestRoutes(authHandler, seckillHandler, jwtManager)

	return &IntegrationTestEnv{
		DB:             db,
		Redis:          redisClient,
		Router:         router,
		Config:         cfg,
		JWTManager:     jwtManager,
		Lock:           distributedLock,
		Queue:          messageQueue,
		IDGenerator:    idGenerator,
		AuthService:    authService,
		SeckillService: seckillService,
		AuthHandler:    authHandler,
		SeckillHandler: seckillHandler,
	}
}

// TeardownIntegrationTestEnv 清理测试环境
func TeardownIntegrationTestEnv(env *IntegrationTestEnv) {
	if env.Queue != nil {
		env.Queue.Close()
	}

	if env.Redis != nil {
		cleanupTestRedis(env.Redis)
		env.Redis.Close()
	}

	if env.DB != nil {
		cleanupTestDB(env.DB)
		sqlDB, _ := env.DB.DB()
		sqlDB.Close()
	}
}

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T, cfg config.DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(
		&model.User{},
		&model.SeckillActivity{},
		&model.Goods{},
		&model.Order{},
		&model.OrderDetail{},
		&model.StockLog{},
	)
	require.NoError(t, err)

	return db
}

// setupTestRedis 设置测试Redis
func setupTestRedis(t *testing.T, cfg config.RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	require.NoError(t, err)

	return client
}

// createTokenValidator 创建token验证器
func createTokenValidator(jwtManager *utils.JWTManager) func(token string) (*middleware.UserInfo, error) {
	return func(token string) (*middleware.UserInfo, error) {
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			return nil, err
		}

		return &middleware.UserInfo{
			ID:   claims.UserID,
			Role: claims.Role,
		}, nil
	}
}

// setupTestRoutes 设置测试路由
func setupTestRoutes(authHandler *handler.AuthHandler, seckillHandler *handler.SeckillHandler, jwtManager *utils.JWTManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 添加中间件
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// 创建token验证器
	tokenValidator := createTokenValidator(jwtManager)

	// API路由组
	api := router.Group("/api")

	// 认证路由
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", middleware.Auth(tokenValidator), authHandler.Logout)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// 秒杀路由
	seckillGroup := api.Group("/seckill")
	seckillGroup.Use(middleware.Auth(tokenValidator))
	{
		seckillGroup.POST("/do", seckillHandler.DoSeckill)
		seckillGroup.GET("/result", seckillHandler.QueryResult)
		seckillGroup.POST("/prewarm/:activity_id", seckillHandler.PrewarmActivity)
	}

	return router
}

// cleanupTestDB 清理测试数据库
func cleanupTestDB(db *gorm.DB) {
	// 清理测试数据
	db.Exec("DELETE FROM stock_logs")
	db.Exec("DELETE FROM order_details")
	db.Exec("DELETE FROM orders")
	db.Exec("DELETE FROM seckill_activities")
	db.Exec("DELETE FROM goods")
	db.Exec("DELETE FROM users")
}

// cleanupTestRedis 清理测试Redis
func cleanupTestRedis(client *redis.Client) {
	ctx := context.Background()
	client.FlushDB(ctx)
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault 获取环境变量整数值或默认值
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
