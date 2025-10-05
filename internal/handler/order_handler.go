package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"seckill/internal/service/order"
	"seckill/pkg/utils"
)

// OrderHandler order handler
type OrderHandler struct {
	orderService order.OrderService
}

// NewOrderHandler creates an order handler
func NewOrderHandler(orderService order.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// GetOrder gets an order by order number
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Missing order_no parameter")
		return
	}

	order, err := h.orderService.GetOrderByOrderNo(c.Request.Context(), orderNo)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, order)
}

// ListOrders lists user orders
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	orders, total, err := h.orderService.ListUserOrders(
		c.Request.Context(),
		uint64(userID.(int64)),
		page,
		pageSize,
	)

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"list":  orders,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// PayOrder pays an order
func (h *OrderHandler) PayOrder(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Missing order_no parameter")
		return
	}

	if err := h.orderService.PayOrder(c.Request.Context(), orderNo); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Payment failed: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Payment successful")
}

