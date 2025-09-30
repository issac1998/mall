package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"seckill/pkg/log"
)

// Logger request logging middleware
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Log request information
		log.WithFields(map[string]interface{}{
			"status":     param.StatusCode,
			"method":     param.Method,
			"path":       param.Path,
			"ip":         param.ClientIP,
			"user_agent": param.Request.UserAgent(),
			"latency":    param.Latency,
			"time":       param.TimeStamp.Format(time.RFC3339),
		}).Info("Request processed")
		
		return ""
	})
}

// LoggerWithConfig request logging middleware with configuration
func LoggerWithConfig(config gin.LoggerConfig) gin.HandlerFunc {
	return gin.LoggerWithConfig(config)
}

// CustomLogger custom request logging middleware
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate processing time
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		// Build complete path
		if raw != "" {
			path = path + "?" + raw
		}

		// Log request information
		fields := map[string]interface{}{
			"status":     statusCode,
			"method":     method,
			"path":       path,
			"ip":         clientIP,
			"user_agent": c.Request.UserAgent(),
			"latency":    latency,
		}

		// Add error information if exists
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Log based on status code
		if statusCode >= 500 {
			log.WithFields(fields).Error("Server error")
		} else if statusCode >= 400 {
			log.WithFields(fields).Warn("Client error")
		} else {
			log.WithFields(fields).Info("Request completed")
		}
	}
}