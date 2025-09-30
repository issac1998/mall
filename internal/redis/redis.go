package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"seckill/internal/config"
	"seckill/pkg/log"
)

var (
	Client *redis.Client
)

// Init initializes the Redis client with the given configuration.
func Init(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Redis.IdleTimeout) * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	log.Info("Redis connected successfully")
	return nil
}


// Close closes the Redis client connection.
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

// GetClient returns the Redis client instance.
func GetClient() *redis.Client {
	return Client
}

// Health checks the health status of the Redis client.
func Health() error {
	if Client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return Client.Ping(ctx).Err()
}

// WithContext returns a Redis client with the given context.
func WithContext(ctx context.Context) *redis.Client {
	if Client == nil {
		return nil
	}
	return Client.WithContext(ctx)
}

// Set sets the v