package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/makhkets/wildberries-l0/internal/model"
	"log/slog"
	"time"
)

type Repo interface {
	Close() error
	Health() error

	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	GetOrder(uid string) *model.Order
	SetOrder(uid string, order *model.Order)
	RestoreCacheFromDB(orders []*model.Order)
	GetCacheStats(ctx context.Context) (map[string]interface{}, error)
}

// Close закрывает подключение к Redis
func (c *Cache) Close() error {
	return c.client.Close()
}

// Set устанавливает значение в кэш
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get получает значение из кэша
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(redis.Nil, err) {
		return "", fmt.Errorf("key does not exist")
	}
	return val, err
}

// Delete удаляет значение из кэша
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists проверяет существование ключа
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	val, err := c.client.Exists(ctx, key).Result()
	return val > 0, err
}

// Health проверяет состояние подключения к Redis
func (c *Cache) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.client.Ping(ctx).Err()
}

// GetOrder получает заказ из кэша
func (c *Cache) GetOrder(uid string) *model.Order {
	ctx := context.Background()
	val, err := c.client.Get(ctx, fmt.Sprintf("order:%s", uid)).Result()
	if err != nil {
		return nil
	}

	var order model.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		slog.Error("failed to unmarshal order from cache", "uid", uid, "error", err)
		return nil
	}

	return &order
}

// SetOrder сохраняет заказ в кэш
func (c *Cache) SetOrder(uid string, order *model.Order) {
	ctx := context.Background()
	orderData, err := json.Marshal(order)
	if err != nil {
		slog.Error("failed to marshal order for cache", "uid", uid, "error", err)
		return
	}

	// Сохраняем на 1 час
	if err := c.client.Set(ctx, fmt.Sprintf("order:%s", uid), orderData, time.Hour).Err(); err != nil {
		slog.Error("failed to set order in cache", "uid", uid, "error", err)
	}
}

// RestoreCacheFromDB восстанавливает кэш из базы данных при старте
func (c *Cache) RestoreCacheFromDB(orders []*model.Order) {
	ctx := context.Background()

	slog.Info("Restoring cache from database", "orders_count", len(orders))

	for _, order := range orders {
		orderData, err := json.Marshal(order)
		if err != nil {
			slog.Error("failed to marshal order for cache restore", "uid", order.OrderUID, "error", err)
			continue
		}

		// Сохраняем на 24 часа при восстановлении кэша
		key := fmt.Sprintf("order:%s", order.OrderUID)
		if err := c.client.Set(ctx, key, orderData, 24*time.Hour).Err(); err != nil {
			slog.Error("failed to set order in cache during restore", "uid", order.OrderUID, "error", err)
		}
	}

	slog.Info("Cache restoration completed")
}

// GetCacheStats возвращает статистику кэша
func (c *Cache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	// Получаем количество ключей с префиксом order:
	keys, err := c.client.Keys(ctx, "order:*").Result()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"cached_orders": len(keys),
		"redis_info":    info,
	}

	return stats, nil
}
