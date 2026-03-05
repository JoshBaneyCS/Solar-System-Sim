.PHONY: all build run clean deps

# Default target
all: deps build run

# Install dependencies
deps:
	@echo "📥 Downloading dependencies..."
	@go mod tidy

# Build the application
build: deps
	@echo "🔨 Building Solar System Simulator..."
	@go build -o solar_system_sim solar_system_sim.go
	@echo "✅ Build complete!"

# Run the application
run: build
	@echo "🚀 Launching Solar System Simulator..."
	@./solar_system_sim

# Quick run without rebuild
quick:
	@./solar_system_sim

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f solar_system_sim
	@echo "✅ Clean complete!"

# Development build (with race detector)
dev:
	@echo "🔨 Building with race detector..."
	@go build -race -o solar_system_sim solar_system_sim.go
	@./solar_system_sim

# Help
help:
	@echo "Solar System Simulator - Makefile targets:"
	@echo ""
	@echo "  make          - Download deps, build, and run"
	@echo "  make build    - Build the application"
	@echo "  make run      - Build and run the application"
	@echo "  make quick    - Run without rebuilding"
	@echo "  make deps     - Download dependencies"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make dev      - Build with race detector"
	@echo "  make help     - Show this help message"