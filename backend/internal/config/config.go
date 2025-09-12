package config

import (
	"log/slog"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	HTTPPort    int
	Environment string
	DB          Database
	Redis       Redis
	Kafka       Kafka
}

type Redis struct {
	Host      string
	Port      int
	MaxOrders int
}

type Database struct {
	Host     string
	Db       string
	User     string
	Password string
	Port     int
}

type Kafka struct {
	Brokers []string
	Topic   string
	GroupID string
}

func GetConfig() *Config {
	conf := &Config{
		HTTPPort:    getEnvAsInt("API_PORT", 8080),
		Environment: getEnv("ENVIRONMENT", "development"),
		DB: Database{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Db:       getEnv("POSTGRES_DB", "postgres"),
			User:     getEnv("POSTGRES_USER", "user"),
			Password: getEnv("POSTGRES_PASSWORD", "user"),
			Port:     getEnvAsInt("POSTGRES_PORT", 5432),
		},
		Redis: Redis{
			Host:      getEnv("REDIS_HOST", "localhost"),
			Port:      getEnvAsInt("REDIS_PORT", 6379),
			MaxOrders: getEnvAsInt("REDIS_MAX_ORDERS", 100),
		},
		Kafka: Kafka{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			Topic:   getEnv("KAFKA_TOPIC", "orders"),
			GroupID: getEnv("KAFKA_GROUP_ID", "wildberries-consumer"),
		},
	}

	if conf.Redis.MaxOrders < 5 {
		slog.Error("REDIS_MAX_ORDERS must be at least 5")
		os.Exit(1)
	}

	return conf
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// dsads
// ds
