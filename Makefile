# Makefile for open-db9 project

.PHONY: build test run clean install docker-build docker-run docker-stop dev-up dev-down rag-bootstrap rag-demo fmt vet lint test-e2e benchmark build-mcp test-rag build-all

# Variables
BINARY_NAME=db9
SERVER_BINARY_NAME=db9-server
FS9_BINARY_NAME=fs9-service
CMD_DIR=./cmd
BUILD_DIR=./build
GO=go
GOFLAGS=-v
DOCKER_COMPOSE=docker-compose

# Build targets
build: build-cli build-server build-fs9 build-mcp

build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/db9/main.go

build-server:
	@echo "Building API server..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY_NAME) $(CMD_DIR)/server/main.go

build-fs9:
	@echo "Building FS9 service..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(FS9_BINARY_NAME) $(CMD_DIR)/fs9-service/main.go

build-mcp:
	@echo "Building MCP server..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/mcp-server $(CMD_DIR)/mcp-server/main.go

build-all: build
	@echo "All components built successfully."

# Test targets
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-e2e:
	@echo "Running E2E tests..."
	$(GO) test -v -race ./e2e/...

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

test-rag:
	@echo "Running RAG Python tests..."
	cd rag && python -m pytest tests/ -v --tb=short

# Run targets
run-cli: build-cli
	@echo "Running CLI..."
	./$(BUILD_DIR)/$(BINARY_NAME)

run-server: build-server
	@echo "Running API server..."
	./$(BUILD_DIR)/$(SERVER_BINARY_NAME)

# Development targets
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

vet:
	@echo "Vetting code..."
	$(GO) vet ./...

lint: fmt vet

# Clean target
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install target
install: build
	@echo "Installing binaries..."
	$(GO) install $(CMD_DIR)/db9/main.go
	$(GO) install $(CMD_DIR)/server/main.go

# Docker targets
docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml build

docker-run:
	@echo "Running Docker containers..."
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml up -d

docker-stop:
	@echo "Stopping Docker containers..."
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml down

dev-up:
	@echo "Starting local development stack..."
	$(DOCKER_COMPOSE) --env-file deployments/docker/.env.example -f deployments/docker/docker-compose.yml up -d --build

dev-down:
	@echo "Stopping local development stack..."
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml down

rag-bootstrap:
	@python rag/scripts/bootstrap_local_rag.py

rag-demo:
	@python rag/scripts/e2e_api_demo.py

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build CLI and server binaries"
	@echo "  build-cli       - Build CLI binary only"
	@echo "  build-server    - Build server binary only"
	@echo "  build-fs9       - Build FS9 service binary only"
	@echo "  build-mcp        - Build MCP server binary only"
	@echo "  build-all        - Build all components (cli + server + fs9 + mcp)"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  benchmark       - Run benchmarks"
	@echo "  test-rag        - Run RAG Python tests"
	@echo "  run-cli         - Build and run CLI"
	@echo "  run-server      - Build and run server"
	@echo "  fmt             - Format code"
	@echo "  vet             - Vet code"
	@echo "  lint            - Run fmt and vet"
	@echo "  clean           - Remove build artifacts"
	@echo "  install         - Install binaries to GOBIN"
	@echo "  docker-build    - Build Docker images"
	@echo "  docker-run      - Run Docker containers"
	@echo "  docker-stop     - Stop Docker containers"
	@echo "  dev-up          - Start local development stack with Docker"
	@echo "  dev-down        - Stop local development stack"
	@echo "  rag-bootstrap   - Create local tenant/database context and JWT"
	@echo "  rag-demo        - Run the local RAG API end-to-end demo"
	@echo "  deps            - Download and tidy dependencies"
	@echo "  help            - Show this help message"
