package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"seckill/internal/config"
	"seckill/internal/consumer"
	"seckill/internal/database"
	"seckill/internal/handler"
	"seckill/internal/middleware"
	"seckill/internal/redis"
	"seckill/internal/repository"
	"seckill/internal/service/auth"
	"seckill/internal/service/order"
	"seckill/internal/service/seckill"
	"seckill/internal/utils"
	"seckill/pkg/breaker"
	"seckill/pkg/limiter"
	"seckill/pkg/log"
	"seckill/pkg/queue"
	"seckill/pkg/snowflake"

	"github.com/gin-gonic/gin"
	redisv9 "github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to load config")
	}
	config.GlobalConfig = cfg

	logConfig := log.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxAge:     cfg.Log.MaxAge,
		MaxBackups: cfg.Log.MaxBackups,
		Compress:   cfg.Log.Compress,
	}
	log.Init(logConfig)

	// database
	if err := database.Init(cfg); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to initialize database")
	}
	defer database.Close()

	// redis
	if err := redis.Init(cfg); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to initialize redis")
	}
	defer redis.Close()

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize dependencies
	db := database.GetDB()
	
	// Create Redis v9 client for services
	redisV9Client := redisv9.NewClient(&redisv9.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Create repositories
	goodsRepo := repository.NewGoodsRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	// Create ID generator
	idGenerator, err := snowflake.NewIDGenerator(1)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to create ID generator")
	}

	// Create message queue
	messageQueue, err := queue.NewMemoryQueue(nil)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to create message queue")
	}

	// Create multi-level inventory
	inventory, err := seckill.NewMultiLevelInventory(redisV9Client)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to create inventory manager")
	}

	router := setupRouter(redisV9Client, goodsRepo, orderRepo, idGenerator, messageQueue, inventory)

	// Start order consumer
	orderConsumer := consumer.NewOrderConsumer(
		order.NewOrderService(orderRepo, goodsRepo, inventory, idGenerator),
		messageQueue,
	)
	orderConsumer.Start(context.Background())

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(cfg.Server.IdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	go func() {
		log.WithFields(map[string]interface{}{
			"port": cfg.Server.Port,
			"mode": cfg.Server.Mode,
		}).Info("Starting HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}

func setupRouter(redisV9Client *redisv9.Client, goodsRepo repository.GoodsRepository, orderRepo repository.OrderRepository, idGenerator *snowflake.IDGenerator, messageQueue *queue.MemoryQueue, inventory *seckill.MultiLevelInventory) *gin.Engine {
	router := gin.New()

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	router.GET("/health", healthCheck)
	router.GET("/ping", ping)

	// Initialize services
	db := database.GetDB()

	// Create repositories
	userRepo := repository.NewUserRepository(db)
	activityRepo := repository.NewActivityRepository(db)

	// Create JWT manager
	cfg := config.GlobalConfig
	jwtManager := utils.NewJWTManager(
		cfg.Security.JWT.Secret,
		cfg.Security.JWT.Issuer,
		cfg.Security.JWT.Expire,
		cfg.Security.JWT.RefreshTTL,
	)

	// Create multi-dimension rate limiter
	rateLimiter := limiter.NewMultiDimensionLimiter(redisV9Client)

	// Create circuit breaker manager
	circuitBreakerManager := breaker.NewManager(breaker.Config{
		MaxRequests:   5,
		Interval:      time.Minute,
		Timeout:       30 * time.Second,
		ReadyToTrip:   nil, // Use default
		OnStateChange: nil,
	})

	// Create services
	authService := auth.NewAuthService(userRepo, jwtManager, redisV9Client)
	seckillService := seckill.NewSeckillService(
		activityRepo,
		inventory,
		rateLimiter,
		circuitBreakerManager,
		messageQueue,
		redisV9Client,
	)

	// Create handlers
	authHandler := handler.NewAuthHandler(authService)
	activityHandler := handler.NewActivityHandler(activityRepo)
	seckillHandler := handler.NewSeckillHandler(seckillService)

	// Setup routes
	api := router.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			v1.GET("/health", healthCheck)
			v1.GET("/ping", ping)

			// Public auth routes
			authGroup := v1.Group("/auth")
			{
				authGroup.POST("/register", authHandler.Register)
				authGroup.POST("/login", authHandler.Login)
				authGroup.POST("/refresh", authHandler.RefreshToken)
			}

			// Protected routes
			tokenValidator := func(token string) (*middleware.UserInfo, error) {
				claims, err := authService.ValidateToken(context.Background(), token)
				if err != nil {
					return nil, err
				}
				return &middleware.UserInfo{
					ID:   claims.UserID,
					Role: claims.Role,
				}, nil
			}

			protected := v1.Group("")
			protected.Use(middleware.Auth(tokenValidator))
			{
				protected.POST("/auth/logout", authHandler.Logout)
				protected.POST("/auth/change-password", authHandler.ChangePassword)
				
				// Activity routes
				protected.GET("/activities", activityHandler.ListActivities)
				protected.GET("/activities/:id", activityHandler.GetActivity)
				
				// Seckill routes
				seckillGroup := protected.Group("/seckill")
				seckillGroup.Use(middleware.SeckillRateLimit())
				{
					seckillGroup.POST("/execute", seckillHandler.DoSeckill)
					seckillGroup.GET("/result/:request_id", seckillHandler.QueryResult)
					seckillGroup.POST("/prewarm/:activity_id", seckillHandler.PrewarmActivity)
				}
			}
		}
	}

	return router
}

func healthCheck(c *gin.Context) {
	dbHealth := checkDatabase()

	redisHealth := checkRedis()

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"services": map[string]interface{}{
			"database": dbHealth,
			"redis":    redisHealth,
		},
	}

	if !dbHealth["healthy"].(bool) || !redisHealth["healthy"].(bool) {
		health["status"] = "error"
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	c.JSON(http.StatusOK, health)
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now().Unix(),
	})
}

func checkDatabase() map[string]interface{} {
	db := database.GetDB()
	if db == nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   "database connection is nil",
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"healthy": true,
		"status":  "connected",
	}
}

func checkRedis() map[string]interface{} {
	client := redis.GetClient()
	if client == nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   "redis client is nil",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"healthy": true,
		"status":  "connected",
	}
}
