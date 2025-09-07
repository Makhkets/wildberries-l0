package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/makhkets/wildberries-l0/internal/model"
)

// getOrderByUID получает заказ по UID
func (h *Handler) getOrderByUID(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uid parameter is required"})
		return
	}

	//order, err := h.services.Order.GetOrderByUID(c.Request.Context(), uid)
	//if err != nil {
	//	slog.Error("failed to get order", slog.String("uid", uid), slog.String("error", err.Error()))
	//	c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	//	return
	//}
	//
	//c.JSON(http.StatusOK, order)
}

// createOrder создает новый заказ
func (h *Handler) createOrder(c *gin.Context) {
	var order model.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		slog.Error("failed to bind JSON", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}

	//if err := h.services.Order.CreateOrder(c.Request.Context(), &order); err != nil {
	//	slog.Error("failed to create order", slog.String("uid", order.OrderUID), slog.String("error", err.Error()))
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
	//	return
	//}
	//
	//slog.Info("order created successfully", slog.String("uid", order.OrderUID))
	//c.JSON(http.StatusCreated, gin.H{"message": "order created successfully", "uid": order.OrderUID})
}

// getAllOrders получает все заказы
func (h *Handler) getAllOrders(c *gin.Context) {
	//orders, err := h.services.Order.GetAllOrders(c.Request.Context())
	//if err != nil {
	//	slog.Error("failed to get all orders", slog.String("error", err.Error()))
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get orders"})
	//	return
	//}
	//
	//c.JSON(http.StatusOK, gin.H{"orders": orders, "count": len(orders)})
}

// healthCheck проверяет состояние сервиса
func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "wildberries-order-service",
	})
}
