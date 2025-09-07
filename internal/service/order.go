package service

import (
	"context"
	"github.com/makhkets/wildberries-l0/internal/cache"
	"github.com/makhkets/wildberries-l0/internal/db"
	"github.com/makhkets/wildberries-l0/internal/errors"
	"github.com/makhkets/wildberries-l0/internal/model"
	"log/slog"
	"strings"
	"time"
)

type Order interface {
	GetOrderByUID(ctx context.Context, uid string) (*model.Order, error)
	CreateOrder(ctx context.Context, order *model.Order) error

	mergeDeliveryData(existing, new *model.Delivery) model.Delivery
	mergePaymentData(existing, new *model.Payment) model.Payment
	mergeItemsData(existingItems, newItems []model.Item) []model.Item
	mergeItemData(existing, new *model.Item) model.Item
}

// OrderService представляет сервис для работы с заказами
type OrderService struct {
	repo   db.Repo
	cache  cache.Repo
	logger *slog.Logger
}

// NewOrderService создает новый сервис заказов
func NewOrderService(repo db.Repo, cache cache.Repo) Order {
	return &OrderService{
		repo:  repo,
		cache: cache,
	}
}

// GetOrderByUID получает заказ по UID с дополнительной бизнес-логикой
func (s *OrderService) GetOrderByUID(ctx context.Context, uid string) (*model.Order, error) {
	// Валидация входных данных
	if err := s.validateOrderUID(uid); err != nil {
		s.logger.Warn("Invalid order UID provided", "uid", uid, "error", err)
		return nil, err
	}

	// Получаем заказ из repository
	order, err := s.repo.GetOrderByUID(ctx, uid)
	if err != nil {
		s.logger.Error("Failed to get order from repository",
			"uid", uid, "error", err)

		// Проверяем тип ошибки и решаем, что возвращать
		if errors.IsErrorType(err, errors.ErrorTypeNotFound) {
			return nil, err
		}

		return nil, errors.NewAppError(errors.ErrorTypeInternal,
			"Failed to retrieve order")
	}

	s.logger.Info("Order retrieved successfully", "uid", uid)
	return order, nil
}

// CreateOrder создает новый заказ с валидацией или обновляет существующий
func (s *OrderService) CreateOrder(ctx context.Context, order *model.Order) error {
	// Проверяем, существует ли заказ
	existingOrder, err := s.GetOrderByUID(ctx, order.OrderUID)
	if err != nil {
		// Если ошибка НЕ "не найден", то это серьезная ошибка
		if !errors.IsErrorType(err, errors.ErrorTypeNotFound) {
			s.logger.Error("Failed to check existing order", "uid", order.OrderUID, "error", err)
			return err
		}

		// Заказ не найден - создаем новый
		s.logger.Info("Creating new order", "uid", order.OrderUID)

		err = s.repo.CreateOrder(ctx, order)
		if err != nil {
			s.logger.Error("Failed to create order in repository",
				"uid", order.OrderUID, "error", err)

			if errors.IsErrorType(err, errors.ErrorTypeConflict) {
				return err
			}

			return errors.NewAppError(errors.ErrorTypeInternal,
				"Failed to create order")
		}

		//// Добавляем в кэш после успешного создания
		//if err := s.cache.Set(ctx, order.OrderUID, order); err != nil {
		//	s.logger.Warn("Failed to cache new order", "uid", order.OrderUID, "error", err)
		//	// Не возвращаем ошибку, так как основная операция прошла успешно
		//}

		s.logger.Info("Order created successfully", "uid", order.OrderUID)
		return nil
	}

	// Заказ уже существует - обновляем его
	s.logger.Info("Order already exists, updating with new data", "uid", order.OrderUID)

	// Объединяем существующие данные с новыми
	updatedOrder := s.mergeOrderData(existingOrder, order)

	// Обновляем заказ в базе данных
	err = s.repo.UpdateOrder(ctx, updatedOrder)
	if err != nil {
		s.logger.Error("Failed to update order in repository",
			"uid", order.OrderUID, "error", err)

		if errors.IsErrorType(err, errors.ErrorTypeNotFound) {
			return err
		}

		return errors.NewAppError(errors.ErrorTypeInternal,
			"Failed to update order")
	}

	//// Обновляем кэш
	//if err := s.cache.Set(ctx, order.OrderUID, updatedOrder); err != nil {
	//	s.logger.Warn("Failed to update order in cache", "uid", order.OrderUID, "error", err)
	//}

	// Копируем обновленные данные обратно в переданный объект
	*order = *updatedOrder

	s.logger.Info("Order updated successfully", "uid", order.OrderUID)
	return nil
}

// GetOrdersByCustomer получает заказы клиента с проверкой прав доступа
func (s *OrderService) GetOrdersByCustomer(ctx context.Context, customerID string, requestingUserID string) ([]*model.Order, error) {
	// Проверяем права доступа (пример бизнес-логики)
	if !s.canAccessCustomerOrders(customerID, requestingUserID) {
		s.logger.Warn("Unauthorized access to customer orders",
			"customerID", customerID, "requestingUser", requestingUserID)
		return nil, errors.NewForbiddenError("Access to customer orders denied")
	}

	orders, err := s.repo.GetOrdersByCustomerID(ctx, customerID)
	if err != nil {
		s.logger.Error("Failed to get customer orders",
			"customerID", customerID, "error", err)
		return nil, errors.NewAppError(errors.ErrorTypeInternal,
			"Failed to retrieve customer orders")
	}

	return orders, nil
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

	// Валидация delivery
	if order.Delivery.Name != "" {
		if err := s.validateDelivery(order.Delivery); err != nil {
			return err
		}
	}

	// Валидация payment
	if order.Payment.Transaction != "" {
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
	if item.Name == "" {
		return errors.NewValidationError("items["+string(rune(index))+"].name", "cannot be empty")
	}

	if item.Price <= 0 {
		return errors.NewValidationError("items["+string(rune(index))+"].price", "must be greater than 0")
	}

	if item.Brand == "" {
		return errors.NewValidationError("items["+string(rune(index))+"].brand", "cannot be empty")
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

	// Обновляем данные доставки
	*updated.Delivery = s.mergeDeliveryData(updated.Delivery, new.Delivery)

	// Обновляем данные платежа
	*updated.Payment = s.mergePaymentData(updated.Payment, new.Payment)

	// Обновляем товарные позиции
	updated.Items = s.mergeItemsData(updated.Items, new.Items)

	// Обновляем время изменения
	updated.UpdatedAt = time.Now()

	return &updated
}

// mergeDeliveryData объединяет данные доставки
func (s *OrderService) mergeDeliveryData(existing, new *model.Delivery) model.Delivery {
	updated := *existing

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

	return updated
}

// mergePaymentData объединяет данные платежа
func (s *OrderService) mergePaymentData(existing, new *model.Payment) model.Payment {
	updated := *existing

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

	return updated
}

// mergeItemsData объединяет товарные позиции
// Стратегия: заменяем все существующие товары новыми, если новые товары предоставлены
func (s *OrderService) mergeItemsData(existingItems, newItems []model.Item) []model.Item {
	// Если новых товаров нет, оставляем существующие
	if len(newItems) == 0 {
		return existingItems
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
