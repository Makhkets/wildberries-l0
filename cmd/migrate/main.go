package main

import (
	"fmt"
	"github.com/makhkets/wildberries-l0/pkg/lib/logger/sl"
	"github.com/makhkets/wildberries-l0/pkg/logging"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/makhkets/wildberries-l0/internal/config"
	"github.com/makhkets/wildberries-l0/internal/migrate"
)

func main() {
	logging.SetupLogger()

	// Загружаем переменные окружения с приоритетом для .env.local
	if _, err := os.Stat(".env"); err == nil {
		if err = godotenv.Load(".env"); err != nil {
			slog.Warn("Failed to load .env.local file", sl.Err(err))
		}
	}
	//} else if _, err := os.Stat(".env"); err == nil {
	//	if err := godotenv.Load(".env"); err != nil {
	//		slog.Warn("Failed to load .env file", sl.Err(err))
	//	}
	//}

	// Парсим аргументы командной строки
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Загружаем конфигурацию
	cfg := config.GetConfig()
	slog.Debug("initializing migrator with configuration", slog.Any("config", cfg))

	// Создаем мигратор
	migrator, err := migrate.NewMigrator(cfg)
	if err != nil {
		slog.Error("Failed to create migrator", sl.Err(err))
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		if err = migrator.Close(); err != nil {
			slog.Error("Failed to close migrator", sl.Err(err))
		}
	}()

	// Выполняем команду
	switch command {
	case "up":
		if err = migrator.Up(); err != nil {
			slog.Error("Migration up failed", sl.Err(err))
			os.Exit(1)
		}
	case "down":
		if err = migrator.Down(); err != nil {
			slog.Error("Migration down failed", sl.Err(err))
			os.Exit(1)
		}
	case "steps":
		if len(os.Args) < 3 {
			fmt.Println("Usage: migrate steps <number>")
			os.Exit(1)
		}
		steps, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Invalid steps number: %s\n", os.Args[2])
			os.Exit(1)
		}
		if err = migrator.Steps(steps); err != nil {
			slog.Error("Migration steps failed", sl.Err(err))
			os.Exit(1)
		}
	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			slog.Error("Failed to get migration version", sl.Err(err))
			os.Exit(1)
		}
		fmt.Printf("Current migration version: %d\n", version)
		if dirty {
			fmt.Println("Migration state is dirty")
		}
	case "force":
		if len(os.Args) < 3 {
			fmt.Println("Usage: migrate force <version>")
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Invalid version number: %s\n", os.Args[2])
			os.Exit(1)
		}
		if err = migrator.Force(version); err != nil {
			slog.Error("Migration force failed", sl.Err(err))
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [arguments]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  up               - Apply all pending migrations")
	fmt.Println("  down             - Rollback all migrations")
	fmt.Println("  steps <number>   - Apply/rollback specific number of migrations")
	fmt.Println("  version          - Show current migration version")
	fmt.Println("  force <version>  - Force set migration version (use with caution)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  migrate up")
	fmt.Println("  migrate down")
	fmt.Println("  migrate steps 2")
	fmt.Println("  migrate steps -1")
	fmt.Println("  migrate version")
	fmt.Println("  migrate force 5")
}
