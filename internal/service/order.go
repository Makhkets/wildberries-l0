package service

import (
	"context"

	"github.com/makhkets/wildberries-l0/internal/cache"
	"github.com/makhkets/wildberries-l0/internal/db"
	"github.com/makhkets/wildberries-l0/internal/model"
)

type Order interface {
	CreateOrder(ctx context.Context, order *model.Order) error
}

// orderService реализация интерфейса OrderService
type orderService struct {
	db    db.Repo
	cache cache.Repo
}

// NewOrderService создает новый экземпляр сервиса заказов
func NewOrderService(db db.Repo, cache cache.Repo) Order {
	service := &orderService{
		db:    db,
		cache: cache,
	}

	// Восстанавливаем кэш из базы данных при инициализации
	//go service.restoreCache()

	return service
}

func (s *orderService) CreateOrder(ctx context.Context, order *model.Order) error {
	// todo create order
	return nil
}
