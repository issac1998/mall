package limiter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func setupRedis(t *testing.T) *redis.Client {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	
	return client
}

func TestSlidingWindowLimiter(t *testing.T) {
	client := setupRedis(t)
	ctx := context.Background()

	t.Run("AllowWithinLimit", func(t *testing.T) {
		limiter := NewSlidingWindowLimiter(client, 5, time.Minute)
		
		// Should allow first 5 requests with different keys
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(ctx, fmt.Sprintf("test_key_%d", i))
			assert.NoError(t, err)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}
	})

	t.Run("RejectAfterLimit", func(t *testing.T) {
		limiter := NewSlidingWindowLimiter(client, 1, time.Minute)
		
		// Use first request
		allowed, err := limiter.Allow(ctx, "reject_test_key")
		assert.NoError(t, err)
		assert.True(t, allowed)
		
		// Should reject 2nd request
		allowed, err = limiter.Allow(ctx, "reject_test_key")
		assert.NoError(t, err)
		assert.False(t, allowed, "2nd request should be rejected")
	})

	t.Run("DifferentKeys", func(t *testing.T) {
		limiter := NewSlidingWindowLimiter(client, 2, time.Minute)
		
		// Different keys should have separate limits
		allowed1, err := limiter.Allow(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, allowed1)
		
		allowed2, err := limiter.Allow(ctx, "key2")
		assert.NoError(t, err)
		assert.True(t, allowed2)
		
		// Both keys should still allow one more request
		allowed1, err = limiter.Allow(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, allowed1)
		
		allowed2, err = limiter.Allow(ctx, "key2")
		assert.NoError(t, err)
		assert.True(t, allowed2)
	})

	t.Run("WindowSliding", func(t *testing.T) {
		limiter := NewSlidingWindowLimiter(client, 1, time.Second)
		
		// First request should be allowed
		allowed, err := limiter.Allow(ctx, "sliding_key")
		assert.NoError(t, err)
		assert.True(t, allowed, "First request should be allowed")
		
		// Second request should be blocked (same key, within window)
		allowed, err = limiter.Allow(ctx, "sliding_key")
		assert.NoError(t, err)
		assert.False(t, allowed, "Second request should be blocked")
		
		// Wait for window to slide
		time.Sleep(1100 * time.Millisecond)
		
		// Should allow again after window slides
		allowed, err = limiter.Allow(ctx, "sliding_key")
		assert.NoError(t, err)
		assert.True(t, allowed, "Request after window slide should be allowed")
	})
}

func TestTokenBucketLimiter(t *testing.T) {
	ctx := context.Background()

	t.Run("AllowWithinRate", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(rate.Limit(10), 10) // 10 requests per second, burst of 10
		
		// Should allow burst requests
		for i := 0; i < 10; i++ {
			allowed, err := limiter.Allow(ctx, "test_key")
			assert.NoError(t, err)
			assert.True(t, allowed)
		}
		
		// Should reject next request (no tokens left)
		allowed, err := limiter.Allow(ctx, "test_key")
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("AllowN", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(rate.Limit(10), 10)
		
		// Should allow 5 requests at once
		allowed, err := limiter.AllowN(ctx, "test_key", 5)
		assert.NoError(t, err)
		assert.True(t, allowed)
		
		// Should allow another 5 requests
		allowed, err = limiter.AllowN(ctx, "test_key", 5)
		assert.NoError(t, err)
		assert.True(t, allowed)
		
		// Should reject request for 1 more (no tokens left)
		allowed, err = limiter.AllowN(ctx, "test_key", 1)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("KeyIgnored", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(rate.Limit(5), 5)
		
		// Token bucket doesn't use key, so different keys share the same bucket
		allowed1, err := limiter.Allow(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, allowed1)
		
		allowed2, err := limiter.Allow(ctx, "key2")
		assert.NoError(t, err)
		assert.True(t, allowed2)
		
		// Continue until bucket is empty
		for i := 0; i < 3; i++ {
			allowed, err := limiter.Allow(ctx, "any_key")
			assert.NoError(t, err)
			assert.True(t, allowed)
		}
		
		// Should be rejected now
		allowed, err := limiter.Allow(ctx, "any_key")
		assert.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestMultiDimensionLimiter(t *testing.T) {
	client := setupRedis(t)
	ctx := context.Background()

	t.Run("AllowWithinAllLimits", func(t *testing.T) {
		limiter := NewMultiDimensionLimiter(client)
		
		dimensions := map[string]string{
			"user": "user123",
			"ip":   "192.168.1.1",
		}
		
		// Should allow request within all limits
		allowed, err := limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("RejectWhenOneDimensionExceeded", func(t *testing.T) {
		limiter := NewMultiDimensionLimiter(client)
		
		// Set a very low limit for user dimension
		limiter.SetLimit("user", 1, time.Minute)
		
		dimensions := map[string]string{
			"user": "user456",
			"ip":   "192.168.1.2",
		}
		
		// First request should be allowed
		allowed, err := limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.True(t, allowed)
		
		// Second request should be rejected (user limit exceeded)
		allowed, err = limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("SetLimit", func(t *testing.T) {
		limiter := NewMultiDimensionLimiter(client)
		
		// Set custom limit with unique key
		limiter.SetLimit("custom_test", 1, time.Minute)
		
		dimensions := map[string]string{
			"custom_test": "unique_test_value_123",
		}
		
		// Should allow 1 request
		allowed, err := limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.True(t, allowed, "First request should be allowed")
		
		// Should reject 2nd request
		allowed, err = limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.False(t, allowed, "2nd request should be rejected")
	})

	t.Run("UnknownDimensionIgnored", func(t *testing.T) {
		limiter := NewMultiDimensionLimiter(client)
		
		dimensions := map[string]string{
			"unknown": "value",
			"user":    "user789",
		}
		
		// Should allow request (unknown dimension is ignored)
		allowed, err := limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("EmptyDimensions", func(t *testing.T) {
		limiter := NewMultiDimensionLimiter(client)
		
		dimensions := map[string]string{}
		
		// Should allow request with empty dimensions
		allowed, err := limiter.Allow(ctx, dimensions)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestLimiterConfig(t *testing.T) {
	t.Run("NewLimiterConfig", func(t *testing.T) {
		config := &LimiterConfig{
			Limit:  100,
			Window: time.Hour,
		}
		
		assert.Equal(t, 100, config.Limit)
		assert.Equal(t, time.Hour, config.Window)
	})
}

func TestRateLimiterInterface(t *testing.T) {
	client := setupRedis(t)
	ctx := context.Background()

	t.Run("SlidingWindowImplementsInterface", func(t *testing.T) {
		var limiter RateLimiter = NewSlidingWindowLimiter(client, 5, time.Minute)
		
		allowed, err := limiter.Allow(ctx, "test_key")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("TokenBucketImplementsInterface", func(t *testing.T) {
		var limiter RateLimiter = NewTokenBucketLimiter(rate.Limit(10), 10)
		
		allowed, err := limiter.Allow(ctx, "test_key")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}