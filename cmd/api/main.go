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
	"seckill/internal/service/stock"
	"seckill/internal/utils"
	"seckill/pkg/breaker"
	"seckill/pkg/degrade"
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
	// activityRepo := repository.NewActivityRepository(db) // For stock sync service (disabled for now)

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

	// Start VIP priority order consumer
	// 3 VIP workers + 10 normal workers
	vipConsumer := consumer.NewVIPPriorityConsumer(
		order.NewOrderService(orderRepo, goodsRepo, inventory, idGenerator),
		messageQueue,
		3,  // VIP workers
		10, // Normal workers
	)
	vipConsumer.Start(context.Background())

	// Create services for workers
	activityRepo := repository.NewActivityRepository(db)
	orderService := order.NewOrderService(orderRepo, goodsRepo, inventory, idGenerator)
	stockService := stock.NewStockService(activityRepo, goodsRepo, inventory, redisV9Client)

	// Create context for workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	// Start all background workers
	startWorkers(workerCtx, orderService, stockService, activityRepo)

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

	// Cancel worker context to stop all workers
	workerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}

// ========== Worker Functions ==========

// startWorkers starts all background workers
func startWorkers(ctx context.Context, orderService order.OrderService, stockService stock.StockService, activityRepo repository.ActivityRepository) {
	// Worker 1: Handle expired orders (every 1 minute)
	go expiredOrderWorker(ctx, orderService, 1*time.Minute)

	// Worker 2: Sync stock from MySQL to Redis (every 3 minutes)
	go stockToRedisWorker(ctx, stockService, activityRepo, 3*time.Minute)

	// Worker 3: Sync stock from Redis to MySQL (every 5 minutes)
	go stockToMySQLWorker(ctx, stockService, activityRepo, 5*time.Minute)

	// Worker 4: Check stock consistency and repair (every 10 minutes)
	go stockConsistencyWorker(ctx, stockService, activityRepo, 10*time.Minute)

	// Worker 5: Start periodic sync service
	go func() {
		stockService.StartPeriodicSync(ctx, 2*time.Minute)
	}()

	log.Info("All workers started successfully")
}

// expiredOrderWorker handles expired orders periodically
func expiredOrderWorker(ctx context.Context, orderService order.OrderService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("Expired order worker started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			log.Info("Expired order worker stopped")
			return
		case <-ticker.C:
			log.Info("Processing expired orders...")
			if err := orderService.HandleExpiredOrders(ctx); err != nil {
				log.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Error("Failed to handle expired orders")
			}
		}
	}
}

// stockToRedisWorker syncs stock from MySQL to Redis periodically
func stockToRedisWorker(ctx context.Context, stockService stock.StockService, activityRepo repository.ActivityRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("Stock to Redis worker started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			log.Info("Stock to Redis worker stopped")
			return
		case <-ticker.C:
			log.Info("Syncing stock from MySQL to Redis...")
			activityIDs := getActiveActivityIDs(ctx, activityRepo)
			for _, activityID := range activityIDs {
				if err := stockService.SyncStockToRedis(ctx, activityID); err != nil {
					log.WithFields(map[string]interface{}{
						"activity_id": activityID,
						"error":       err.Error(),
					}).Error("Failed to sync stock to Redis")
				}
			}
		}
	}
}

// stockToMySQLWorker syncs stock from Redis to MySQL periodically (renamed from stockSyncWorker)
func stockToMySQLWorker(ctx context.Context, stockService stock.StockService, activityRepo repository.ActivityRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("Stock to MySQL worker started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			log.Info("Stock to MySQL worker stopped")
			return
		case <-ticker.C:
			log.Info("Syncing stock from Redis to MySQL...")
			activityIDs := getActiveActivityIDs(ctx, activityRepo)
			for _, activityID := range activityIDs {
				if err := stockService.SyncStockToMySQL(ctx, activityID); err != nil {
					log.WithFields(map[string]interface{}{
						"activity_id": activityID,
						"error":       err.Error(),
					}).Error("Failed to sync stock to MySQL")
				}
			}
		}
	}
}

// stockConsistencyWorker checks stock consistency and repairs inconsistencies periodically
func stockConsistencyWorker(ctx context.Context, stockService stock.StockService, activityRepo repository.ActivityRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("Stock consistency worker started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			log.Info("Stock consistency worker stopped")
			return
		case <-ticker.C:
			log.Info("Checking stock consistency...")
			activityIDs := getActiveActivityIDs(ctx, activityRepo)
			for _, activityID := range activityIDs {
				report, err := stockService.CheckStockConsistency(ctx, activityID)
				if err != nil {
					log.WithFields(map[string]interface{}{
						"activity_id": activityID,
						"error":       err.Error(),
					}).Error("Failed to check stock consistency")
					continue
				}

				// Log consistency report
				log.WithFields(map[string]interface{}{
					"activity_id":    activityID,
					"redis_stock":    report.RedisStock,
					"mysql_stock":    report.MySQLStock,
					"reserved_stock": report.ReservedStock,
					"difference":     report.Difference,
					"is_consistent":  report.IsConsistent,
				}).Info("Stock consistency report")

				// Auto repair inconsistencies
				if !report.IsConsistent {
					log.WithFields(map[string]interface{}{
						"activity_id": activityID,
						"difference":  report.Difference,
					}).Warn("Stock inconsistency detected, attempting repair")

					if err := stockService.RepairStockInconsistency(ctx, activityID); err != nil {
						log.WithFields(map[string]interface{}{
							"activity_id": activityID,
							"error":       err.Error(),
						}).Error("Failed to repair stock inconsistency")
					} else {
						log.WithFields(map[string]interface{}{
							"activity_id": activityID,
						}).Info("Stock inconsistency repaired successfully")
					}
				}
			}
		}
	}
}

// getActiveActivityIDs returns list of active activity IDs from database
func getActiveActivityIDs(ctx context.Context, activityRepo repository.ActivityRepository) []uint64 {
	// Query active activities from database
	activities, _, err := activityRepo.ListActive(ctx, 1, 100) // Get first 100 active activities
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to query active activities")
		return []uint64{}
	}

	activityIDs := make([]uint64, 0, len(activities))
	for _, activity := range activities {
		activityIDs = append(activityIDs, activity.ID)
	}

	log.WithFields(map[string]interface{}{
		"count":        len(activityIDs),
		"activity_ids": activityIDs,
	}).Debug("Active activities loaded")

	return activityIDs
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

	// Create degrade manager
	degradeManager := degrade.NewDegradeManager(redisV9Client)

	// Create services
	authService := auth.NewAuthService(userRepo, jwtManager, redisV9Client)
	seckillService := seckill.NewSeckillService(
		activityRepo,
		inventory,
		rateLimiter,
		circuitBreakerManager,
		degradeManager,
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
