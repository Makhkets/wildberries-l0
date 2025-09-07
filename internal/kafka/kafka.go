package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/makhkets/wildberries-l0/internal/config"
	"github.com/makhkets/wildberries-l0/internal/model"
	"github.com/makhkets/wildberries-l0/internal/service"
)

type Consumer interface {
	Start(ctx context.Context) error
	Close() error
}

type consumer struct {
	reader       *kafka.Reader
	orderService service.Order
	config       *config.Config
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(cfg *config.Config, orderService service.Order) Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Kafka.Brokers,
		Topic:       cfg.Kafka.Topic,
		GroupID:     cfg.Kafka.GroupID,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})

	return &consumer{
		reader:       reader,
		orderService: orderService,
		config:       cfg,
	}
}

// Start запускает процесс чтения сообщений из Kafka
func (c *consumer) Start(ctx context.Context) error {
	log.Println("Запуск Kafka consumer для топика:", c.config.Kafka.Topic)

	for {
		select {
		case <-ctx.Done():
			log.Println("Остановка Kafka consumer")
			return c.reader.Close()
		default:
			// Читаем сообщение из Kafka
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Ошибка чтения сообщения из Kafka: %v", err)
				continue
			}

			// Обрабатываем сообщение
			if err = c.processMessage(ctx, message); err != nil {
				log.Printf("Ошибка обработки сообщения: %v", err)
				continue
			}

			log.Printf("Успешно обработано сообщение: offset=%d, partition=%d",
				message.Offset, message.Partition)
		}
	}
}

// processMessage обрабатывает отдельное сообщение
func (c *consumer) processMessage(ctx context.Context, message kafka.Message) error {
	log.Printf("Получено сообщение: key=%s, offset=%d, partition=%d",
		string(message.Key), message.Offset, message.Partition)

	// Парсим JSON сообщение в структуру Order
	var order model.Order
	if err := json.Unmarshal(message.Value, &order); err != nil {
		return err
	}

	log.Printf("Обработка заказа: %s", order.OrderUID)

	// Сохраняем заказ через сервис
	if err := c.orderService.CreateOrder(ctx, &order); err != nil {
		return err
	}

	log.Printf("Заказ %s успешно сохранен", order.OrderUID)
	return nil
}

// Close закрывает соединение с Kafka
func (c *consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
