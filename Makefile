.PHONY: all build build-cli run test bench clean deps dev help rust-build rust-test rust-clean build-rust test-rust run-rust

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

# --- Rust targets ---

# Build the Rust physics_core crate
rust-build:
	@echo "Building Rust physics_core..."
	@cd crates/physics_core && cargo build --release
	@echo "Rust build complete"

# Run Rust tests
rust-test:
	@echo "Testing Rust physics_core..."
	@cd crates/physics_core && cargo test

# Clean Rust build artifacts
rust-clean:
	@cd crates/physics_core && cargo clean

# Build GUI with Rust physics backend
build-rust: deps rust-build
	@echo "Building Solar System Simulator (Rust physics)..."
	@mkdir -p bin
	@CGO_ENABLED=1 go build -tags rust_physics -o bin/solar-system-sim ./cmd/gui
	@echo "Build complete: bin/solar-system-sim (Rust physics enabled)"

# Run tests with Rust physics backend
test-rust: rust-build
	@echo "Running tests with Rust physics backend..."
	@CGO_ENABLED=1 go test -tags rust_physics -v ./...

# Run GUI with Rust physics backend
run-rust: build-rust
	@echo "Launching Solar System Simulator (Rust physics)..."
	@./bin/solar-system-sim

# --- Clean ---

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@if [ -d crates/physics_core ]; then cd crates/physics_core && cargo clean 2>/dev/null; fi
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
	@echo ""
	@echo "Rust physics backend:"
	@echo "  make rust-build - Build the Rust physics_core crate"
	@echo "  make rust-test  - Run Rust unit tests"
	@echo "  make build-rust - Build GUI with Rust physics backend"
	@echo "  make test-rust  - Run Go tests with Rust physics backend"
	@echo "  make run-rust   - Build and run GUI with Rust physics"
