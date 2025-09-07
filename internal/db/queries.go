package db

import (
	"context"
	"github.com/makhkets/wildberries-l0/internal/model"
)

type Repo interface {
	Health() error
	Close() error

	GetOrderByUID(ctx context.Context, uid string) (*model.Order, error)
}

// GetOrderByUID получает заказ по UID из базы данных
func (db *Database) GetOrderByUID(ctx context.Context, uid string) (*model.Order, error) {
	return nil, nil
}
