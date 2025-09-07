package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	errors2 "github.com/makhkets/wildberries-l0/internal/errors"
	"github.com/makhkets/wildberries-l0/internal/model"
)

type Repo interface {
	Health() error
	Close() error

	// Основные операции CRUD
	GetOrderByUID(ctx context.Context, uid string) (*model.Order, error)
	CreateOrder(ctx context.Context, order *model.Order) error
	UpdateOrder(ctx context.Context, order *model.Order) error
	DeleteOrder(ctx context.Context, uid string) error

	// Поиск и фильтрация заказов
	GetAllOrders(ctx context.Context) ([]*model.Order, error)
	GetOrdersByCustomerID(ctx context.Context, customerID string) ([]*model.Order, error)
	GetOrdersByDateRange(ctx context.Context, from, to time.Time) ([]*model.Order, error)
	GetOrdersWithPagination(ctx context.Context, limit, offset int) ([]*model.Order, error)
	GetOrdersByTrackNumber(ctx context.Context, trackNumber string) ([]*model.Order, error)
	GetOrdersByPaymentProvider(ctx context.Context, provider string) ([]*model.Order, error)
	GetOrdersByDeliveryRegion(ctx context.Context, region string) ([]*model.Order, error)
	GetOrdersByItemBrand(ctx context.Context, brand string) ([]*model.Order, error)
	GetOrdersByPriceRange(ctx context.Context, minPrice, maxPrice int) ([]*model.Order, error)
	SearchOrders(ctx context.Context, searchTerm string) ([]*model.Order, error)

	// Утилитарные методы
	OrderExists(ctx context.Context, uid string) (bool, error)
	GetOrdersCount(ctx context.Context) (int, error)
}

