package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS Cross-Origin Resource Sharing middleware
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	
	// Allow all origins in development environment
	config.AllowAllOrigins = true
	
	// Allow common headers
	config.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"X-Requested-With",
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Connection",
		"Host",
		"Referer",
		"User-Agent",
		"X-Real-IP",
		"X-Forwarded-For",
		"X-Forwarded-Proto",
	}
	
	// Allow common methods
	config.AllowMethods = []string{
		"GET",
		"POST",
		"PUT",
		"PATCH",
		"DELETE",
		"HEAD",
		"OPTIONS",
	}
	
	// Allow credentials
	config.AllowCredentials = true
	
	return cors.New(config)
}

// CORSWithConfig CORS middleware with custom configuration
func CORSWithConfig(config cors.Config) gin.HandlerFunc {
	return cors.New(config)
}