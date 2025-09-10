package api

import (
	"net/http"
	"strconv"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"

	errors2 "github.com/makhkets/wildberries-l0/internal/errors"
	"github.com/makhkets/wildberries-l0/internal/model"
)

// ErrorResponse представляет структуру ошибки в HTTP ответе
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse представляет успешный ответ
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// GetOrderByUID обрабатывает GET /orders/:uid
func (h *Handler) GetOrderByUID(c *gin.Context) {
	// Извлекаем UID из URL параметров
	uid := c.Param("uid")

	// Вызываем сервис
	order, err := h.services.GetOrderByUID(c.Request.Context(), uid)
	if err != nil {
		// Обрабатываем ошибку и возвращаем соответствующий HTTP ответ
		h.handleError(c, err)
		return
	}

	// Возвращаем успешный ответ
	c.JSON(http.StatusOK, SuccessResponse{
		Data:    order,
		Message: "Order retrieved successfully",
	})

	slog.Info("Order retrieved via API",
		"uid", uid,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"client_ip", c.ClientIP())
}

// CreateOrder обрабатывает POST /orders
func (h *Handler) CreateOrder(c *gin.Context) {
	// Парсим JSON из запроса
	var order model.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		slog.Warn("Failed to decode request body", "error", err)
		h.handleError(c, errors2.NewValidationError("request_body", "invalid JSON format: "+err.Error()))
		return
	}

	// Вызываем сервис
	err := h.services.CreateOrder(c.Request.Context(), &order)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Возвращаем успешный ответ
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

// getUserIDFromContext извлекает ID пользователя из Gin контекста
func (h *Handler) getUserIDFromContext(c *gin.Context) string {
	// В реальном приложении здесь была бы логика извлечения ID из JWT токена
	// который был установлен в middleware авторизации

	// Сначала пробуем получить из контекста Gin (установлено middleware)
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}

	// Fallback: используем заголовок (в продакшене так делать нельзя!)
	return c.GetHeader("X-User-ID")
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

// Middleware для установки request ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Генерируем новый ID если не предоставлен
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Простая функция генерации request ID (в продакшене используйте UUID)
func generateRequestID() string {
	return "req_" + strconv.FormatInt(time.Now().UnixNano(), 36)
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
