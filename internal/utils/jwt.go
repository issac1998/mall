package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT claims
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager JWT manager
type JWTManager struct {
	secretKey     []byte
	issuer        string
	accessExpire  time.Duration
	refreshExpire time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey, issuer string, accessExpire, refreshExpire time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		issuer:        issuer,
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}
}

// GenerateAccessToken generates an access token
func (m *JWTManager) GenerateAccessToken(userID int64, username, role string) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpire)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GenerateRefreshToken generates a refresh token
func (m *JWTManager) GenerateRefreshToken(userID int64, username string) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpire)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken validates a token
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshAccessToken refreshes an access token
func (m *JWTManager) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	return m.GenerateAccessToken(claims.UserID, claims.Username, claims.Role)
}

