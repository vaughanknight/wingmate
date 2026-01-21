.PHONY: build build-all test lint clean help

# Binary name
BINARY := wingmate

# Build output directory
BUILD_DIR := bin

# Go build flags
LDFLAGS := -s -w

# Default target
all: build

# Build for current platform
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/wingmate

# Cross-compile for all platforms
build-all:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/wingmate
	@echo "Building for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/wingmate
	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/wingmate
	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/wingmate
	@echo "Building for windows/amd64..."
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/wingmate
	@echo "Building for android/arm64..."
	GOOS=android GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-android-arm64 ./cmd/wingmate
	@echo "Done! Binaries in $(BUILD_DIR)/"

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race -v ./...

# Run integration tests only
test-integration:
	go test -v ./tests/integration/...

# Run linter
lint:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f flight.jsonl

# Show help
help:
	@echo "Wingmate Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          Build for current platform"
	@echo "  make build-all      Cross-compile for all platforms"
	@echo "  make test           Run all tests"
	@echo "  make test-race      Run tests with race detector"
	@echo "  make test-integration  Run integration tests only"
	@echo "  make lint           Run linter"
	@echo "  make clean          Remove build artifacts"
	@echo "  make help           Show this help"
