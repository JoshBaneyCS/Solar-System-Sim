.PHONY: all build build-cli run test bench clean deps dev help

# Default target
all: deps build

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod tidy

# Build the GUI application
build: deps
	@echo "Building Solar System Simulator (GUI)..."
	@mkdir -p bin
	@go build -o bin/solar-system-sim ./cmd/gui
	@echo "Build complete: bin/solar-system-sim"

# Build the CLI application
build-cli: deps
	@echo "Building Solar System Simulator (CLI)..."
	@mkdir -p bin
	@go build -o bin/solar-system-cli ./cmd/cli
	@echo "Build complete: bin/solar-system-cli"

# Run the GUI application
run: build
	@echo "Launching Solar System Simulator..."
	@./bin/solar-system-sim

# Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./internal/physics/...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "Clean complete!"

# Development build (with race detector)
dev:
	@echo "Building with race detector..."
	@mkdir -p bin
	@go build -race -o bin/solar-system-sim ./cmd/gui
	@./bin/solar-system-sim

# Vet all packages
vet:
	@go vet ./...

# Help
help:
	@echo "Solar System Simulator - Makefile targets:"
	@echo ""
	@echo "  make           - Download deps and build GUI"
	@echo "  make build     - Build the GUI application"
	@echo "  make build-cli - Build the CLI application"
	@echo "  make run       - Build and run the GUI"
	@echo "  make test      - Run all tests"
	@echo "  make bench     - Run benchmarks"
	@echo "  make deps      - Download dependencies"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make dev       - Build with race detector and run"
	@echo "  make vet       - Run go vet on all packages"
	@echo "  make help      - Show this help message"
