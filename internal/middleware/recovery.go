package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"seckill/pkg/log"
	"seckill/pkg/utils"
)

// Recovery panic recovery middleware
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Log panic information
		log.WithFields(map[string]interface{}{
			"error":  recovered,
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
			"stack":  string(debug.Stack()),
		}).Error("Panic recovered")

		// Return error response
		utils.Error(c, utils.CodeInternalError, "Internal server error")
	})
}

// RecoveryWithWriter panic recovery middleware with custom writer
func RecoveryWithWriter(writer gin.RecoveryFunc) gin.HandlerFunc {
	return gin.CustomRecovery(writer)
}

// DefaultRecoveryFunc default recovery function
func DefaultRecoveryFunc(c *gin.Context, err interface{}) {
	// Log error
	log.WithFields(map[string]interface{}{
		"error":      err,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}).Error("Panic recovered")

	// Return 500 error
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    utils.CodeInternalError,
		"message": "Internal server error",
		"data":    nil,
	})
	c.Abort()
}

// RecoveryWithLogger panic recovery middleware with logger
func RecoveryWithLogger() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Format error message
		errMsg := fmt.Sprintf("Panic recovered: %v", recovered)
		
		// Log detailed information
		log.WithFields(map[string]interface{}{
			"error":      errMsg,
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"stack":      string(debug.Stack()),
		}).Error("Panic recovered with logger")

		// Return error response
		utils.Error(c, utils.CodeInternalError, "Internal server error")
	})
}