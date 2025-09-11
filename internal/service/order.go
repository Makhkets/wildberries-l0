package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/makhkets/wildberries-l0/internal/cache"
	"github.com/makhkets/wildberries-l0/internal/config"
	"github.com/makhkets/wildberries-l0/internal/db"
	"github.com/makhkets/wildberries-l0/internal/errors"
	"github.com/makhkets/wildberries-l0/internal/model"
	"github.com/makhkets/wildberries-l0/pkg/lib/logger/sl"
)

type Order interface {
	GetOrderByUID(ctx context.Context, uid string) (*model.Order, error)
	CreateOrder(ctx context.Context, order *model.Order) error

	MustLoadCache(ctx context.Context)
}

// OrderService представляет сервис для работы с заказами
type OrderService struct {
	repo   db.Repo
	cache  cache.Repo
	config *config.Config
}

// NewOrderService создает новый сервис заказов
func NewOrderService(repo db.Repo, cache cache.Repo, config *config.Config) Order {
	return &OrderService{
		repo:   repo,
		cache:  cache,
		config: config,
	}
}

func (s *OrderService) MustLoadCache(ctx context.Context) {
	keys, err := s.cache.GetAllKeys(ctx, "order:*")
	if err != nil {
		slog.Error("Failed to get all keys from cache", sl.Err(err))
		return
	}
	for _, key := range keys {
		fmt.Println(key)
	}

	// >
	if len(keys) <= s.config.Redis.MaxOrders {
		//delKeys := keys[:len(keys)-s.config.Redis.MaxOrders+1]
	}

	return // todo delete
	orders, err := s.repo.GetCacheOrders(ctx, s.config.Redis.MaxOrders)
	if err != nil {
		slog.Error("Failed to load orders for cache", sl.Err(err))
		panic(err)
	}

	successAdded := s.cache.SetOrders(ctx, orders)

	slog.Info("Orders loaded into cache successfully", slog.Int("count", successAdded))
}

// GetOrderByUID получает заказ по UID с дополнительной бизнес-логикой
func (s *OrderService) GetOrderByUID(ctx context.Context, uid string) (*model.Order, error) {
	// Валидация входных данных
	if err := s.validateOrderUID(uid); err != nil {
		slog.Warn("Invalid order UID provided", "uid", uid, "error", err)
		return nil, err
	}

	// Получаем заказ из repository
	order, err := s.repo.GetOrderByUID(ctx, uid)
	if err != nil {
		slog.Error("Failed to get order from repository",
			"uid", uid, "error", err)

		// Проверяем тип ошибки и решаем, что возвращать
		if errors.IsErrorType(err, errors.ErrorTypeNotFound) {
			return nil, err
		}

		return nil, errors.NewAppError(errors.ErrorTypeInternal,
			"Failed to retrieve order")
	}

	slog.Info("Order retrieved successfully", "uid", uid)
	return order, nil
}

// CreateOrder создает новый заказ с валидацией или обновляет существующий
func (s *OrderService) CreateOrder(ctx context.Context, order *model.Order) error {
	// Проверяем, существует ли заказ
	existingOrder, err := s.GetOrderByUID(ctx, order.OrderUID)
	if err != nil {
		// Если ошибка НЕ "не найден", то это серьезная ошибка
		if !errors.IsErrorType(err, errors.ErrorTypeNotFound) {
			slog.Error("Failed to check existing order", "uid", order.OrderUID, "error", err)
			return err
		}

		// Заказ не найден - создаем новый
		slog.Info("Creating new order", "uid", order.OrderUID)

		err = s.repo.CreateOrder(ctx, order)
		if err != nil {
			slog.Error("Failed to create order in repository",
				"uid", order.OrderUID, "error", err)

			if errors.IsErrorType(err, errors.ErrorTypeConflict) {
				return err
			}

			return errors.NewAppError(errors.ErrorTypeInternal,
				"Failed to create order")
		}

		slog.Info("Order created successfully", "uid", order.OrderUID)
		return nil
	}

	// Заказ уже существует - обновляем его
	slog.Info("Order already exists, updating with new data", "uid", order.OrderUID)

	// Объединяем существующие данные с новыми
	updatedOrder := s.mergeOrderData(existingOrder, order)

	// Обновляем заказ в базе данных
	err = s.repo.UpdateOrder(ctx, updatedOrder)
	if err != nil {
		slog.Error("Failed to update order in repository",
			"uid", order.OrderUID, "error", err)

		if errors.IsErrorType(err, errors.ErrorTypeNotFound) {
			return err
		}

		return errors.NewAppError(errors.ErrorTypeInternal,
			"Failed to update order")
	}

	// Копируем обновленные данные обратно в переданный объект
	*order = *updatedOrder

	slog.Info("Order updated successfully", "uid", order.OrderUID)
	return nil
}

// validateOrderUID проверяет корректность UID заказа
func (s *OrderService) validateOrderUID(uid string) error {
	if uid == "" {
		return errors.NewValidationError("order_uid", "cannot be empty")
	}

	if len(uid) < 10 || len(uid) > 255 {
		return errors.NewValidationError("order_uid", "must be between 10 and 255 characters")
	}

	// Дополнительные проверки формата UID
	if strings.Contains(uid, " ") {
		return errors.NewValidationError("order_uid", "cannot contain spaces")
	}

	return nil
}

