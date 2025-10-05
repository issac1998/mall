package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"seckill/internal/service/auth"
	"seckill/pkg/utils"
)

// AuthHandler authentication handler
type AuthHandler struct {
	authService auth.AuthService
}

// NewAuthHandler creates an authentication handler
func NewAuthHandler(authService auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid parameters: "+err.Error())
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"user_id":  user.ID,
		"username": user.Username,
	})
}

// Login user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid parameters: "+err.Error())
		return
	}

	ip := c.ClientIP()
	tokenResp, err := h.authService.Login(c.Request.Context(), &req, ip)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SuccessResponse(c, tokenResp)
}

// Logout user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetInt64("user_id")
	token := c.GetString("token")

	err := h.authService.Logout(c.Request.Context(), userID, token)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "logout failed")
		return
	}

	utils.SuccessResponse(c, nil)
}

// RefreshToken refreshes token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	tokenResp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SuccessResponse(c, tokenResp)
}

// ChangePassword changes password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	userID := c.GetInt64("user_id")
	err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, nil)
}

