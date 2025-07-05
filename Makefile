# Build variables
APP = k8s-cli
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_FLAGS = -v -o bin/$(APP) -ldflags "-X=github.com/obezsmertnyi/k8s-custom-controller/cmd.appVersion=$(VERSION)"
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")
GO_PACKAGES=$(shell go list ./...)

# Colors for terminal output
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build test run docker-build clean lint coverage test-server test-logging help

all: clean lint test build

build:
	@echo "$(BLUE)🔨 Building $(APP)...$(NC)"
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) main.go
	@echo "$(GREEN)✅ Build complete: bin/$(APP)$(NC)"

test:
	@echo "$(BLUE)🧪 Running tests...$(NC)"
	go test -v -race -cover $(GO_PACKAGES)
	@echo "$(GREEN)✅ Tests complete$(NC)"

run:
	@echo "$(BLUE)🚀 Running server...$(NC)"
	go run main.go server

docker-build:
	@echo "$(BLUE)🐳 Building Docker image...$(NC)"
	docker build --build-arg VERSION=$(VERSION) -t $(APP):latest .
	@echo "$(GREEN)✅ Docker build complete$(NC)"

clean:
	@echo "$(YELLOW)🧹 Cleaning...$(NC)"
	rm -rf bin/
	@echo "$(GREEN)✅ Clean complete$(NC)"

lint:
	@echo "$(BLUE)🔍 Linting code...$(NC)"
	go vet ./...
	@echo "$(GREEN)✅ Lint complete$(NC)"

coverage:
	@echo "$(BLUE)📊 Generating test coverage report...$(NC)"
	go test -coverprofile=coverage.out $(GO_PACKAGES)
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"

test-server: build
	@echo "$(BLUE)🧪 Testing server component...$(NC)"
	go test -v ./tests/server_test.go
	@echo "$(GREEN)✅ Server tests complete$(NC)"

test-logging: build
	@echo "$(BLUE)🧪 Testing logging component...$(NC)"
	go test -v ./tests/logging_test.go
	@echo "$(GREEN)✅ Logging tests complete$(NC)"

docker-run: docker-build
	@echo "$(BLUE)🚀 Running Docker container...$(NC)"
	docker run --rm -p 8080:8080 $(APP):latest

help:
	@echo "$(BLUE)📚 Available commands:$(NC)"
	@echo "  all          : Clean, lint, test, and build"
	@echo "  build        : Build the binary"
	@echo "  clean        : Remove build artifacts"
	@echo "  lint         : Run linters"
	@echo "  test         : Run all tests"
	@echo "  coverage     : Generate test coverage report"
	@echo "  docker-build : Build Docker image"
	@echo "  docker-run   : Run in Docker container"
	@echo "  run          : Build and run the server"
	@echo "  test-server  : Run server tests"
	@echo "  test-logging : Run logging tests"
	@echo "  help         : Show this help message"