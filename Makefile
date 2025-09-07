.PHONY: help build up down watch logs clean restart rebuild dev dev-up dev-watch stop start ps logs-app build-app db-shell migrate-up migrate-down migrate-version migrate-steps migrate-force migrate-create build-migrate fix-postgres kafka-topics test-services

BIN_NAME=main
EXT=
ifeq ($(OS),Windows_NT)
    EXT=.exe
endif

help: ## Показать все доступные команды
	@chcp 65001 >nul 2>&1
	@echo "Доступные команды:"
	@echo "  help          - Показать все доступные команды"
	@echo "  build         - Собрать все сервисы"
	@echo "  build-app     - Собрать только Go приложение"
	@echo "  build-migrate - Собрать CLI для миграций"
	@echo "  up            - Запустить все сервисы"
	@echo "  down          - Остановить все сервисы"
	@echo "  watch         - Запустить с live-reload (air)"
	@echo "  dev           - Запустить в development режиме с live-reload"
	@echo "  dev-up        - Запустить все сервисы в development режиме с hot reload"
	@echo "  dev-watch     - Запустить development режим с Docker watch (hot reload)"
	@echo "  logs          - Показать логи всех сервисов"
	@echo "  logs-app      - Показать логи только приложения"
	@echo "  clean         - Остановить сервисы и удалить volumes"
	@echo "  restart       - Перезапустить все сервисы"
	@echo "  rebuild       - Пересобрать и запустить заново"
	@echo "  stop          - Остановить все сервисы"
	@echo "  start         - Запустить остановленные сервисы"
	@echo "  ps            - Показать статус сервисов"
	@echo "  db-shell      - Подключиться к PostgreSQL через psql"
	@echo ""
	@echo "Команды миграций:"
	@echo "  migrate-create NAME - Создать новую миграцию (использование: make migrate-create NAME=create_users_table)"
	@echo "  migrate-up      - Применить все неприменённые миграции"
	@echo "  migrate-down    - Откатить все миграции"
	@echo "  migrate-version - Показать текущую версию миграции"
	@echo "  migrate-steps N - Применить/откатить N миграций (migrate-steps 2 или migrate-steps -1)"
	@echo "  migrate-force N - Принудительно установить версию миграции N"
	@echo ""
	@echo "Команды устранения неполадок:"
	@echo "  fix-postgres    - Исправить проблемы с PostgreSQL (пересоздать с чистыми данными)"
	@echo "  kafka-topics    - Показать топики Kafka"
	@echo "  test-services   - Проверить состояние всех сервисов"

build-app: ## Собрать приложение
	go build -o ./tmp/$(BIN_NAME)$(EXT) ./cmd/main

build-migrate: ## Собрать CLI для миграций
	go build -o ./tmp/migrate$(EXT) ./cmd/migrate

build: ## Собрать все сервисы
	docker-compose build

up: ## Запустить все сервисы
	docker-compose up -d

down: ## Остановить все сервисы
	docker-compose down

watch: ## Запустить с live-reload (air)
	docker-compose up --build --watch

dev: ## Запустить в development режиме с live-reload
	docker-compose -f docker-compose.yml up --build --watch


dev-watch: ## Запустить development режим с Docker watch (hot reload)
	docker-compose -f docker-compose.yml watch

logs: ## Показать логи всех сервисов
	docker-compose logs -f

logs-app: ## Показать логи только приложения
	docker-compose logs -f app

clean: ## Остановить сервисы и удалить volumes
	docker-compose down -v

restart: ## Перезапустить все сервисы
	docker-compose restart

rebuild: ## Пересобрать и запустить заново
	docker-compose down
	docker-compose build --no-cache
	docker-compose up -d

stop: ## Остановить все сервисы
	docker-compose stop

start: ## Запустить остановленные сервисы
	docker-compose start

ps: ## Показать статус сервисов
	docker-compose ps

db-shell: ## Подключиться к PostgreSQL через psql
	docker-compose exec postgres psql -U postgres -d orders

migrate-create: build-migrate ## Создать новую миграцию (использование: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Использование: make migrate-create NAME=migration_name"; \
		echo "Пример: make migrate-create NAME=create_users_table"; \
		exit 1; \
	fi
	./tmp/migrate$(EXT) create $(NAME)

migrate-up: build-migrate ## Применить все неприменённые миграции
	./tmp/migrate$(EXT) up

migrate-down: build-migrate ## Откатить все миграции
	./tmp/migrate$(EXT) down

migrate-version: build-migrate ## Показать текущую версию миграции
	./tmp/migrate$(EXT) version

migrate-steps: build-migrate ## Применить/откатить N миграций (использование: make migrate-steps STEPS=2)
	@if [ -z "$(STEPS)" ]; then \
		echo "Использование: make migrate-steps STEPS=N"; \
		echo "Пример: make migrate-steps STEPS=2"; \
		echo "Пример: make migrate-steps STEPS=-1"; \
		exit 1; \
	fi
	./tmp/migrate$(EXT) steps $(STEPS)

migrate-force: build-migrate ## Принудительно установить версию миграции (использование: make migrate-force VERSION=N)
	@if [ -z "$(VERSION)" ]; then \
		echo "Использование: make migrate-force VERSION=N"; \
		echo "Пример: make migrate-force VERSION=5"; \
		exit 1; \
	fi
	./tmp/migrate$(EXT) force $(VERSION)

fix-postgres: ## Исправить проблемы с PostgreSQL
	@echo "Останавливаем сервисы..."
	docker-compose down
	@echo "Удаляем PostgreSQL volume..."
	-docker volume rm wildberries_postgres_data
	@echo "Пересоздаем PostgreSQL с чистыми данными..."
	docker-compose up --build -d postgres
	@echo "Ожидаем готовности PostgreSQL..."
	@timeout 60 bash -c 'until docker-compose exec postgres pg_isready -U postgres; do sleep 2; done' && echo "PostgreSQL готов!" || echo "PostgreSQL не запустился"

kafka-topics: ## Показать топики Kafka
	docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

test-services: ## Проверить состояние всех сервисов
	@echo "Проверяем состояние сервисов..."
	@echo "PostgreSQL:"
	@docker-compose exec postgres pg_isready -U postgres && echo "✅ PostgreSQL готов" || echo "❌ PostgreSQL не готов"
	@echo "Redis:"
	@docker-compose exec redis redis-cli ping && echo "✅ Redis готов" || echo "❌ Redis не готов"
	@echo "Kafka:"
	@docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list > /dev/null 2>&1 && echo "✅ Kafka готов" || echo "❌ Kafka не готов"
	@echo "Проверка завершена!"
