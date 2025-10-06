package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter rate limiter interface
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// SlidingWindowLimiter sliding window rate limiter using Redis
type SlidingWindowLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(client *redis.Client, limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

// Allow checks if the request is allowed
func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixMilli()
	windowStart := now - l.window.Milliseconds()

	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_seconds = tonumber(ARGV[4])

		-- Remove expired entries
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

		-- Get current count in window
		local current = redis.call('ZCARD', key)

		if current < limit then
			-- Add current request
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, window_seconds)
			return 1
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script,
		[]string{rateLimitKey},
		now,
		windowStart,
		l.limit,
		int(l.window.Seconds())).Int()

	if err != nil {
		return false, err
	}

	return result == 1, nil
}

// TokenBucketLimiter token bucket rate limiter using golang.org/x/time/rate
type TokenBucketLimiter struct {
	limiter *rate.Limiter
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(r rate.Limit, b int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		limiter: rate.NewLimiter(r, b),
	}
}

// Allow checks if the request is allowed
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.limiter.Allow(), nil
}

// AllowN checks if n requests are allowed
func (l *TokenBucketLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	return l.limiter.AllowN(time.Now(), n), nil
}

// MultiDimensionLimiter multi-dimension rate limiter
type MultiDimensionLimiter struct {
	client  *redis.Client
	limiters map[string]*LimiterConfig
}

// LimiterConfig limiter configuration
type LimiterConfig struct {
	Limit  int
	Window time.Duration
}

// NewMultiDimensionLimiter creates a new multi-dimension rate limiter
func NewMultiDimensionLimiter(client *redis.Client) *MultiDimensionLimiter {
	return &MultiDimensionLimiter{
		client: client,
		limiters: map[string]*LimiterConfig{
			"global":   {Limit: 50000, Window: time.Minute},
			"user":     {Limit: 50, Window: time.Minute},
			"ip":       {Limit: 500, Window: time.Minute},
			"activity": {Limit: 20000, Window: time.Minute},
		},
	}
}

// Allow checks if the request is allowed across all dimensions
func (l *MultiDimensionLimiter) Allow(ctx context.Context, dimensions map[string]string) (bool, error) {
	for dimension, key := range dimensions {
		config, ok := l.limiters[dimension]
		if !ok {
			continue
		}

		limiter := NewSlidingWindowLimiter(l.client, config.Limit, config.Window)
		allowed, err := limiter.Allow(ctx, fmt.Sprintf("%s:%s", dimension, key))
		if err != nil {
			return false, err
		}

		if !allowed {
			return false, nil
		}
	}

	return true, nil
}

// SetLimit sets the limit for a dimension
func (l *MultiDimensionLimiter) SetLimit(dimension string, limit int, window time.Duration) {
	l.limiters[dimension] = &LimiterConfig{
		Limit:  limit,
		Window: window,
	}
}

