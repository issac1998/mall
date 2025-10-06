package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"seckill/internal/model"
	"seckill/internal/service/auth"
	"seckill/internal/utils"
)

// MockAuthService is a mock implementation of AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req *auth.RegisterRequest) (*model.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req *auth.LoginRequest, ip string) (*auth.TokenResponse, error) {
	args := m.Called(ctx, req, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenResponse), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, userID int64, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenResponse), args.Error(1)
}

func (m *MockAuthService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	return args.Error(0)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*utils.JWTClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*utils.JWTClaims), args.Error(1)
}

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("successful registration", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewAuthHandler(mockService)
		
		router := gin.New()
		router.POST("/register", handler.Register)

		req := auth.RegisterRequest{
			Username: "testuser",
			Password: "password123",
			Phone:    "13800138000",
		}

		user := &model.User{
			ID:       1,
			Username: "testuser",
		}

		mockService.On("Register", mock.Anything, mock.MatchedBy(func(r *auth.RegisterRequest) bool {
			return r.Username == "testuser" && r.Password == "password123" && r.Phone == "13800138000"
		})).Return(user, nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewAuthHandler(mockService)
		
		router := gin.New()
		router.POST("/register", handler.Register)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer([]byte("invalid json")))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("successful login", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewAuthHandler(mockService)
		
		router := gin.New()
		router.POST("/login", handler.Login)

		req := auth.LoginRequest{
			Account:  "testuser",
			Password: "password123",
		}

		tokenResp := &auth.TokenResponse{
			AccessToken:  "access_token",
			RefreshToken: "refresh_token",
			ExpiresIn:    3600,
		}

		mockService.On("Login", mock.Anything, mock.MatchedBy(func(r *auth.LoginRequest) bool {
			return r.Account == "testuser" && r.Password == "password123"
		}), mock.AnythingOfType("string")).Return(tokenResp, nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("successful logout", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewAuthHandler(mockService)
		
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", int64(1))
			c.Set("token", "test_token")
			c.Next()
		})
		router.POST("/logout", handler.Logout)

		mockService.On("Logout", mock.Anything, int64(1), "test_token").Return(nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/logout", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("successful refresh", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewAuthHandler(mockService)
		
		router := gin.New()
		router.POST("/refresh", handler.RefreshToken)

		req := struct {
			RefreshToken string `json:"refresh_token"`
		}{
			RefreshToken: "refresh_token",
		}

		tokenResp := &auth.TokenResponse{
			AccessToken:  "new_access_token",
			RefreshToken: "new_refresh_token",
			ExpiresIn:    3600,
		}

		mockService.On("RefreshToken", mock.Anything, "refresh_token").Return(tokenResp, nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("successful password change", func(t *testing.T) {
		mockService := &MockAuthService{}
		handler := NewAuthHandler(mockService)
		
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", int64(1))
			c.Next()
		})
		router.POST("/change-password", handler.ChangePassword)

		req := struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}{
			OldPassword: "oldpassword",
			NewPassword: "newpassword123",
		}

		mockService.On("ChangePassword", mock.Anything, int64(1), "oldpassword", "newpassword123").Return(nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/change-password", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}