// validateOrder проверяет корректность данных заказа
func (s *OrderService) validateOrder(order *model.Order) error {
	if order == nil {
		return errors.NewValidationError("order", "cannot be nil")
	}

	// Проверяем обязательные поля
	if err := s.validateOrderUID(order.OrderUID); err != nil {
		return err
	}

	if order.TrackNumber == "" {
		return errors.NewValidationError("track_number", "cannot be empty")
	}

	if order.CustomerID == "" {
		return errors.NewValidationError("customer_id", "cannot be empty")
	}

	// Валидация delivery с проверкой на nil
	if order.Delivery != nil && order.Delivery.Name != "" {
		if err := s.validateDelivery(order.Delivery); err != nil {
			return err
		}
	}

	// Валидация payment с проверкой на nil
	if order.Payment != nil && order.Payment.Transaction != "" {
		if err := s.validatePayment(order.Payment); err != nil {
			return err
		}
	}

	// Валидация items
	if len(order.Items) == 0 {
		return errors.NewValidationError("items", "order must contain at least one item")
	}

	for i, item := range order.Items {
		if err := s.validateItem(&item, i); err != nil {
			return err
		}
	}

	return nil
}

// validateDelivery проверяет данные доставки
func (s *OrderService) validateDelivery(delivery *model.Delivery) error {
	if delivery == nil {
		return errors.NewValidationError("delivery", "cannot be nil")
	}

	if delivery.Name == "" {
		return errors.NewValidationError("delivery.name", "cannot be empty")
	}

	if delivery.Phone == "" {
		return errors.NewValidationError("delivery.phone", "cannot be empty")
	}

	if delivery.Address == "" {
		return errors.NewValidationError("delivery.address", "cannot be empty")
	}

	// Простая валидация email
	if delivery.Email != "" && !strings.Contains(delivery.Email, "@") {
		return errors.NewValidationError("delivery.email", "invalid email format")
	}

	return nil
}

// validatePayment проверяет данные платежа
func (s *OrderService) validatePayment(payment *model.Payment) error {
	if payment == nil {
		return errors.NewValidationError("payment", "cannot be nil")
	}

	if payment.Transaction == "" {
		return errors.NewValidationError("payment.transaction", "cannot be empty")
	}

	if payment.Currency == "" {
		return errors.NewValidationError("payment.currency", "cannot be empty")
	}

	if payment.Provider == "" {
		return errors.NewValidationError("payment.provider", "cannot be empty")
	}

	if payment.Amount <= 0 {
		return errors.NewValidationError("payment.amount", "must be greater than 0")
	}

	return nil
}

// validateItem проверяет товарную позицию
func (s *OrderService) validateItem(item *model.Item, index int) error {
	if item == nil {
		return errors.NewValidationError(fmt.Sprintf("items[%d]", index), "cannot be nil")
	}

	if item.Name == "" {
		return errors.NewValidationError(fmt.Sprintf("items[%d].name", index), "cannot be empty")
	}

	if item.Price <= 0 {
		return errors.NewValidationError(fmt.Sprintf("items[%d].price", index), "must be greater than 0")
	}

	if item.Brand == "" {
		return errors.NewValidationError(fmt.Sprintf("items[%d].brand", index), "cannot be empty")
	}

	return nil
}

// canAccessCustomerOrders проверяет права доступа к заказам клиента
func (s *OrderService) canAccessCustomerOrders(customerID, requestingUserID string) bool {
	// Упрощенная логика: пользователь может видеть только свои заказы
	// В реальном приложении здесь была бы проверка ролей, токенов и т.д.
	return customerID == requestingUserID
}

// mergeOrderData объединяет существующие данные заказа с новыми
// Новые данные имеют приоритет над существующими (если они не пустые)
func (s *OrderService) mergeOrderData(existing, new *model.Order) *model.Order {
	// Создаем копию существующего заказа
	updated := *existing

	// Обновляем основные поля заказа (только непустые значения)
	if new.TrackNumber != "" {
		updated.TrackNumber = new.TrackNumber
	}
	if new.Entry != "" {
		updated.Entry = new.Entry
	}
	if new.Locale != "" {
		updated.Locale = new.Locale
	}
	if new.InternalSignature != "" {
		updated.InternalSignature = new.InternalSignature
	}
	if new.CustomerID != "" {
		updated.CustomerID = new.CustomerID
	}
	if new.DeliveryService != "" {
		updated.DeliveryService = new.DeliveryService
	}
	if new.Shardkey != "" {
		updated.Shardkey = new.Shardkey
	}
	if new.SmID != 0 {
		updated.SmID = new.SmID
	}
	if !new.DateCreated.IsZero() {
		updated.DateCreated = new.DateCreated
	}
	if new.OofShard != "" {
		updated.OofShard = new.OofShard
	}

	// Обновляем данные доставки безопасно
	if updated.Delivery == nil {
		updated.Delivery = &model.Delivery{}
	}
	mergedDelivery := s.mergeDeliveryData(updated.Delivery, new.Delivery)
	updated.Delivery = &mergedDelivery

	// Обновляем данные платежа безопасно
	if updated.Payment == nil {
		updated.Payment = &model.Payment{}
	}
	mergedPayment := s.mergePaymentData(updated.Payment, new.Payment)
	updated.Payment = &mergedPayment

	// Обновляем товарные позиции
	updated.Items = s.mergeItemsData(updated.Items, new.Items)

	// Обновляем время изменения
	updated.UpdatedAt = time.Now()

	return &updated
}

