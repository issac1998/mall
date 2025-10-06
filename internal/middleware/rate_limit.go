package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"seckill/pkg/log"
)

// MiddlewareRateLimitConfig rate limiting middleware configuration
type MiddlewareRateLimitConfig struct {
	// Rate requests per second
	Rate float64
	// Burst maximum burst size
	Burst int
	// KeyFunc function to generate rate limit key
	KeyFunc func(c *gin.Context) string
	// ErrorHandler error handling function
	ErrorHandler func(c *gin.Context)
	// SkipFunc function to skip rate limiting
	SkipFunc func(c *gin.Context) bool
}

// DefaultMiddlewareRateLimitConfig default rate limiting configuration
func DefaultMiddlewareRateLimitConfig() MiddlewareRateLimitConfig {
	return MiddlewareRateLimitConfig{
		Rate:  100,  // 100 requests per second
		Burst: 200,  // Maximum burst of 200 requests
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
		ErrorHandler: func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many requests",
			})
			c.Abort()
		},
		SkipFunc: func(c *gin.Context) bool {
			return false
		},
	}
}

// RateLimit rate limiting middleware
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	config := DefaultMiddlewareRateLimitConfig()
	config.Rate = rps
	config.Burst = burst
	return RateLimitWithConfig(config)
}

// RateLimitWithConfig rate limiting middleware with configuration
func RateLimitWithConfig(config MiddlewareRateLimitConfig) gin.HandlerFunc {
	limiters := make(map[string]*rate.Limiter)
	
	return func(c *gin.Context) {
		// Skip if skip function returns true
		if config.SkipFunc(c) {
			c.Next()
			return
		}

		// Generate rate limit key
		key := config.KeyFunc(c)
		
		// Get or create limiter for this key
		limiter, exists := limiters[key]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(config.Rate), config.Burst)
			limiters[key] = limiter
		}

		// Check if request is allowed
		if !limiter.Allow() {
			log.WithFields(map[string]interface{}{
				"key":    key,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"ip":     c.ClientIP(),
			}).Warn("Rate limit exceeded")
			
			config.ErrorHandler(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimit IP-based rate limiting middleware
func IPRateLimit(rps float64, burst int) gin.HandlerFunc {
	config := DefaultMiddlewareRateLimitConfig()
	config.Rate = rps
	config.Burst = burst
	config.KeyFunc = func(c *gin.Context) string {
		return c.ClientIP()
	}
	return RateLimitWithConfig(config)
}

// UserRateLimit user-based rate limiting middleware
func UserRateLimit(rps float64, burst int) gin.HandlerFunc {
	config := DefaultMiddlewareRateLimitConfig()
	config.Rate = rps
	config.Burst = burst
	config.KeyFunc = func(c *gin.Context) string {
		userID := c.GetString("user_id")
		if userID == "" {
			return c.ClientIP()
		}
		return fmt.Sprintf("user:%s", userID)
	}
	return RateLimitWithConfig(config)
}

// APIRateLimit API-based rate limiting middleware
func APIRateLimit(rps float64, burst int) gin.HandlerFunc {
	config := DefaultMiddlewareRateLimitConfig()
	config.Rate = rps
	config.Burst = burst
	config.KeyFunc = func(c *gin.Context) string {
		return fmt.Sprintf("%s:%s", c.ClientIP(), c.Request.URL.Path)
	}
	return RateLimitWithConfig(config)
}

// SeckillRateLimit seckill-specific rate limiting middleware
func SeckillRateLimit() gin.HandlerFunc {
	config := DefaultMiddlewareRateLimitConfig()
	config.Rate = 50   // 50 requests per second for seckill
	config.Burst = 100  // Maximum burst of 100 requests
	config.KeyFunc = func(c *gin.Context) string {
		userID := c.GetString("user_id")
		if userID == "" {
			return c.ClientIP()
		}
		return fmt.Sprintf("seckill:%s", userID)
	}
	config.ErrorHandler = func(c *gin.Context) {
		c.Header("X-RateLimit-Limit", strconv.FormatFloat(config.Rate, 'f', 0, 64))
		c.Header("X-RateLimit-Remaining", "0")
		c.Header("Retry-After", "1")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":    429,
			"message": "Seckill rate limit exceeded, please try again later",
		})
	}
	return RateLimitWithConfig(config)
}