// GetOrderByUID получает заказ по UID из базы данных одним запросом с JOIN
func (db *Database) GetOrderByUID(ctx context.Context, uid string) (*model.Order, error) {
	// сначала получаем основную информацию о заказе, доставке и платеже одним запросом
	mainQuery := `
		SELECT 
			o.id, o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
			o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created,
			o.oof_shard, o.created_at, o.updated_at,
			
			COALESCE(d.id, 0) as delivery_id,
			COALESCE(d.order_id, 0) as delivery_order_id, 
			COALESCE(d.name, '') as delivery_name,
			COALESCE(d.phone, '') as delivery_phone,
			COALESCE(d.zip, '') as delivery_zip,
			COALESCE(d.city, '') as delivery_city,
			COALESCE(d.address, '') as delivery_address,
			COALESCE(d.region, '') as delivery_region,
			COALESCE(d.email, '') as delivery_email,
			
			COALESCE(p.id, 0) as payment_id,
			COALESCE(p.order_id, 0) as payment_order_id,
			COALESCE(p.transaction, '') as payment_transaction,
			COALESCE(p.request_id, '') as payment_request_id,
			COALESCE(p.currency, '') as payment_currency,
			COALESCE(p.provider, '') as payment_provider,
			COALESCE(p.amount, 0) as payment_amount,
			COALESCE(p.payment_dt, 0) as payment_dt,
			COALESCE(p.bank, '') as payment_bank,
			COALESCE(p.delivery_cost, 0) as payment_delivery_cost,
			COALESCE(p.goods_total, 0) as payment_goods_total,
			COALESCE(p.custom_fee, 0) as payment_custom_fee
		FROM orders o
		LEFT JOIN delivery d ON o.id = d.order_id
		LEFT JOIN payment p ON o.id = p.order_id
		WHERE o.order_uid = $1`

	order := &model.Order{}

	err := db.DB.QueryRowContext(ctx, mainQuery, uid).Scan(
		&order.ID, &order.OrderUID, &order.TrackNumber, &order.Entry,
		&order.Locale, &order.InternalSignature, &order.CustomerID,
		&order.DeliveryService, &order.Shardkey, &order.SmID,
		&order.DateCreated, &order.OofShard, &order.CreatedAt, &order.UpdatedAt,

		&order.Delivery.ID, &order.Delivery.OrderID, &order.Delivery.Name,
		&order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
		&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,

		&order.Payment.ID, &order.Payment.OrderID, &order.Payment.Transaction,
		&order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
		&order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank,
		&order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors2.NewNotFoundError("order")
		}
		return nil, errors2.NewDatabaseError("get order", err)
	}

	// Получаем товарные позиции отдельным запросом (так как их может быть много)
	itemsQuery := `
		SELECT id, order_id, chrt_id, track_number, price, rid, name,
		       sale, size, total_price, nm_id, brand, status
		FROM items 
		WHERE order_id = $1
		ORDER BY id`

	rows, err := db.DB.QueryContext(ctx, itemsQuery, order.ID)
	if err != nil {
		return nil, errors2.NewDatabaseError("get order items", err)
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		err = rows.Scan(
			&item.ID, &item.OrderID, &item.ChrtID, &item.TrackNumber,
			&item.Price, &item.RID, &item.Name, &item.Sale, &item.Size,
			&item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return nil, errors2.NewDatabaseError("scan order item", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, errors2.NewDatabaseError("iterate order items", err)
	}

	order.Items = items
	return order, nil
}

// CreateOrder создает новый заказ в базе данных
func (db *Database) CreateOrder(ctx context.Context, order *model.Order) error {
	// Проверяем, что заказ с таким UID не существует
	exists, err := db.OrderExists(ctx, order.OrderUID)
	if err != nil {
		return errors2.NewDatabaseError("check order existence", err)
	}
	if exists {
		return errors2.NewConflictError("order")
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return errors2.NewDatabaseError("begin transaction", err)
	}
	defer tx.Rollback()

	// Создаем основной заказ
	orderQuery := `
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature,
		                   customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, orderQuery,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return errors2.NewDatabaseError("insert order", err)
	}

	// Создаем информацию о доставке
	if order.Delivery.Name != "" {
		deliveryQuery := `
			INSERT INTO delivery (order_id, name, phone, zip, city, address, region, email)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`

		err = tx.QueryRowContext(ctx, deliveryQuery,
			order.ID, order.Delivery.Name, order.Delivery.Phone,
			order.Delivery.Zip, order.Delivery.City, order.Delivery.Address,
			order.Delivery.Region, order.Delivery.Email,
		).Scan(&order.Delivery.ID)
		if err != nil {
			return errors2.NewDatabaseError("insert delivery", err)
		}
		order.Delivery.OrderID = order.ID
	}

	// Создаем информацию о платеже
	if order.Payment.Transaction != "" {
		paymentQuery := `
			INSERT INTO payment (order_id, transaction, request_id, currency, provider,
			                    amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id`

		err = tx.QueryRowContext(ctx, paymentQuery,
			order.ID, order.Payment.Transaction, order.Payment.RequestID,
			order.Payment.Currency, order.Payment.Provider, order.Payment.Amount,
			order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
			order.Payment.GoodsTotal, order.Payment.CustomFee,
		).Scan(&order.Payment.ID)
		if err != nil {
			return errors2.NewDatabaseError("insert payment", err)
		}
		order.Payment.OrderID = order.ID
	}

	// Создаем товарные позиции
	for i := range order.Items {
		itemQuery := `
			INSERT INTO items (order_id, chrt_id, track_number, price, rid, name,
			                  sale, size, total_price, nm_id, brand, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`

		err = tx.QueryRowContext(ctx, itemQuery,
			order.ID, order.Items[i].ChrtID, order.Items[i].TrackNumber,
			order.Items[i].Price, order.Items[i].RID, order.Items[i].Name,
			order.Items[i].Sale, order.Items[i].Size, order.Items[i].TotalPrice,
			order.Items[i].NmID, order.Items[i].Brand, order.Items[i].Status,
		).Scan(&order.Items[i].ID)
		if err != nil {
			return errors2.NewDatabaseError(fmt.Sprintf("insert item %d", i), err)
		}
		order.Items[i].OrderID = order.ID
	}

	if err := tx.Commit(); err != nil {
		return errors2.NewDatabaseError("commit transaction", err)
	}

	return nil
}

// UpdateOrder обновляет существующий заказ
func (db *Database) UpdateOrder(ctx context.Context, order *model.Order) error {
	query := `
		UPDATE orders 
		SET track_number = $2, entry = $3, locale = $4, internal_signature = $5,
		    customer_id = $6, delivery_service = $7, shardkey = $8, sm_id = $9,
		    date_created = $10, oof_shard = $11
		WHERE order_uid = $1`

	result, err := db.DB.ExecContext(ctx, query,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order with UID %s not found", order.OrderUID)
	}

	return nil
}

// DeleteOrder удаляет заказ по UID
func (db *Database) DeleteOrder(ctx context.Context, uid string) error {
	query := `DELETE FROM orders WHERE order_uid = $1`

	result, err := db.DB.ExecContext(ctx, query, uid)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order with UID %s not found", uid)
	}

	return nil
}

// GetOrdersByCustomerID получает все заказы для конкретного клиента
func (db *Database) GetOrdersByCustomerID(ctx context.Context, customerID string) ([]*model.Order, error) {
	query := `
		SELECT order_uid
		FROM orders 
		WHERE customer_id = $1
		ORDER BY date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for customer: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetOrdersByDateRange получает заказы в определенном диапазоне дат
func (db *Database) GetOrdersByDateRange(ctx context.Context, from, to time.Time) ([]*model.Order, error) {
	query := `
		SELECT order_uid
		FROM orders 
		WHERE date_created BETWEEN $1 AND $2
		ORDER BY date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by date range: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetOrdersWithPagination получает заказы с пагинацией
func (db *Database) GetOrdersWithPagination(ctx context.Context, limit, offset int) ([]*model.Order, error) {
	query := `
		SELECT order_uid
		FROM orders 
		ORDER BY date_created DESC
		LIMIT $1 OFFSET $2`

	rows, err := db.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders with pagination: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetAllOrders получает все заказы из базы данных
func (db *Database) GetAllOrders(ctx context.Context) ([]*model.Order, error) {
	query := `
		SELECT order_uid
		FROM orders 
		ORDER BY date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetOrdersByTrackNumber находит заказы по трек-номеру
func (db *Database) GetOrdersByTrackNumber(ctx context.Context, trackNumber string) ([]*model.Order, error) {
	query := `
		SELECT order_uid
		FROM orders 
		WHERE track_number = $1
		ORDER BY date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, trackNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by track number: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// OrderExists проверяет существование заказа по UID
func (db *Database) OrderExists(ctx context.Context, uid string) (bool, error) {
	query := `SELECT 1 FROM orders WHERE order_uid = $1`

	var exists int
	err := db.DB.QueryRowContext(ctx, query, uid).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check order existence: %w", err)
	}

	return true, nil
}

// GetOrdersCount получает общее количество заказов
func (db *Database) GetOrdersCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM orders`

	var count int
	err := db.DB.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get orders count: %w", err)
	}

	return count, nil
}

// GetOrdersByPaymentProvider получает заказы по провайдеру платежей
func (db *Database) GetOrdersByPaymentProvider(ctx context.Context, provider string) ([]*model.Order, error) {
	query := `
		SELECT o.order_uid
		FROM orders o
		JOIN payment p ON o.id = p.order_id
		WHERE p.provider = $1
		ORDER BY o.date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by payment provider: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetOrdersByDeliveryRegion получает заказы по региону доставки
func (db *Database) GetOrdersByDeliveryRegion(ctx context.Context, region string) ([]*model.Order, error) {
	query := `
		SELECT o.order_uid
		FROM orders o
		JOIN delivery d ON o.id = d.order_id
		WHERE d.region = $1
		ORDER BY o.date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by delivery region: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetOrdersByItemBrand получает заказы содержащие товары определенного бренда
func (db *Database) GetOrdersByItemBrand(ctx context.Context, brand string) ([]*model.Order, error) {
	query := `
		SELECT DISTINCT o.order_uid
		FROM orders o
		JOIN items i ON o.id = i.order_id
		WHERE i.brand = $1
		ORDER BY o.date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by item brand: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetOrdersByPriceRange получает заказы в определенном ценовом диапазоне
func (db *Database) GetOrdersByPriceRange(ctx context.Context, minPrice, maxPrice int) ([]*model.Order, error) {
	query := `
		SELECT o.order_uid
		FROM orders o
		JOIN payment p ON o.id = p.order_id
		WHERE p.amount BETWEEN $1 AND $2
		ORDER BY o.date_created DESC`

	rows, err := db.DB.QueryContext(ctx, query, minPrice, maxPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by price range: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// SearchOrders выполняет поиск заказов по различным критериям
func (db *Database) SearchOrders(ctx context.Context, searchTerm string) ([]*model.Order, error) {
	query := `
		SELECT DISTINCT o.order_uid
		FROM orders o
		LEFT JOIN delivery d ON o.id = d.order_id
		LEFT JOIN payment p ON o.id = p.order_id
		LEFT JOIN items i ON o.id = i.order_id
		WHERE o.order_uid ILIKE $1 
		   OR o.track_number ILIKE $1
		   OR o.customer_id ILIKE $1
		   OR d.name ILIKE $1
		   OR d.email ILIKE $1
		   OR i.name ILIKE $1
		   OR i.brand ILIKE $1
		ORDER BY o.date_created DESC`

	searchPattern := "%" + searchTerm + "%"

	rows, err := db.DB.QueryContext(ctx, query, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return nil, fmt.Errorf("failed to scan order UID: %w", err)
		}

		order, err := db.GetOrderByUID(ctx, orderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderUID, err)
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}
