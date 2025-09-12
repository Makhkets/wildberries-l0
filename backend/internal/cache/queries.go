package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/makhkets/wildberries-l0/internal/model"
	"github.com/makhkets/wildberries-l0/pkg/lib/logger/sl"
)

type Repo interface {
	Close() error
	Health() error

	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	GetOrder(context context.Context, uid string) *model.Order
	SetOrders(context context.Context, orders []*model.Order) int
	GetCacheStats(ctx context.Context) (map[string]interface{}, error)

	GetAllKeys(ctx context.Context, pattern string) ([]string, error)
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
func (c *Cache) GetOrder(context context.Context, uid string) *model.Order {
	val, err := c.client.Get(context, fmt.Sprintf("order:%s", uid)).Result()
	if err != nil {
		return nil
	}

	var order model.Order
	if err = json.Unmarshal([]byte(val), &order); err != nil {
		slog.Error("failed to unmarshal order from cache", "uid", uid, "error", err)
		return nil
	}

	return &order
}

// SetOrders сохраняет заказы в кэш, возвращает количество успешно добавленных заказов
// если достигнуто максимальное количество заказов в кэше (см. в конфиге), то новые заказы, добавляются вместо старых
func (c *Cache) SetOrders(context context.Context, orders []*model.Order) int {
	successAdded := 0

	for _, order := range orders {
		orderData, err := json.Marshal(order)
		if err != nil {
			slog.Error("failed to marshal order for cache", slog.String("uid", order.OrderUID), sl.Err(err))
		}

		if err = c.client.Set(context, fmt.Sprintf("order:%s", order.OrderUID), orderData, 0).Err(); err != nil {
			slog.Error("failed to set order in cache", slog.String("uid", order.OrderUID), sl.Err(err))
			continue
		}

		//slog.Debug("order cached", slog.String("uid", order.OrderUID))
		successAdded++
	}

	return successAdded
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

// GetAllKeys возвращает все ключи, соответствующие заданному шаблону
func (c *Cache) GetAllKeys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	return keys, nil
}
