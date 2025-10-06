package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"seckill/internal/repository"
	"seckill/pkg/utils"
)

// ActivityHandler activity handler
type ActivityHandler struct {
	activityRepo repository.ActivityRepository
}

// NewActivityHandler creates an activity handler
func NewActivityHandler(activityRepo repository.ActivityRepository) *ActivityHandler {
	return &ActivityHandler{
		activityRepo: activityRepo,
	}
}

// ListActivities lists active activities
func (h *ActivityHandler) ListActivities(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	activities, total, err := h.activityRepo.ListActive(c.Request.Context(), page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"list":  activities,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetActivity gets activity by ID
func (h *ActivityHandler) GetActivity(c *gin.Context) {
	activityIDStr := c.Param("id")
	activityID, err := strconv.ParseInt(activityIDStr, 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid activity ID")
		return
	}

	activity, err := h.activityRepo.GetByID(c.Request.Context(), activityID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Activity not found")
		return
	}

	utils.SuccessResponse(c, activity)
}