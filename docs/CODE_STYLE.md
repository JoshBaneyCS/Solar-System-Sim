# Code Style Guide

## Go

### Formatting and Linting

- **Formatter:** `gofmt` (canonical, no custom config)
- **Vetting:** `go vet ./...`
- **Run both:** `make lint`

### Naming

- Exported identifiers: `PascalCase` (`NewSimulator`, `PhysicsBackend`)
- Unexported identifiers: `camelCase` (`maxTrailLen`, `stepVerlet`)
- Acronyms stay uppercase: `FFI`, `GPU`, `GR`, `AU`
- Test helpers: unexported, in `_test.go` files (`assertRelativeError`)

### Package Layout

```
cmd/          Entry points (one main package per binary)
internal/     Private packages, organized by responsibility
pkg/          Public packages (constants only, currently)
crates/       Rust crates (physics_core, render_core)
docs/         Documentation
```

Key separations:
- `internal/physics/` owns simulation state and integration
- `internal/render/` owns canvas drawing
- `internal/ui/` owns Fyne widgets and layout
- `internal/launch/` owns orbital mechanics for launch planning
- `internal/ffi/` owns Go-Rust FFI bindings
- `pkg/constants/` owns physical constants shared across packages

### Error Handling

**CLI layer** (`cmd/`): print to stderr and exit.
```go
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

**Internal packages**: return errors to the caller where applicable. Avoid `os.Exit` in library code.

**Future direction:** use `fmt.Errorf("context: %w", err)` for error wrapping when richer error context is needed.

### Concurrency

- `sync.RWMutex` protects simulator state (`Simulator.mu`)
- Snapshot pattern for renderer reads: `GetPlanetSnapshot()` returns a copy under read lock
- Never hold a lock across FFI calls

### Build Tags

Place `//go:build` on the first line of the file. Every feature-gated file must have a `_noop.go` counterpart:

| Tag | Enables | Example files |
|-----|---------|--------------|
| `nogui` | Headless build (no Fyne) | `gui_enabled.go` / `gui_noop.go` |
| `rust_physics` | Rust physics backend | `backend_init_rust.go` / `backend_init_go.go` |
| `rust_render` | GPU rendering via wgpu | `gpu_renderer.go` / `gpu_renderer_noop.go` |

## Rust

### Formatting and Linting

- **Formatter:** `rustfmt` (default settings)
- **Linter:** `cargo clippy`
- **Edition:** 2021

### FFI Functions

All exported C functions follow this pattern:

```rust
#[no_mangle]
pub unsafe extern "C" fn physics_create(
    n_bodies: u32,
    sun_mass: f64,
    // ...
) -> *mut Simulation {
    // Null-pointer guard on inputs
    // Box::into_raw(Box::new(...)) for opaque handle
}
```

- Always guard null pointers on input
- Use `Box::into_raw` to hand ownership to Go, `Box::from_raw` to reclaim in `_free()`
- Keep unsafe scope minimal: only in `ffi.rs`

### Crate Layout

```
crates/<name>/
  Cargo.toml          # [lib] crate-type = ["cdylib"]
  include/<name>.h    # C ABI header
  src/
    lib.rs            # Crate root
    ffi.rs            # FFI boundary (all unsafe here)
    ...               # Pure Rust modules
```

## Cross-Language Conventions

### C ABI Boundary

- **Types:** primitives only (`f64`, `u32`, `u8`, raw pointers). No Rust-specific types cross FFI.
- **Handles:** opaque pointers (`*mut Simulation`). Go sees `*C.PhysicsSim`.
- **Buffers:** caller (Go) allocates output buffers, passes pointer and length. Rust writes into them.
- **Strings:** not passed across FFI currently. If needed, use null-terminated `*const c_char`.

### Memory Ownership

- Simulation handles: Rust allocates, Go holds, Go calls `_free()` to deallocate
- State buffers: Go allocates flat `[]float64`, passes to Rust for read/write
- No garbage collection across FFI boundary

### Keeping FFI in Sync

When modifying the Go-Rust boundary, update all three layers together:
1. C header (`include/<name>.h`)
2. Rust FFI (`src/ffi.rs`)
3. Go CGo bindings (`internal/ffi/<name>_rust.go`)

## Comments and Documentation

- Doc comments on all exported Go types and functions
- Rust: `///` doc comments on public items
- Physics code: reference formulas or papers in comments (e.g., "1PN post-Newtonian correction, see Weinberg 1972")
- Avoid redundant comments that restate the code

## Testing

- Go tests: same-package `_test.go` files
- Custom assertion helpers in `test_helpers_test.go`: `assertRelativeError`, `assertFloat64Near`
- Tolerance-based float comparison for physics (never use `==` on floats)
- Benchmarks in `_benchmark_test.go` with `testing.B`
- Rust tests: `#[cfg(test)] mod tests` in each module
- Validation scenarios: `internal/validation/` for physics acceptance tests
