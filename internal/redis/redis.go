package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"seckill/internal/config"
	"seckill/pkg/log"
)

var (
	Client        *redis.Client
	ClusterClient *redis.ClusterClient
)

// Init initializes the Redis client with the given configuration.
func Init(cfg *config.Config) error {
	if cfg.Redis.Cluster.Enabled {
		return initCluster(cfg)
	}
	return initSingle(cfg)
}

// initSingle initializes single Redis client
func initSingle(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		ConnMaxIdleTime: cfg.Redis.IdleTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	log.Info("Redis single mode connected successfully")
	return nil
}

// initCluster initializes Redis cluster client
func initCluster(cfg *config.Config) error {
	ClusterClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          cfg.Redis.Cluster.Addrs,
		Password:       cfg.Redis.Cluster.Password,
		MaxRedirects:   cfg.Redis.Cluster.MaxRedirects,
		ReadOnly:       cfg.Redis.Cluster.ReadOnly,
		RouteByLatency: cfg.Redis.Cluster.RouteByLatency,
		RouteRandomly:  cfg.Redis.Cluster.RouteRandomly,
		
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		ConnMaxIdleTime: cfg.Redis.IdleTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ClusterClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect redis cluster: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"addrs": cfg.Redis.Cluster.Addrs,
	}).Info("Redis cluster mode connected successfully")
	return nil
}

// Close closes the Redis client connection.
func Close() error {
	if ClusterClient != nil {
		if err := ClusterClient.Close(); err != nil {
			return err
		}
	}
	if Client != nil {
		return Client.Close()
	}
	return nil
}

// GetClient returns the Redis client instance.
func GetClient() redis.Cmdable {
	if ClusterClient != nil {
		return ClusterClient
	}
	return Client
}

// Health checks the health status of the Redis client.
func Health() error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return client.Ping(ctx).Err()
}

