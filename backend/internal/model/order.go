package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Order основная структура заказа
type Order struct {
	ID                int       `json:"id" db:"id"`
	OrderUID          string    `json:"order_uid" db:"order_uid"`
	TrackNumber       string    `json:"track_number" db:"track_number"`
	Entry             string    `json:"entry" db:"entry"`
	Locale            string    `json:"locale" db:"locale"`
	InternalSignature string    `json:"internal_signature" db:"internal_signature"`
	CustomerID        string    `json:"customer_id" db:"customer_id"`
	DeliveryService   string    `json:"delivery_service" db:"delivery_service"`
	Shardkey          string    `json:"shardkey" db:"shardkey"`
	SmID              int       `json:"sm_id" db:"sm_id"`
	DateCreated       time.Time `json:"date_created" db:"date_created"`
	OofShard          string    `json:"oof_shard" db:"oof_shard"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`

	// Связанные данные
	Delivery *Delivery `json:"delivery"`
	Payment  *Payment  `json:"payment"`
	Items    []Item    `json:"items"`
}

// Delivery информация о доставке
type Delivery struct {
	ID      int    `json:"id" db:"id"`
	OrderID int    `json:"order_id" db:"order_id"`
	Name    string `json:"name" db:"name"`
	Phone   string `json:"phone" db:"phone"`
	Zip     string `json:"zip" db:"zip"`
	City    string `json:"city" db:"city"`
	Address string `json:"address" db:"address"`
	Region  string `json:"region" db:"region"`
	Email   string `json:"email" db:"email"`
}

// Payment информация о платеже
type Payment struct {
	ID           int    `json:"id" db:"id"`
	OrderID      int    `json:"order_id" db:"order_id"`
	Transaction  string `json:"transaction" db:"transaction"`
	RequestID    string `json:"request_id" db:"request_id"`
	Currency     string `json:"currency" db:"currency"`
	Provider     string `json:"provider" db:"provider"`
	Amount       int    `json:"amount" db:"amount"`
	PaymentDt    int64  `json:"payment_dt" db:"payment_dt"`
	Bank         string `json:"bank" db:"bank"`
	DeliveryCost int    `json:"delivery_cost" db:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total" db:"goods_total"`
	CustomFee    int    `json:"custom_fee" db:"custom_fee"`
}

// Item товарная позиция в заказе
type Item struct {
	ID          int    `json:"id" db:"id"`
	OrderID     int    `json:"order_id" db:"order_id"`
	ChrtID      int    `json:"chrt_id" db:"chrt_id"`
	TrackNumber string `json:"track_number" db:"track_number"`
	Price       int    `json:"price" db:"price"`
	RID         string `json:"rid" db:"rid"`
	Name        string `json:"name" db:"name"`
	Sale        int    `json:"sale" db:"sale"`
	Size        string `json:"size" db:"size"`
	TotalPrice  int    `json:"total_price" db:"total_price"`
	NmID        int    `json:"nm_id" db:"nm_id"`
	Brand       string `json:"brand" db:"brand"`
	Status      int    `json:"status" db:"status"`
}

// OrderStatus статусы заказа
type OrderStatus int

const (
	OrderStatusNew OrderStatus = iota + 1
	OrderStatusProcessing
	OrderStatusShipped
	OrderStatusDelivered
	OrderStatusCancelled
)

// PaymentStatus статусы платежа
type PaymentStatus int

const (
	PaymentStatusPending PaymentStatus = iota + 1
	PaymentStatusCompleted
	PaymentStatusFailed
	PaymentStatusRefunded
)

// ItemStatus статусы товара
type ItemStatus int

const (
	ItemStatusActive   ItemStatus = 202
	ItemStatusInactive ItemStatus = 404
)

// Value implements driver.Valuer interface for JSON serialization
func (o Order) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner interface for JSON deserialization
func (o *Order) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, o)
	case string:
		return json.Unmarshal([]byte(v), o)
	default:
		return nil
	}
}
