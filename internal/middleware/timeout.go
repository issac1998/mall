package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"seckill/pkg/utils"
)

// TimeoutConfig timeout configuration
type TimeoutConfig struct {
	// Timeout timeout duration
	Timeout time.Duration
	// ErrorHandler timeout error handler function
	ErrorHandler gin.HandlerFunc
	// SkipFunc function to skip timeout check
	SkipFunc func(*gin.Context) bool
}

// DefaultTimeoutConfig default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: 30 * time.Second,
		ErrorHandler: func(c *gin.Context) {
			utils.ErrorResponse(c, http.StatusRequestTimeout, "Request timeout")
		},
		SkipFunc: nil,
	}
}

// Timeout timeout middleware
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return TimeoutWithConfig(TimeoutConfig{
		Timeout: timeout,
	})
}

// TimeoutWithConfig timeout middleware with configuration
func TimeoutWithConfig(config TimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if timeout check should be skipped
		if config.SkipFunc != nil && config.SkipFunc(c) {
			c.Next()
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), config.Timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Create completion channel
		done := make(chan struct{})

		// Handle request in goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic
					if config.ErrorHandler != nil {
						config.ErrorHandler(c)
					} else {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "Internal server error",
						})
					}
				}
				close(done)
			}()
			c.Next()
		}()

		// Wait for request completion or timeout
		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Request timeout
			if config.ErrorHandler != nil {
				config.ErrorHandler(c)
			} else {
				c.JSON(http.StatusRequestTimeout, gin.H{
					"error":   "Request timeout",
					"timeout": config.Timeout.String(),
				})
			}
			c.Abort()
		}
	}
}

// SeckillTimeout seckill timeout middleware
func SeckillTimeout(timeout time.Duration) gin.HandlerFunc {
	config := DefaultTimeoutConfig()
	config.Timeout = timeout
	config.ErrorHandler = func(c *gin.Context) {
		utils.ErrorResponse(c, http.StatusRequestTimeout, "Seckill request timeout, please retry")
	}
	return TimeoutWithConfig(config)
}

// APITimeout API timeout middleware
func APITimeout(timeout time.Duration) gin.HandlerFunc {
	config := DefaultTimeoutConfig()
	config.Timeout = timeout
	config.ErrorHandler = func(c *gin.Context) {
		utils.ErrorResponse(c, http.StatusRequestTimeout, "API request timeout")
	}
	return TimeoutWithConfig(config)
}