package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makhkets/wildberries-l0/internal/config"
	"github.com/makhkets/wildberries-l0/internal/service"
	"log/slog"
)

// Server структура HTTP сервера
type Server struct {
	httpServer *http.Server
	handler    *Handler
	logger     *slog.Logger
}

// NewServer создает новый экземпляр сервера
func NewServer(cfg *config.Config, services service.Order) *Server {
	handler := NewHandler(services)

	// todo поменять на релиз мод в продакшене
	gin.SetMode(gin.DebugMode)

	router := handler.InitRoutes()

	server := &Server{
		handler: handler,
		logger:  slog.Default(),
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	return server
}

// Start запускает сервер
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown корректно останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
