package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"seckill/pkg/utils"
)

const (
	// AuthorizationHeader 认证头部名称
	AuthorizationHeader = "Authorization"
	// BearerPrefix Bearer前缀
	BearerPrefix = "Bearer "
	// UserIDKey 用户ID在上下文中的键
	UserIDKey = "user_id"
	// UserRoleKey 用户角色在上下文中的键
	UserRoleKey = "user_role"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// TokenValidator Token验证函数
	TokenValidator func(token string) (*UserInfo, error)
	// SkipPaths 跳过认证的路径
	SkipPaths []string
	// RequiredRole 需要的角色
	RequiredRole string
}

// UserInfo 用户信息
type UserInfo struct {
	ID   int64  `json:"id"`
	Role string `json:"role"`
}

// Auth 认证中间件
func Auth(validator func(token string) (*UserInfo, error)) gin.HandlerFunc {
	return AuthWithConfig(AuthConfig{
		TokenValidator: validator,
	})
}

// AuthWithConfig 带配置的认证中间件
func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// 检查是否跳过认证
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 获取Authorization头部
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			utils.Error(c, utils.CodeUnauthorized, "Missing authorization header")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			utils.Error(c, utils.CodeUnauthorized, "Invalid authorization header format")
			c.Abort()
			return
		}

		// 提取token
		token := strings.TrimPrefix(authHeader, BearerPrefix)
		if token == "" {
			utils.Error(c, utils.CodeUnauthorized, "Missing token")
			c.Abort()
			return
		}

		// 验证token
		userInfo, err := config.TokenValidator(token)
		if err != nil {
			utils.Error(c, utils.CodeUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		// 检查角色权限
		if config.RequiredRole != "" && userInfo.Role != config.RequiredRole {
			utils.Error(c, utils.CodeForbidden, "Insufficient permissions")
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set(UserIDKey, userInfo.ID)
		c.Set(UserRoleKey, userInfo.Role)

		c.Next()
	}
}

// RequireAuth 需要认证的中间件
func RequireAuth(validator func(token string) (*UserInfo, error)) gin.HandlerFunc {
	return Auth(validator)
}

// RequireRole 需要特定角色的中间件
func RequireRole(validator func(token string) (*UserInfo, error), role string) gin.HandlerFunc {
	return AuthWithConfig(AuthConfig{
		TokenValidator: validator,
		RequiredRole:   role,
	})
}

// OptionalAuth 可选认证中间件
func OptionalAuth(validator func(token string) (*UserInfo, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, BearerPrefix)
		if token == "" {
			c.Next()
			return
		}

		userInfo, err := validator(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set(UserIDKey, userInfo.ID)
		c.Set(UserRoleKey, userInfo.Role)
		c.Next()
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	
	switch v := userID.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return id, true
	default:
		return 0, false
	}
}

// GetUserRole 从上下文获取用户角色
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(UserRoleKey)
	if !exists {
		return "", false
	}
	
	if roleStr, ok := role.(string); ok {
		return roleStr, true
	}
	return "", false
}

// MustGetUserID 从上下文获取用户ID（必须存在）
func MustGetUserID(c *gin.Context) int64 {
	userID, exists := GetUserID(c)
	if !exists {
		panic("user ID not found in context")
	}
	return userID
}

// MustGetUserRole 从上下文获取用户角色（必须存在）
func MustGetUserRole(c *gin.Context) string {
	role, exists := GetUserRole(c)
	if !exists {
		panic("user role not found in context")
	}
	return role
}