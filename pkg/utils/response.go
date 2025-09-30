package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response unified response structure
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ResponseCode response code definition
type ResponseCode int

const (
	// Success response code
	CodeSuccess ResponseCode = 200

	// Client error response codes
	CodeBadRequest     ResponseCode = 400
	CodeUnauthorized   ResponseCode = 401
	CodeForbidden      ResponseCode = 403
	CodeNotFound       ResponseCode = 404
	CodeTooManyRequest ResponseCode = 429

	// Server error response codes
	CodeInternalError ResponseCode = 500
	CodeServiceError  ResponseCode = 501
	CodeDatabaseError ResponseCode = 502
	CodeRedisError    ResponseCode = 503

	// Business error response codes
	CodeInvalidParam    ResponseCode = 1001
	CodeUserNotFound    ResponseCode = 1002
	CodeProductNotFound ResponseCode = 1003
	CodeStockNotEnough  ResponseCode = 1004
	CodeOrderExists     ResponseCode = 1005
	CodeOrderNotFound   ResponseCode = 1006
	CodeSeckillNotStart ResponseCode = 1007
	CodeSeckillEnd      ResponseCode = 1008
	CodeRateLimit       ResponseCode = 1009
)

// ResponseMessage response message mapping
var ResponseMessage = map[ResponseCode]string{
	CodeSuccess: "success",

	CodeBadRequest:     "bad request",
	CodeUnauthorized:   "unauthorized",
	CodeForbidden:      "forbidden",
	CodeNotFound:       "not found",
	CodeTooManyRequest: "too many requests",

	CodeInternalError: "internal server error",
	CodeServiceError:  "service error",
	CodeDatabaseError: "database error",
	CodeRedisError:    "redis error",

	CodeInvalidParam:    "invalid parameter",
	CodeUserNotFound:    "user not found",
	CodeProductNotFound: "product not found",
	CodeStockNotEnough:  "stock not enough",
	CodeOrderExists:     "order already exists",
	CodeOrderNotFound:   "order not found",
	CodeSeckillNotStart: "seckill not started",
	CodeSeckillEnd:      "seckill ended",
	CodeRateLimit:       "rate limit exceeded",
}

// Success success response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    int(CodeSuccess),
		Message: ResponseMessage[CodeSuccess],
		Data:    data,
	})
}

// Error error response
func Error(c *gin.Context, code ResponseCode, message ...string) {
	msg := ResponseMessage[code]
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:    int(code),
		Message: msg,
	})
}

// ErrorWithData error response with data
func ErrorWithData(c *gin.Context, code ResponseCode, data interface{}, message ...string) {
	msg := ResponseMessage[code]
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:    int(code),
		Message: msg,
		Data:    data,
	})
}

// getHTTPStatus get HTTP status code based on business response code
func getHTTPStatus(code ResponseCode) int {
	switch {
	case code == CodeSuccess:
		return http.StatusOK
	case code >= 400 && code < 500:
		return int(code)
	case code >= 500 && code < 600:
		return int(code)
	case code == CodeTooManyRequest:
		return http.StatusTooManyRequests
	case code >= 1000 && code < 2000:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Pagination pagination response structure
type Pagination struct {
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Total    int64       `json:"total"`
	Data     interface{} `json:"data"`
}

// SuccessWithPagination pagination success response
func SuccessWithPagination(c *gin.Context, page, pageSize int, total int64, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    int(CodeSuccess),
		Message: ResponseMessage[CodeSuccess],
		Data: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Data:     data,
		},
	})
}