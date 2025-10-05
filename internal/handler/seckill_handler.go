package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"seckill/internal/service/seckill"
	"seckill/pkg/utils"
)

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
	var req seckill.SeckillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid parameters: "+err.Error())
		return
	}

	// Get user ID from JWT middleware
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	req.UserID = uint64(userID.(int64))

	// Get client information
	req.IP = c.ClientIP()
	req.UserAgent = c.Request.UserAgent()

	result, err := h.seckillService.DoSeckill(c.Request.Context(), &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Seckill failed: "+err.Error())
		return
	}

	if result.Success {
		utils.SuccessResponse(c, result)
	} else {
		utils.FailedResponse(c, result.Message, result)
	}
}

// QueryResult queries seckill result
func (h *SeckillHandler) QueryResult(c *gin.Context) {
	requestID := c.Query("request_id")
	if requestID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Missing request_id parameter")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	result, err := h.seckillService.QuerySeckillResult(
		c.Request.Context(),
		requestID,
		uint64(userID.(int64)),
	)

	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, result)
}

// PrewarmActivity prewarms activity
func (h *SeckillHandler) PrewarmActivity(c *gin.Context) {
	activityIDStr := c.Param("activity_id")
	activityID, err := strconv.ParseUint(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	if err := h.seckillService.PrewarmActivity(c.Request.Context(), activityID); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Prewarm failed: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Prewarm successful")
}

