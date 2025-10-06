package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"seckill/internal/model"
	"seckill/internal/repository"
	"seckill/internal/utils"
	"seckill/pkg/log"
)

// RegisterRequest register request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Phone    string `json:"phone" binding:"required,len=11"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

// LoginRequest login request
type LoginRequest struct {
	Account  string `json:"account" binding:"required"` // username/phone/email
	Password string `json:"password" binding:"required"`
}

// TokenResponse token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// AuthService authentication service interface
type AuthService interface {
	// Register user
	Register(ctx context.Context, req *RegisterRequest) (*model.User, error)

	// Login user
	Login(ctx context.Context, req *LoginRequest, ip string) (*TokenResponse, error)

	// Logout user
	Logout(ctx context.Context, userID int64, token string) error

	// Validate token
	ValidateToken(ctx context.Context, token string) (*utils.JWTClaims, error)

	// Refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)

	// Change password
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error
}

// authService authentication service implementation
type authService struct {
	userRepo   repository.UserRepository
	jwtManager *utils.JWTManager
	redis      *redis.Client
}

// NewAuthService creates an authentication service
func NewAuthService(
	userRepo repository.UserRepository,
	jwtManager *utils.JWTManager,
	redis *redis.Client,
) AuthService {
	return &authService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		redis:      redis,
	}
}

// Register registers a user
func (s *authService) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	log.Info("user register", "username", req.Username)

	// 1. Check if username exists
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		log.Error("check username failed", "error", err)
		return nil, errors.New("system error")
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	// 2. Check if phone exists
	exists, err = s.userRepo.ExistsByPhone(ctx, req.Phone)
	if err != nil {
		log.Error("check phone failed", "error", err)
		return nil, errors.New("system error")
	}
	if exists {
		return nil, errors.New("phone already registered")
	}

	// 3. Generate salt
	salt, err := s.generateSalt()
	if err != nil {
		log.Error("generate salt failed", "error", err)
		return nil, errors.New("system error")
	}

	// 4. Hash password
	passwordHash, err := s.hashPassword(req.Password + salt)
	if err != nil {
		log.Error("hash password failed", "error", err)
		return nil, errors.New("system error")
	}

	// 5. Create user
	nickname := req.Username
	var email *string
	if req.Email != "" {
		email = &req.Email
	}
	user := &model.User{
		Username:     req.Username,
		Phone:        req.Phone,
		Email:        email,
		PasswordHash: passwordHash,
		Salt:         salt,
		Nickname:     &nickname,
		Level:        1,
		Status:       1,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		log.Error("create user failed", "error", err)
		return nil, errors.New("registration failed")
	}

	log.Info("user register success", "user_id", user.ID, "username", user.Username)
	return user, nil
}

// Login logs in a user
func (s *authService) Login(ctx context.Context, req *LoginRequest, ip string) (*TokenResponse, error) {
	log.Info("user login", "account", req.Account, "ip", ip)

	// 1. Find user (support username/phone/email, each one can satisfied)
	user, err := s.findUserByAccount(ctx, req.Account)
	if err != nil {
		log.Warn("user not found", "account", req.Account)
		return nil, errors.New("username or password incorrect")
	}

	// 2. Check user status, later can add more status check
	if !user.IsActive() {
		return nil, errors.New("account disabled")
	}

	userID := int64(user.ID)

	// 3. Check login attempts
	if err := s.checkLoginAttempts(ctx, userID); err != nil {
		return nil, err
	}

	// 4. Verify password
	if !s.verifyPassword(req.Password+user.Salt, user.PasswordHash) {
		// Record login failure to Redis
		s.recordLoginFailure(ctx, userID)
		return nil, errors.New("username or password incorrect")
	}

	// 5. Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(userID, user.Username, "user")
	if err != nil {
		log.Error("generate access token failed", "error", err)
		return nil, errors.New("system error")
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID, user.Username)
	if err != nil {
		log.Error("generate refresh token failed", "error", err)
		return nil, errors.New("system error")
	}

	// 6. Save token to Redis
	tokenKey := fmt.Sprintf("auth:token:%d", userID)
	s.redis.Set(ctx, tokenKey, accessToken, 2*time.Hour)

	// 7. Update last login info
	s.userRepo.UpdateLastLogin(ctx, userID, ip)

	// 8. Clear login failures
	s.clearLoginFailures(ctx, userID)

	log.Info("user login success", "user_id", userID, "username", user.Username)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200, // 2 hours
		TokenType:    "Bearer",
	}, nil
}

