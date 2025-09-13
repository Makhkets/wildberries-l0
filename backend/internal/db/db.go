package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
	"github.com/makhkets/wildberries-l0/internal/config"
)

type Database struct {
	*sql.DB
}

// MustLoad создает новое подключение к базе данных PostgreSQL
func MustLoad(cfg *config.Config) Repo {
	// host=postgres port=5432 user=postgres password=1324 dbname=orders sslmode=disable
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Db,
	)

	slog.Info("Attempting to connect to Postgres",
		slog.String("host", cfg.DB.Host),
		slog.Int("port", cfg.DB.Port),
		slog.String("database", cfg.DB.Db),
	)

	var db *sql.DB
	var err error

	// Retry connection up to 30 times with 2 second intervals (1 minute total)
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			slog.Warn("Failed to open database connection", "attempt", i+1, "error", err)
			time.Sleep(2 * time.Second)
			continue

		// Test the connection
		err = db.Ping()
		if err != nil {
			slog.Warn("Failed to ping database", "attempt", i+1, "error", err)
			db.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		// Connection successful
		break
	}

	if err != nil {
		slog.Error("Failed to connect to database after 30 attempts", "error", err)
		panic(err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	slog.Info("Successfully connected to PostgreSQL database")

	return &Database{db}
}

// Close закрывает подключение к базе данных
func (db *Database) Close() error {
	return db.DB.Close()
}

// Health проверяет состояние подключения к базе данных
func (db *Database) Health() error {
	return db.Ping()
}
