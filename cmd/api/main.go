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
	"seckill/internal/database"
	"seckill/internal/middleware"
	"seckill/internal/redis"
	"seckill/pkg/log"

	"github.com/gin-gonic/gin"
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

	router := setupRouter()

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

func setupRouter() *gin.Engine {
	router := gin.New()

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	router.GET("/health", healthCheck)
	router.GET("/ping", ping)

	api := router.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			v1.GET("/health", healthCheck)
			v1.GET("/ping", ping)
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
