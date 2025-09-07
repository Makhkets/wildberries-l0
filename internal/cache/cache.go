package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/makhkets/wildberries-l0/internal/config"
)

type Cache struct {
	client *redis.Client
}

// MustLoad создает новое подключение к Redis
func MustLoad(cfg *config.Config) Repo {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     "", // без пароля
		DB:           0,  // использовать базу данных по умолчанию
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})

	slog.Info("Attempting to connect to Redis", "host", cfg.Redis.Host, "port", cfg.Redis.Port)

	// Retry connection up to 30 times with 2 second intervals (1 minute total)
	var err error
	for i := 0; i < 30; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = rdb.Ping(ctx).Err()
		cancel()

		if err == nil {
			// Connection successful
			break
		}

		slog.Warn("Failed to connect to Redis", "attempt", i+1, "error", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		slog.Error("Failed to connect to Redis after 30 attempts", "error", err)
		panic(err)
	}

	slog.Info("Successfully connected to Redis")

	return &Cache{client: rdb}
}
