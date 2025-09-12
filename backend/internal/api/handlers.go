package api

import (
	"net/http"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"

	errors2 "github.com/makhkets/wildberries-l0/internal/errors"
	"github.com/makhkets/wildberries-l0/internal/model"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// GetOrderByUID GET /orders/:uid
func (h *Handler) GetOrderByUID(c *gin.Context) {
	uid := c.Param("uid")
	order, err := h.services.GetOrderByUID(c.Request.Context(), uid)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: order,
	})

	slog.Info("Order retrieved via API",
		"uid", uid,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"client_ip", c.ClientIP())
}

// CreateOrder POST /orders
func (h *Handler) CreateOrder(c *gin.Context) {
	// Парсим JSON из запроса
	var order model.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		slog.Warn("Failed to decode request body", "error", err)
		h.handleError(c, errors2.NewValidationError("request_body", "invalid JSON format: "+err.Error()))
		return
	}

	err := h.services.CreateOrder(c.Request.Context(), &order)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    order,
		Message: "Order created successfully",
	})

	slog.Info("Order created via API",
		"uid", order.OrderUID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"client_ip", c.ClientIP())
}

// handleError обрабатывает ошибки и возвращает соответствующий HTTP ответ
func (h *Handler) handleError(c *gin.Context, err error) {
	// Получаем структурированную ошибку
	var appErr *errors2.AppError
	if !errors2.IsAppError(err, &appErr) {
		// Если это не наша структурированная ошибка, создаем внутреннюю ошибку
		appErr = errors2.NewAppError(errors2.ErrorTypeInternal, "Internal server error")
		slog.Error("Unhandled error", "error", err, "path", c.Request.URL.Path)
	}

	// Логируем ошибку с соответствующим уровнем
	h.logError(appErr, c)

	// Подготавливаем ответ для клиента
	errorResponse := ErrorResponse{
		Error:   string(appErr.Type),
		Message: appErr.Message,
		Details: appErr.Details,
	}

	// Для внутренних ошибок скрываем детали
	if appErr.Type == errors2.ErrorTypeInternal {
		errorResponse.Details = "" // Не показываем внутренние детали пользователю
	}

	// Используем Gin для отправки JSON ответа
	c.JSON(appErr.StatusCode, errorResponse)
}

// logError логирует ошибку с соответствующим уровнем
func (h *Handler) logError(appErr *errors2.AppError, c *gin.Context) {
	logAttrs := []any{
		"error_type", appErr.Type,
		"message", appErr.Message,
		"status_code", appErr.StatusCode,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"user_agent", c.Request.UserAgent(),
		"client_ip", c.ClientIP(),
	}

	switch appErr.Type {
	case errors2.ErrorTypeValidation, errors2.ErrorTypeNotFound:
		// Ошибки валидации и "не найдено" - это обычные случаи
		slog.Info("Client error", logAttrs...)
	case errors2.ErrorTypeUnauthorized, errors2.ErrorTypeForbidden:
		// Ошибки авторизации требуют внимания
		slog.Warn("Authorization error", logAttrs...)
	case errors2.ErrorTypeConflict:
		// Конфликты - тоже обычные случаи
		slog.Info("Business logic conflict", logAttrs...)
	default:
		// Все остальные ошибки - серьезные проблемы
		if appErr.Internal != nil {
			logAttrs = append(logAttrs, "internal_error", appErr.Internal.Error())
		}
		slog.Error("Internal server error", logAttrs...)
	}
}

// HealthCheck обрабатывает GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "orders-api",
		"timestamp": gin.H{
			"unix": time.Now().UnixNano(),
		},
	})
}

// CORSMiddleware добавляет CORS заголовки
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-User-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware логирует все входящие запросы
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		slog.Info("HTTP Request",
			"method", param.Method,
			"status", param.StatusCode,
			"latency", param.Latency,
			"client_ip", param.ClientIP,
		)
		return ""
	})
}