// Logout logs out a user
func (s *authService) Logout(ctx context.Context, userID int64, token string) error {
	// 1. Delete token from Redis
	tokenKey := fmt.Sprintf("auth:token:%d", userID)
	s.redis.Del(ctx, tokenKey)

	// 2. Add token to blacklist, aviod reuse until it expires
	blacklistKey := fmt.Sprintf("auth:blacklist:%s", token)
	s.redis.Set(ctx, blacklistKey, "1", 2*time.Hour)

	log.Info("user logout", "user_id", userID)
	return nil
}

// ValidateToken validates a token
func (s *authService) ValidateToken(ctx context.Context, token string) (*utils.JWTClaims, error) {
	// 1. Check if token is in blacklist
	blacklistKey := fmt.Sprintf("auth:blacklist:%s", token)
	exists, _ := s.redis.Exists(ctx, blacklistKey).Result()
	if exists > 0 {
		return nil, errors.New("token invalid")
	}

	// 2. Validate token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// 3. Check if token exists in Redis
	tokenKey := fmt.Sprintf("auth:token:%d", claims.UserID)
	storedToken, err := s.redis.Get(ctx, tokenKey).Result()
	if err != nil || storedToken != token {
		return nil, errors.New("token invalid")
	}

	return claims, nil
}

// RefreshToken refreshes a token
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// 1. Validate refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("refresh token invalid")
	}

	// 2. Generate new access token
	accessToken, err := s.jwtManager.GenerateAccessToken(claims.UserID, claims.Username, claims.Role)
	if err != nil {
		return nil, errors.New("generate token failed")
	}

	// 3. Update token in Redis
	tokenKey := fmt.Sprintf("auth:token:%d", claims.UserID)
	s.redis.Set(ctx, tokenKey, accessToken, 2*time.Hour)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200,
		TokenType:    "Bearer",
	}, nil
}

// ChangePassword changes user password
func (s *authService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	// 1. Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 2. Verify old password
	if !s.verifyPassword(oldPassword+user.Salt, user.PasswordHash) {
		return errors.New("old password incorrect")
	}

	// 3. Generate new salt
	salt, err := s.generateSalt()
	if err != nil {
		return errors.New("system error")
	}

	// 4. Hash new password
	newPasswordHash, err := s.hashPassword(newPassword + salt)
	if err != nil {
		return errors.New("system error")
	}

	// 5. Update user password
	user.PasswordHash = newPasswordHash
	user.Salt = salt

	if err := s.userRepo.Update(ctx, user); err != nil {
		return errors.New("change password failed")
	}

	log.Info("user changed password", "user_id", userID)
	return nil
}

// Helper methods

// findUserByAccount finds a user by account (username/phone/email)
func (s *authService) findUserByAccount(ctx context.Context, account string) (*model.User, error) {
	// Try username
	user, err := s.userRepo.GetByUsername(ctx, account)
	if err == nil {
		return user, nil
	}

	// Try phone
	user, err = s.userRepo.GetByPhone(ctx, account)
	if err == nil {
		return user, nil
	}

	// Try email
	user, err = s.userRepo.GetByEmail(ctx, account)
	if err == nil {
		return user, nil
	}

	return nil, errors.New("user not found")
}

// generateSalt generates a salt
func (s *authService) generateSalt() (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(salt), nil
}

// hashPassword hashes a password
func (s *authService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword verifies a password
func (s *authService) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// checkLoginAttempts checks login attempts
func (s *authService) checkLoginAttempts(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("auth:login_attempts:%d", userID)
	attempts, _ := s.redis.Get(ctx, key).Int()

	if attempts >= 5 {
		return errors.New("login failed too many times, please try again in 30 minutes")
	}

	return nil
}

// recordLoginFailure records a login failure
func (s *authService) recordLoginFailure(ctx context.Context, userID int64) {
	key := fmt.Sprintf("auth:login_attempts:%d", userID)
	s.redis.Incr(ctx, key)
	s.redis.Expire(ctx, key, 30*time.Minute)
}

// clearLoginFailures clears login failures
func (s *authService) clearLoginFailures(ctx context.Context, userID int64) {
	key := fmt.Sprintf("auth:login_attempts:%d", userID)
	s.redis.Del(ctx, key)
}

