package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"seckill/internal/service/seckill"
	"seckill/pkg/utils"
)

// SeckillAPIRequest API request structure for seckill
type SeckillAPIRequest struct {
	RequestID  string `json:"request_id" binding:"required"`
	ActivityID uint64 `json:"activity_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	DeviceID   string `json:"device_id"`
}

// SeckillHandler seckill handler
type SeckillHandler struct {
	seckillService seckill.SeckillService
}

// NewSeckillHandler creates a seckill handler
func NewSeckillHandler(seckillService seckill.SeckillService) *SeckillHandler {
	return &SeckillHandler{
		seckillService: seckillService,
	}
}

// DoSeckill executes seckill
func (h *SeckillHandler) DoSeckill(c *gin.Context) {
	var apiReq SeckillAPIRequest
	if err := c.ShouldBindJSON(&apiReq); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid parameters: "+err.Error())
		return
	}

	// Get user ID from JWT middleware
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Convert API request to service request
	req := &seckill.SeckillRequest{
		RequestID:  apiReq.RequestID,
		ActivityID: apiReq.ActivityID,
		UserID:     uint64(userID.(int64)),
		Quantity:   apiReq.Quantity,
		IP:         c.ClientIP(),
		DeviceID:   apiReq.DeviceID,
		UserAgent:  c.Request.UserAgent(),
	}

	result, err := h.seckillService.DoSeckill(c.Request.Context(), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Seckill failed: "+err.Error())
		return
	}

	if result.Success {
		utils.SuccessResponse(c, result)
	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, result.Message)
	}
}

// QueryResult queries seckill result
func (h *SeckillHandler) QueryResult(c *gin.Context) {
	requestID := c.Param("request_id")
	if requestID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Request ID is required")
		return
	}

	// Get user ID from JWT middleware
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	result, err := h.seckillService.QuerySeckillResult(c.Request.Context(), requestID, uint64(userID.(int64)))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Query failed: "+err.Error())
		return
	}

	utils.SuccessResponse(c, result)
}

// PrewarmActivity prewarms activity cache
func (h *SeckillHandler) PrewarmActivity(c *gin.Context) {
	activityIDStr := c.Param("activity_id")
	activityID, err := strconv.ParseUint(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	err = h.seckillService.PrewarmActivity(c.Request.Context(), activityID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Prewarm failed: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "Activity prewarmed successfully"})
}

