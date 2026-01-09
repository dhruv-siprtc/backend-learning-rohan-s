.PHONY: help build run test clean docker-up docker-down docker-logs migrate db-create db-drop lint fmt

# Variables
APP_NAME := user-management-system
API_BINARY := api
CONSUMER_BINARY := consumer
GO := go
DOCKER_COMPOSE := docker-compose

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build both API and Consumer binaries
build:
	@echo "🔨 Building binaries..."
	$(GO) build -o ./bin/$(API_BINARY) ./cmd/api
	$(GO) build -o ./bin/$(CONSUMER_BINARY) ./cmd/consumer
	@echo "✅ Build complete"

## run-api: Run API server locally
run-api:
	@echo "🚀 Starting API server..."
	$(GO) run ./cmd/api/main.go

## run-consumer: Run consumer locally
run-consumer:
	@echo "🎧 Starting consumer..."
	$(GO) run ./cmd/consumer/main.go

## test: Run all tests
test:
	@echo "🧪 Running tests..."
	@export $$(cat .env.test | xargs) && $(GO) test ./tests/... -v

## test-coverage: Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	@export $$(cat .env.test | xargs) && $(GO) test ./tests/... -v -cover -coverprofile=coverage.out
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

## test-integration: Run integration tests only
test-integration:
	@echo "🔄 Running integration tests..."
	@export $$(cat .env.test | xargs) && $(GO) test ./tests/... -v -run Integration

## docker-up: Start all services with Docker Compose
docker-up:
	@echo "🐳 Starting Docker services..."
	$(DOCKER_COMPOSE) up -d
	@echo "✅ Services started"
	@echo "📍 API: http://localhost:8080"
	@echo "📍 RabbitMQ: http://localhost:15672"

## docker-build: Build Docker images
docker-build:
	@echo "🔨 Building Docker images..."
	$(DOCKER_COMPOSE) build
	@echo "✅ Images built"

## docker-down: Stop all Docker services
docker-down:
	@echo "🛑 Stopping Docker services..."
	$(DOCKER_COMPOSE) down
	@echo "✅ Services stopped"

## docker-down-volumes: Stop services and remove volumes
docker-down-volumes:
	@echo "🗑️  Stopping services and removing volumes..."
	$(DOCKER_COMPOSE) down -v
	@echo "✅ Services stopped and volumes removed"

## docker-logs: Show logs from all services
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-logs-api: Show API logs
docker-logs-api:
	$(DOCKER_COMPOSE) logs -f api

## docker-logs-consumer: Show consumer logs
docker-logs-consumer:
	$(DOCKER_COMPOSE) logs -f consumer

## docker-restart: Restart all services
docker-restart: docker-down docker-up

## migrate: Run database migrations
migrate:
	@echo "🗄️  Running database migrations..."
	$(GO) run ./cmd/migrate/main.go
	@echo "✅ Migrations complete"

## db-create: Create test database
db-create:
	@echo "📦 Creating test database..."
	docker-compose exec postgres psql -U postgres -c "CREATE DATABASE postgis_36_sample_test;"
	@echo "✅ Test database created"

## db-drop: Drop test database
db-drop:
	@echo "🗑️  Dropping test database..."
	docker-compose exec postgres psql -U postgres -c "DROP DATABASE IF EXISTS postgis_36_sample_test;"
	@echo "✅ Test database dropped"

## db-reset: Reset test database (drop and create)
db-reset: db-drop db-create

## lint: Run linter
lint:
	@echo "🔍 Running linter..."
	golangci-lint run ./...
	@echo "✅ Linting complete"

## fmt: Format code
fmt:
	@echo "💅 Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Formatting complete"

## clean: Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf ./bin
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

## deps: Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependencies installed"

## rabbitmq-queues: List RabbitMQ queues
rabbitmq-queues:
	@echo "📊 RabbitMQ Queues:"
	docker-compose exec rabbitmq rabbitmqctl list_queues name messages consumers

## rabbitmq-bindings: List RabbitMQ bindings
rabbitmq-bindings:
	@echo "🔗 RabbitMQ Bindings:"
	docker-compose exec rabbitmq rabbitmqctl list_bindings

## rabbitmq-purge: Purge all RabbitMQ queues
rabbitmq-purge:
	@echo "🗑️  Purging RabbitMQ queues..."
	docker-compose exec rabbitmq rabbitmqctl purge_queue user.created.queue
	docker-compose exec rabbitmq rabbitmqctl purge_queue user.updated.queue
	@echo "✅ Queues purged"

## health: Check service health
health:
	@echo "🏥 Checking service health..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""

## load-test: Run load tests
load-test:
	@echo "⚡ Running load tests..."
	ab -n 1000 -c 10 -p tests/fixtures/user.json -T application/json http://localhost:8080/users
	@echo "✅ Load test complete"

## dev: Start development environment
dev: docker-up
	@echo "🚀 Development environment ready!"
	@echo "📍 API: http://localhost:8080"
	@echo "📍 RabbitMQ Management: http://localhost:15672 (guest/guest)"
	@echo "📍 PostgreSQL: localhost:5432 (postgres/123)"
	@echo ""
	@echo "Run 'make docker-logs' to see logs"

## prod-build: Build production images
prod-build:
	@echo "🏭 Building production images..."
	docker build -f cmd/api/Dockerfile -t $(APP_NAME)-api:latest .
	docker build -f cmd/consumer/Dockerfile -t $(APP_NAME)-consumer:latest .
	@echo "✅ Production images built"

## install-tools: Install development tools
install-tools:
	@echo "🔧 Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "✅ Tools installed"

## git-hooks: Install git hooks
git-hooks:
	@echo "🪝 Installing git hooks..."
	@echo "#!/bin/sh\nmake fmt\nmake lint" > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Git hooks installed"

## all: Run tests, lint, and build
all: fmt lint test build
	@echo "✅ All checks passed"

## ci: Run CI pipeline (used in CI/CD)
ci: deps fmt lint test-coverage
	@echo "✅ CI pipeline complete"

# Default target
.DEFAULT_GOAL := help