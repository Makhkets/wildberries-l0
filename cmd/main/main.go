package main

import (
	"context"
	"github.com/makhkets/wildberries-l0/pkg/lib/logger/sl"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/makhkets/wildberries-l0/internal/api"
	"github.com/makhkets/wildberries-l0/internal/cache"
	"github.com/makhkets/wildberries-l0/internal/config"
	"github.com/makhkets/wildberries-l0/internal/db"
	"github.com/makhkets/wildberries-l0/internal/kafka"
	"github.com/makhkets/wildberries-l0/internal/migrate"
	"github.com/makhkets/wildberries-l0/internal/service"
	"github.com/makhkets/wildberries-l0/pkg/logging"
)

func main() {
	// Инициализация логгера
	logging.SetupLogger()
	cfg := config.GetConfig()

	slog.Info("Starting application with configuration", slog.Any("config", cfg))

	// Подключение к базе данных
	database := db.MustLoad(cfg)
	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Failed to close database", sl.Err(err))
		}
	}()

	// Выполнение миграций
	migrator, err := migrate.NewMigrator(cfg)
	if err != nil {
		slog.Error("Failed to create migrator", sl.Err(err))
		os.Exit(1)
	}

	if err = migrator.Up(); err != nil {
		slog.Error("Failed to run migrations", sl.Err(err))
		os.Exit(1)
	}

	if err = migrator.Close(); err != nil {
		slog.Error("Failed to close migrator", sl.Err(err))
	}

	// Подключение к кэшу
	cacheInstance := cache.MustLoad(cfg)
	defer func() {
		if err = cacheInstance.Close(); err != nil {
			slog.Error("Failed to close cache", sl.Err(err))
		}
	}()

	// Инициализация сервисов
	services := service.NewOrderService(database, cacheInstance)

	// Инициализация Kafka consumer
	kafkaConsumer := kafka.NewConsumer(cfg, services)
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			slog.Error("Failed to close Kafka consumer", sl.Err(err))
		}
	}()

	server := api.NewServer(cfg, services)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Запуск Kafka consumer в отдельной горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting Kafka consumer...")
		if err = kafkaConsumer.Start(ctx); err != nil {
			slog.Error("Kafka consumer error", sl.Err(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting HTTP server...", "port", cfg.HTTPPort)
		if err = server.Start(); err != nil {
			slog.Error("Server failed to start", sl.Err(err))
		}
	}()

	slog.Info("Application started successfully",
		"http_port", cfg.HTTPPort,
		"kafka_topic", cfg.Kafka.Topic,
		"kafka_brokers", cfg.Kafka.Brokers)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("Shutting down application...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err = server.Stop(shutdownCtx); err != nil {
		slog.Error("Failed to shutdown server gracefully", sl.Err(err))
	}

	wg.Wait()

	slog.Info("Application shutdown completed")
}
