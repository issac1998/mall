package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Response standard response structure
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// SuccessResponse returns success response
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// ErrorResponse returns error response
func ErrorResponse(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, Response{
		Code:      httpCode,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}

// FailedResponse returns failed response with custom code
func FailedResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      1,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// PageResponse page response structure
type PageResponse struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// SuccessPageResponse returns success page response
func SuccessPageResponse(c *gin.Context, list interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PageResponse{
			List:  list,
			Total: total,
			Page:  page,
			Size:  size,
		},
		Timestamp: time.Now().Unix(),
	})
}
