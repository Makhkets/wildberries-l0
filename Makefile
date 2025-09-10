.PHONY: help up down watch logs clean restart rebuild stop start db-shell

BIN_NAME=main
EXT=
ifeq ($(OS),Windows_NT)
    EXT=.exe
endif

help: ## Показать все доступные команды
	@chcp 65001 >nul 2>&1
	@echo "Commands:"
	@echo "  help          - Показать все доступные команды"
	@echo "  up            - Запустить все сервисы"
	@echo "  down          - Остановить все сервисы"
	@echo "  watch         - Запустить с live-reload (air)"
	@echo "  logs          - Показать логи всех сервисов"
	@echo "  clean         - Остановить сервисы и удалить volumes"
	@echo "  restart       - Перезапустить все сервисы"
	@echo "  rebuild       - Пересобрать и запустить заново"
	@echo "  stop          - Остановить все сервисы"
	@echo "  start         - Запустить остановленные сервисы"
	@echo "  db-shell      - Подключиться к PostgreSQL через psql"

run:
	go run ./cmd/main/ .

up: ## Запустить все сервисы
	docker-compose up -d

down: ## Остановить все сервисы
	docker-compose down

watch: ## Запустить с live-reload (air)
	docker-compose up --watch

logs: ## Показать логи приложения
	docker-compose logs -f app

clean: ## Остановить сервисы и удалить volumes
	docker-compose down -v

restart: ## Перезапустить все сервисы
	docker-compose restart

rebuild: ## Пересобрать приложение и запустить заново
	docker-compose down
	docker-compose build app --no-cache
	docker-compose up -d

stop: ## Остановить все сервисы
	docker-compose stop

start: ## Запустить остановленные сервисы
	docker-compose start

db-shell: ## Подключиться к PostgreSQL через psql
	docker-compose exec postgres psql -U postgres -d orders
