# Contributing

## Prerequisites

- **Go 1.21+** — [go.dev/dl](https://go.dev/dl/)
- **Make** — GNU Make (macOS: Xcode CLI tools, Linux: `build-essential`, Windows: `choco install make`)
- **Fyne system dependencies** — see [Fyne Getting Started](https://docs.fyne.io/started/) for platform-specific requirements (X11/Wayland headers on Linux, Xcode on macOS)
- **Rust toolchain** (optional) — only needed for `rust_physics` or `rust_render` features. Install via [rustup.rs](https://rustup.rs/)

## Getting Started

```bash
git clone <repo-url>
cd solar-system-simulator
make deps        # go mod tidy
make build       # build GUI binary
make test        # run all Go tests
```

## Project Structure

```
cmd/            Entry points (gui, solar-sim, meshgen, validate-assets)
internal/       Private packages
  physics/      N-body simulation core (Verlet, RK4, GR corrections)
  render/       Fyne canvas renderer + optional GPU renderer
  ui/           Fyne GUI (app, menu, settings, bodies panel)
  launch/       Orbital mechanics and launch planning
  ffi/          Go-Rust FFI bindings
  validation/   Physics validation suite
  math3d/       Vector math (Vec3, Catmull-Rom)
  spacetime/    GR spacetime fabric visualization
  viewport/     Camera, zoom, pan, 3D rotation
  assets/       Asset validation
pkg/
  constants/    Physical constants (G, AU, c)
crates/         Rust crates
  physics_core/ Rust physics engine (optional backend)
  render_core/  Rust GPU renderer via wgpu (optional)
docs/           Documentation
assets/         Runtime assets (textures, models, meshes)
```

## Development Workflow

### Branch Naming

Use descriptive prefixes:
- `feature/` — new functionality
- `fix/` — bug fixes
- `docs/` — documentation changes
- `refactor/` — code restructuring

### Building

| Command | What it builds |
|---------|---------------|
| `make build` | Go-only GUI binary |
| `make build-solar-sim` | Unified CLI with GUI |
| `make build-solar-sim-headless` | Headless CLI (no GUI, `-tags nogui`) |
| `make build-rust` | Go + Rust physics backend |
| `make build-gpu` | Go + Rust physics + GPU rendering |

### Running

| Command | Description |
|---------|------------|
| `make run` | Build and run GUI |
| `make run-gpu` | Build and run with GPU rendering |
| `make dev` | Run with race detector enabled |

### Testing

| Command | Scope |
|---------|-------|
| `make test` | All Go tests |
| `make test-rust` | Go tests with Rust physics backend |
| `make test-gpu` | Go tests with GPU rendering |
| `make bench` | Benchmarks on physics package |

### Linting

```bash
make lint        # gofmt check + go vet + cargo clippy (if Rust crates present)
```

See [CODE_STYLE.md](CODE_STYLE.md) for detailed style conventions.

## Build Tags

The project uses Go build tags to conditionally compile features:

| Tag | Purpose | Required toolchain |
|-----|---------|-------------------|
| `nogui` | Exclude Fyne GUI (headless build) | Go only |
| `rust_physics` | Use Rust physics backend via FFI | Go + Rust + CGO |
| `rust_render` | Use GPU rendering via wgpu | Go + Rust + CGO |

Every feature-gated file has a `_noop.go` counterpart that provides stub implementations when the feature is disabled. When adding a new build-tag-gated feature, always create both files.

## FFI Changes

When modifying the Go-Rust boundary, all three layers must be updated together:

1. **C header** — `crates/<name>/include/<name>.h`
2. **Rust FFI** — `crates/<name>/src/ffi.rs`
3. **Go CGo bindings** — `internal/ffi/<name>_rust.go`

Test with both the Go-only and Rust-enabled builds:
```bash
make test          # Go-only
make test-rust     # With Rust physics
```

See [FFI.md](FFI.md) for the full FFI design and [SECURITY.md](SECURITY.md) for safety guidelines.

## Adding a New Package

- Place in `internal/` unless it needs to be importable by external projects
- Keep packages focused on a single responsibility
- Do not import `internal/ui` from `internal/physics` or vice versa — communicate through the `Simulator` API
- Add tests in `_test.go` files within the same package

## Testing Guidelines

- Write tests in the same package (`package physics`, not `package physics_test`)
- Use existing assertion helpers from `test_helpers_test.go`: `assertRelativeError`, `assertFloat64Near`
- Never compare floats with `==` — always use tolerance-based comparison
- For physics changes: add or update validation scenarios in `internal/validation/`
- For performance-sensitive changes: run `make bench` before and after
- Update golden test data (`internal/physics/golden_test.go`) if physics behavior intentionally changes

## Refactoring Guidelines

Before refactoring:
- [ ] `make test` baseline passes
- [ ] Identify which build tag variants are affected

During refactoring:
- [ ] Preserve the `PhysicsBackend` interface contract
- [ ] Keep `_noop.go` fallbacks in sync with feature-gated files
- [ ] UI code should not import `internal/physics` directly — go through the simulator API
- [ ] Keep the FFI boundary minimal — add new FFI functions only when needed

After refactoring:
- [ ] `make test` passes
- [ ] `make test-rust` passes (if FFI or physics changed)
- [ ] `make test-gpu` passes (if rendering changed)
- [ ] `make bench` shows no unexpected regressions
- [ ] `make lint` passes

## Commit Messages

Use conventional commit style:

```
feat: add asteroid belt simulation
fix: correct Mercury GR precession coefficient
docs: update CLI reference for new export flags
refactor: extract trail rendering into separate method
test: add energy conservation benchmark
```

## Pull Requests

Use the [PR template](../.github/pull_request_template.md) when opening pull requests. The template includes a checklist covering tests, formatting, FFI consistency, and documentation.
