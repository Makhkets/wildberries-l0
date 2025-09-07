package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/makhkets/wildberries-l0/internal/service"
)

// Handler структура для HTTP хендлеров
type Handler struct {
	services service.Order
}

// NewHandler создает новый экземпляр Handler
func NewHandler(services service.Order) *Handler {
	return &Handler{
		services: services,
	}
}

// InitRoutes инициализирует маршруты
func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(h.corsMiddleware())

	// Health check
	router.GET("/health", h.healthCheck)

	// API группа
	api := router.Group("/api/v1")
	{
		// Группа заказов
		orders := api.Group("/orders")
		{
			orders.GET("/:uid", h.getOrderByUID)
			orders.POST("/", h.createOrder)
			orders.GET("/", h.getAllOrders)
		}
	}

	return router
}

// corsMiddleware добавляет CORS заголовки
func (h *Handler) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
