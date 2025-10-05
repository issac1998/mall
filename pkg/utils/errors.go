package utils

import "fmt"

// ResponseCode represents response code type
type ResponseCode int

// Response codes
const (
	CodeSuccess         ResponseCode = 0
	CodeInvalidParam    ResponseCode = 1001
	CodeDatabaseError   ResponseCode = 1002
	CodeServiceError    ResponseCode = 1003
	CodeInternalError   ResponseCode = 1004
	CodeBadRequest      ResponseCode = 1005
	CodeUnauthorized    ResponseCode = 1006
	CodeForbidden       ResponseCode = 1007
	CodeNotFound        ResponseCode = 1008
	CodeConflict        ResponseCode = 1009
	CodeTooManyRequests ResponseCode = 1010
)

// ResponseMessage maps response codes to messages
var ResponseMessage = map[ResponseCode]string{
	CodeSuccess:         "success",
	CodeInvalidParam:    "invalid parameter",
	CodeDatabaseError:   "database error",
	CodeServiceError:    "service error",
	CodeInternalError:   "internal error",
	CodeBadRequest:      "bad request",
	CodeUnauthorized:    "unauthorized",
	CodeForbidden:       "forbidden",
	CodeNotFound:        "not found",
	CodeConflict:        "conflict",
	CodeTooManyRequests: "too many requests",
}

// AppError represents application error
type AppError struct {
	Code    ResponseCode `json:"code"`
	Message string       `json:"message"`
	Err     error        `json:"-"`
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code: %d, message: %s, error: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// NewError creates a new application error
func NewError(code ResponseCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithErr creates a new application error with underlying error
func NewErrorWithErr(code ResponseCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError wraps an error with application error
func WrapError(err error, code ResponseCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsAppError checks if error is an application error
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// GetErrorCode gets error code from error
func GetErrorCode(err error) ResponseCode {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Code
	}
	return CodeInternalError
}

// GetErrorMessage gets error message from error
func GetErrorMessage(err error) string {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Message
	}
	return err.Error()
}

// getHTTPStatus maps response code to HTTP status code
func getHTTPStatus(code ResponseCode) int {
	switch code {
	case CodeSuccess:
		return 200
	case CodeInvalidParam, CodeBadRequest:
		return 400
	case CodeUnauthorized:
		return 401
	case CodeForbidden:
		return 403
	case CodeNotFound:
		return 404
	case CodeConflict:
		return 409
	case CodeTooManyRequests:
		return 429
	case CodeDatabaseError, CodeServiceError, CodeInternalError:
		return 500
	default:
		return 500
	}
}