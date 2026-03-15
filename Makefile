.PHONY: all build build-cli build-solar-sim build-solar-sim-headless run test bench clean deps dev help lint vet package-macos package-linux package-windows rust-build rust-test rust-clean build-rust test-rust run-rust render-build build-gpu run-gpu test-gpu assets-setup meshgen validate-assets

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

# Build the unified CLI (with GUI support)
build-solar-sim: deps
	@echo "Building Solar System Simulator (unified CLI)..."
	@mkdir -p bin
	@go build -o bin/solar-sim ./cmd/solar-sim
	@echo "Build complete: bin/solar-sim"

# Build the unified CLI without GUI (headless only, no graphics deps)
build-solar-sim-headless: deps
	@echo "Building Solar System Simulator (headless CLI)..."
	@mkdir -p bin
	@go build -tags nogui -o bin/solar-sim ./cmd/solar-sim
	@echo "Build complete: bin/solar-sim (headless, no GUI deps)"

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

# --- GPU rendering targets ---

# Build the Rust render_core crate
render-build:
	@echo "Building Rust render_core..."
	@cd crates/render_core && cargo build --release
	@echo "Rust render_core build complete"

# Build GUI with Rust physics + GPU rendering
build-gpu: deps rust-build render-build
	@echo "Building Solar System Simulator (Rust physics + GPU rendering)..."
	@mkdir -p bin
	@CGO_ENABLED=1 go build -tags "rust_physics,rust_render" -o bin/solar-system-sim ./cmd/gui
	@echo "Build complete: bin/solar-system-sim (GPU rendering enabled)"

# Run GUI with GPU rendering
run-gpu: build-gpu
	@echo "Launching Solar System Simulator (GPU rendering)..."
	@./bin/solar-system-sim

# Run tests with GPU rendering
test-gpu: rust-build render-build
	@echo "Running tests with GPU rendering..."
	@CGO_ENABLED=1 go test -tags "rust_physics,rust_render" -v ./...

# --- Asset pipeline targets ---

# Set up asset directory structure from source textures
assets-setup:
	@echo "Setting up asset directory structure..."
	@mkdir -p assets/textures/{sun,mercury,venus,earth,mars,jupiter,saturn,uranus,neptune,skybox}
	@mkdir -p assets/models assets/meshes
	@cp -n space-object-textures/8k_sun.jpg assets/textures/sun/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/8k_mercury.jpg assets/textures/mercury/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/4k_venus_atmosphere.jpg assets/textures/venus/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/8k_mars.jpg assets/textures/mars/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/8k_jupiter.jpg assets/textures/jupiter/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/8k_saturn.jpg assets/textures/saturn/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/8k_saturn_ring_alpha.png assets/textures/saturn/ring_alpha.png 2>/dev/null || true
	@cp -n space-object-textures/2k_uranus.jpg assets/textures/uranus/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/2k_neptune.jpg assets/textures/neptune/albedo.jpg 2>/dev/null || true
	@cp -n space-object-textures/8k_stars_milky_way.jpg assets/textures/skybox/milky_way.jpg 2>/dev/null || true
	@cp -n space-object-textures/Earth_1_12756.glb assets/models/earth.glb 2>/dev/null || true
	@echo "Asset setup complete"

# Build and run mesh generator
meshgen:
	@echo "Building mesh generator..."
	@mkdir -p bin assets/meshes
	@go build -o bin/meshgen ./cmd/meshgen
	@echo "Generating sphere meshes..."
	@./bin/meshgen --segments 32 --output assets/meshes/sphere_32.glb
	@./bin/meshgen --segments 64 --output assets/meshes/sphere_64.glb
	@echo "Mesh generation complete"

# Validate asset directory
validate-assets:
	@echo "Validating assets..."
	@go run ./cmd/validate-assets --dir assets
	@echo "Asset validation complete"

