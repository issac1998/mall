package utils

import (
	"fmt"
)

// AppError application error structure
type AppError struct {
	Code    ResponseCode `json:"code"`
	Message string       `json:"message"`
	Err     error        `json:"-"`
}

// Error implement error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code: %d, message: %s, error: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// Unwrap implement errors.Unwrap interface
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewError create new application error
func NewError(code ResponseCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithErr create application error with original error
func NewErrorWithErr(code ResponseCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError wrap error
func WrapError(err error, code ResponseCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Predefined errors
var (
	// Parameter errors
	ErrInvalidParam = NewError(CodeInvalidParam, "invalid parameter")

	// User related errors
	ErrUserNotFound = NewError(CodeUserNotFound, "user not found")

	// Product related errors
	ErrProductNotFound = NewError(CodeProductNotFound, "product not found")
	ErrStockNotEnough  = NewError(CodeStockNotEnough, "stock not enough")

	// Order related errors
	ErrOrderExists   = NewError(CodeOrderExists, "order already exists")
	ErrOrderNotFound = NewError(CodeOrderNotFound, "order not found")

	// Seckill related errors
	ErrSeckillNotStart = NewError(CodeSeckillNotStart, "seckill not started")
	ErrSeckillEnd      = NewError(CodeSeckillEnd, "seckill ended")
	ErrRateLimit       = NewError(CodeRateLimit, "rate limit exceeded")

	// System errors
	ErrInternalError = NewError(CodeInternalError, "internal server error")
	ErrServiceError  = NewError(CodeServiceError, "service error")
	ErrDatabaseError = NewError(CodeDatabaseError, "database error")
	ErrRedisError    = NewError(CodeRedisError, "redis error")
)

// IsAppError check if it's an application error
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// GetErrorCode get error code
func GetErrorCode(err error) ResponseCode {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Code
	}
	return CodeInternalError
}

// GetErrorMessage get error message
func GetErrorMessage(err error) string {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Message
	}
	return err.Error()
}