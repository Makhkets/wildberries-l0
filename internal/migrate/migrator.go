package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/makhkets/wildberries-l0/internal/config"
)

type Migrator struct {
	migrate *migrate.Migrate
}

// NewMigrator создает новый экземпляр мигратора
func NewMigrator(cfg *config.Config) (*Migrator, error) {
	// Создаем DSN для подключения к базе данных
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Db,
	)

	// Открываем подключение к базе данных
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Создаем драйвер для postgres
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Создаем экземпляр migrate
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{migrate: m}, nil
}

// Up выполняет все неприменённые миграции
func (m *Migrator) Up() error {
	slog.Info("Running migrations up...")

	err := m.migrate.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("No migrations to apply")
	} else {
		slog.Info("Migrations applied successfully")
	}

	return nil
}

// Down откатывает все миграции
func (m *Migrator) Down() error {
	slog.Info("Running migrations down...")

	err := m.migrate.Down()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations down: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("No migrations to rollback")
	} else {
		slog.Info("Migrations rolled back successfully")
	}

	return nil
}

// Steps выполняет указанное количество миграций
// Положительное число - применить миграции, отрицательное - откатить
func (m *Migrator) Steps(n int) error {
	slog.Info("Running migrations", "steps", n)

	err := m.migrate.Steps(n)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("No migrations to apply")
	} else {
		slog.Info("Migration steps completed", "steps", n)
	}

	return nil
}

// Version возвращает текущую версию миграции
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// Force устанавливает версию миграции принудительно (для исправления dirty state)
func (m *Migrator) Force(version int) error {
	slog.Warn("Forcing migration version", "version", version)

	err := m.migrate.Force(version)
	if err != nil {
		return fmt.Errorf("failed to force migration version %d: %w", version, err)
	}

	slog.Info("Migration version forced successfully", "version", version)
	return nil
}

// Close закрывает соединение мигратора
func (m *Migrator) Close() error {
	sourceErr, dbErr := m.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close migration database: %w", dbErr)
	}

	return nil
}
