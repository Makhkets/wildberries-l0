package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/makhkets/wildberries-l0/internal/service"
)

// Handler содержит все зависимости для API handlers
type Handler struct {
	services service.Order
}

// NewHandler создает новый экземпляр Handler
func NewHandler(services service.Order) *Handler {
	return &Handler{
		services: services,
	}
}

// InitRoutes настраивает и возвращает Gin router с настроенными маршрутами
func (h *Handler) InitRoutes() *gin.Engine {
	// Создаем Gin router
	router := gin.New()

	// Middleware
	router.Use(LoggingMiddleware())
	router.Use(CORSMiddleware())
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", h.HealthCheck)

	// API v1 группа
	v1 := router.Group("/api/v1")
	{

		// Orders routes
		orders := v1.Group("/order")
		{
			orders.POST("", h.CreateOrder)       // POST /api/v1/orders
			orders.GET("/:uid", h.GetOrderByUID) // GET /api/v1/orders/{uid}
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