# --- Clean ---

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@if [ -d crates/physics_core ]; then cd crates/physics_core && cargo clean 2>/dev/null; fi
	@if [ -d crates/render_core ]; then cd crates/render_core && cargo clean 2>/dev/null; fi
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

# Run linters (Go + Rust)
lint:
	@echo "Checking Go formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Files need gofmt:" && gofmt -l . && exit 1)
	@echo "Running go vet..."
	@go vet ./...
	@if [ -d crates/physics_core ]; then echo "Running clippy (physics_core)..." && cd crates/physics_core && cargo clippy 2>/dev/null; fi
	@if [ -d crates/render_core ]; then echo "Running clippy (render_core)..." && cd crates/render_core && cargo clippy 2>/dev/null; fi
	@echo "Lint passed."

# --- Packaging targets ---

# Package macOS .dmg (requires bin/solar-sim and bin/solar-sim-headless)
package-macos:
	@bash packaging/package-macos.sh

# Package Linux .tar.gz (requires bin/solar-sim and bin/solar-sim-headless)
package-linux:
	@bash packaging/package-linux.sh

# Package Windows .zip (requires bin/solar-sim.exe and bin/solar-sim-headless.exe)
package-windows:
	@bash packaging/package-windows.sh

# Verify no unexpected runtime dependencies (macOS)
check-deps:
	@echo "Checking runtime dependencies..."
	@if [ -f bin/solar-system-sim ]; then \
		echo "=== solar-system-sim ===" && \
		otool -L bin/solar-system-sim 2>/dev/null | grep -v /usr/lib | grep -v /System | grep -v "is not an object" || echo "No non-system deps"; \
	fi
	@if [ -f bin/solar-sim ]; then \
		echo "=== solar-sim ===" && \
		otool -L bin/solar-sim 2>/dev/null | grep -v /usr/lib | grep -v /System | grep -v "is not an object" || echo "No non-system deps"; \
	fi
	@echo "Dependency check complete"

# Help
help:
	@echo "Solar System Simulator - Makefile targets:"
	@echo ""
	@echo "  make           - Download deps and build GUI"
	@echo "  make build     - Build the GUI application"
	@echo "  make build-cli - Build the CLI application"
	@echo "  make build-solar-sim          - Build unified CLI (with GUI)"
	@echo "  make build-solar-sim-headless - Build unified CLI (headless, no GUI deps)"
	@echo "  make run       - Build and run the GUI"
	@echo "  make test      - Run all tests"
	@echo "  make bench     - Run benchmarks"
	@echo "  make deps      - Download dependencies"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make dev       - Build with race detector and run"
	@echo "  make vet       - Run go vet on all packages"
	@echo "  make lint      - Run linters (Go formatting + vet + Rust clippy)"
	@echo "  make help      - Show this help message"
	@echo ""
	@echo "Rust physics backend:"
	@echo "  make rust-build  - Build the Rust physics_core crate"
	@echo "  make rust-test   - Run Rust unit tests"
	@echo "  make build-rust  - Build GUI with Rust physics backend"
	@echo "  make test-rust   - Run Go tests with Rust physics backend"
	@echo "  make run-rust    - Build and run GUI with Rust physics"
	@echo ""
	@echo "GPU rendering (Rust wgpu):"
	@echo "  make render-build - Build the Rust render_core crate"
	@echo "  make build-gpu    - Build GUI with Rust physics + GPU rendering"
	@echo "  make run-gpu      - Build and run GUI with GPU rendering"
	@echo "  make test-gpu     - Run tests with GPU rendering"
	@echo ""
	@echo "Packaging:"
	@echo "  make package-macos   - Create macOS .dmg (after building)"
	@echo "  make package-linux   - Create Linux .tar.gz (after building)"
	@echo "  make package-windows - Create Windows .zip (after building)"
	@echo ""
	@echo "Asset pipeline:"
	@echo "  make assets-setup     - Set up asset directory from source textures"
	@echo "  make meshgen          - Generate sphere mesh .glb files"
	@echo "  make validate-assets  - Validate asset directory structure"
