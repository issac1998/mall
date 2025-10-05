package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"seckill/internal/service/stock"
	"seckill/pkg/utils"
)

// StockHandler stock handler
type StockHandler struct {
	stockService stock.StockService
}

// NewStockHandler creates a stock handler
func NewStockHandler(stockService stock.StockService) *StockHandler {
	return &StockHandler{
		stockService: stockService,
	}
}

// SyncToRedis sync stock to Redis
func (h *StockHandler) SyncToRedis(c *gin.Context) {
	activityIDStr := c.Param("id")
	activityID, err := strconv.ParseUint(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	if err := h.stockService.SyncStockToRedis(c.Request.Context(), activityID); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "Stock synced to Redis successfully"})
}

// SyncToMySQL sync stock to MySQL
func (h *StockHandler) SyncToMySQL(c *gin.Context) {
	activityIDStr := c.Param("id")
	activityID, err := strconv.ParseUint(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	if err := h.stockService.SyncStockToMySQL(c.Request.Context(), activityID); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "Stock synced to MySQL successfully"})
}

// CheckConsistency check stock consistency
func (h *StockHandler) CheckConsistency(c *gin.Context) {
	activityIDStr := c.Param("id")
	activityID, err := strconv.ParseUint(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	report, err := h.stockService.CheckStockConsistency(c.Request.Context(), activityID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, report)
}

// RepairInconsistency repair stock inconsistency
func (h *StockHandler) RepairInconsistency(c *gin.Context) {
	activityIDStr := c.Param("id")
	activityID, err := strconv.ParseUint(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	if err := h.stockService.RepairStockInconsistency(c.Request.Context(), activityID); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "Stock inconsistency repaired successfully"})
}

