# Makefile for Seckill System

# Variables
APP_NAME = seckill-system
VERSION = v1.0.0
BUILD_DIR = bin
GO_VERSION = 1.21

# Go related variables
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOFMT = gofmt
GOLINT = golangci-lint

# Build flags
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date +%Y-%m-%d_%H:%M:%S)"

# Default target
.PHONY: all
all: clean deps fmt lint test build

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	$(GOLINT) run ./...

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build the application
.PHONY: build
build:
	@echo "Building application..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/api ./cmd/api
	@echo "Build completed: $(BUILD_DIR)/api"

# Build all services
.PHONY: build-all
build-all:
	@echo "Building all services..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/api ./cmd/api
	@echo "All services built successfully"

# Run the API service
.PHONY: run
run:
	@echo "Starting API service..."
	$(GOCMD) run ./cmd/api

# Run with hot reload (requires air)
.PHONY: dev
dev:
	@echo "Starting development server with hot reload..."
	air -c .air.toml

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Docker commands
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --name $(APP_NAME) $(APP_NAME):latest

.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

# Docker Compose commands
.PHONY: up
up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

.PHONY: down
down:
	@echo "Stopping services with Docker Compose..."
	docker-compose down

.PHONY: logs
logs:
	@echo "Showing Docker Compose logs..."
	docker-compose logs -f

# Database commands
.PHONY: db-init
db-init:
	@echo "Initializing database..."
	mysql -h localhost -P 3306 -u root -p < scripts/sql/schema.sql

.PHONY: db-migrate
db-migrate:
	@echo "Running database migrations..."
	migrate -path scripts/migrate -database "mysql://root:root123@tcp(localhost:3306)/seckill" up

.PHONY: db-rollback
db-rollback:
	@echo "Rolling back database migrations..."
	migrate -path scripts/migrate -database "mysql://root:root123@tcp(localhost:3306)/seckill" down 1

# Redis commands
.PHONY: redis-cli
redis-cli:
	@echo "Connecting to Redis..."
	redis-cli -h localhost -p 6379

# Load testing
.PHONY: load-test
load-test:
	@echo "Running load tests..."
	k6 run tests/k6/seckill.js

# Benchmark
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Security scan
.PHONY: security
security:
	@echo "Running security scan..."
	gosec ./...

# Generate API documentation
.PHONY: docs
docs:
	@echo "Generating API documentation..."
	swag init -g cmd/api/main.go -o docs/swagger

# Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/cosmtrek/air@latest
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) -u github.com/swaggo/swag/cmd/swag@latest
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Health check
.PHONY: health
health:
	@echo "Checking service health..."
	curl -f http://localhost:8080/health || echo "Service is not healthy"

# Show help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  all          - Run clean, deps, fmt, lint, test, build"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build all services"
	@echo "  run          - Run the API service"
	@echo "  dev          - Run with hot reload"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-stop  - Stop Docker container"
	@echo "  up           - Start services with Docker Compose"
	@echo "  down         - Stop services with Docker Compose"
	@echo "  logs         - Show Docker Compose logs"
	@echo "  db-init      - Initialize database"
	@echo "  db-migrate   - Run database migrations"
	@echo "  db-rollback  - Rollback database migrations"
	@echo "  redis-cli    - Connect to Redis"
	@echo "  load-test    - Run load tests"
	@echo "  bench        - Run benchmarks"
	@echo "  security     - Run security scan"
	@echo "  docs         - Generate API documentation"
	@echo "  install-tools- Install development tools"
	@echo "  health       - Check service health"
	@echo "  help         - Show this help message"