// mergeDeliveryData объединяет данные доставки
func (s *OrderService) mergeDeliveryData(existing, new *model.Delivery) model.Delivery {
	// Если existing равен nil, возвращаем new или пустую структуру
	if existing == nil {
		if new == nil {
			return model.Delivery{}
		}
		return *new
	}

	// Создаем копию существующих данных
	updated := *existing

	// Обновляем только если new не nil и поля не пустые
	if new != nil {
		if new.Name != "" {
			updated.Name = new.Name
		}
		if new.Phone != "" {
			updated.Phone = new.Phone
		}
		if new.Zip != "" {
			updated.Zip = new.Zip
		}
		if new.City != "" {
			updated.City = new.City
		}
		if new.Address != "" {
			updated.Address = new.Address
		}
		if new.Region != "" {
			updated.Region = new.Region
		}
		if new.Email != "" {
			updated.Email = new.Email
		}
	}

	return updated
}

// mergePaymentData объединяет данные платежа
func (s *OrderService) mergePaymentData(existing, new *model.Payment) model.Payment {
	// Если existing равен nil, возвращаем new или пустую структуру
	if existing == nil {
		if new == nil {
			return model.Payment{}
		}
		return *new
	}

	// Создаем копию существующих данных
	updated := *existing

	// Обновляем только если new не nil и поля не пустые
	if new != nil {
		if new.Transaction != "" {
			updated.Transaction = new.Transaction
		}
		if new.RequestID != "" {
			updated.RequestID = new.RequestID
		}
		if new.Currency != "" {
			updated.Currency = new.Currency
		}
		if new.Provider != "" {
			updated.Provider = new.Provider
		}
		if new.Amount != 0 {
			updated.Amount = new.Amount
		}
		if new.PaymentDt != 0 {
			updated.PaymentDt = new.PaymentDt
		}
		if new.Bank != "" {
			updated.Bank = new.Bank
		}
		if new.DeliveryCost != 0 {
			updated.DeliveryCost = new.DeliveryCost
		}
		if new.GoodsTotal != 0 {
			updated.GoodsTotal = new.GoodsTotal
		}
		if new.CustomFee != 0 {
			updated.CustomFee = new.CustomFee
		}
	}

	return updated
}

// mergeItemsData объединяет массивы товаров
func (s *OrderService) mergeItemsData(existingItems, newItems []model.Item) []model.Item {
	// Если нет новых товаров, возвращаем существующие
	if len(newItems) == 0 {
		return existingItems
	}

	// Если нет существующих товаров, возвращаем новые
	if len(existingItems) == 0 {
		return newItems
	}

	// Если есть новые товары, создаем карту существующих товаров по ChrtID
	existingMap := make(map[int]model.Item)
	for _, item := range existingItems {
		existingMap[item.ChrtID] = item
	}

	var updatedItems []model.Item

	// Обрабатываем новые товары
	for _, newItem := range newItems {
		if existingItem, exists := existingMap[newItem.ChrtID]; exists {
			// Товар существует - обновляем его
			updated := s.mergeItemData(&existingItem, &newItem)
			updatedItems = append(updatedItems, updated)
			// Удаляем из карты, чтобы отследить какие товары остались
			delete(existingMap, newItem.ChrtID)
		} else {
			// Новый товар - добавляем как есть
			updatedItems = append(updatedItems, newItem)
		}
	}

	// Добавляем оставшиеся существующие товары, которых не было в новых данных
	for _, remainingItem := range existingMap {
		updatedItems = append(updatedItems, remainingItem)
	}

	return updatedItems
}

// mergeItemData объединяет данные отдельного товара
func (s *OrderService) mergeItemData(existing, new *model.Item) model.Item {
	updated := *existing

	if new.TrackNumber != "" {
		updated.TrackNumber = new.TrackNumber
	}
	if new.Price != 0 {
		updated.Price = new.Price
	}
	if new.RID != "" {
		updated.RID = new.RID
	}
	if new.Name != "" {
		updated.Name = new.Name
	}
	if new.Sale != 0 {
		updated.Sale = new.Sale
	}
	if new.Size != "" {
		updated.Size = new.Size
	}
	if new.TotalPrice != 0 {
		updated.TotalPrice = new.TotalPrice
	}
	if new.NmID != 0 {
		updated.NmID = new.NmID
	}
	if new.Brand != "" {
		updated.Brand = new.Brand
	}
	if new.Status != 0 {
		updated.Status = new.Status
	}

	return updated
}
