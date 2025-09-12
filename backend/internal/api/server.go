package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/makhkets/wildberries-l0/internal/config"
	"github.com/makhkets/wildberries-l0/internal/service"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server представляет HTTP сервер
type Server struct {
	httpServer *http.Server
}

// NewServer создает новый HTTP сервер
func NewServer(cfg *config.Config, service service.Order) *Server {
	// Настраиваем режим Gin
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// Инициализируем маршруты
	handler := NewHandler(service)
	router := handler.InitRoutes()

	// Создаем HTTP сервер
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
	}
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	slog.Info("Starting HTTP server", "addr", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop останавливает HTTP сервер gracefully
func (s *Server) Stop(ctx context.Context) error {
	slog.Info("Stopping HTTP server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	slog.Info("HTTP server stopped")
	return nil
}
