# Security

## Scope

This is a desktop physics simulation application and CLI tool. It does not run network services, handle user authentication, or process untrusted input from the network.

## FFI Safety

The Go-Rust boundary uses unsafe code on both sides:

- **Rust side** (`crates/*/src/ffi.rs`): All `#[no_mangle] extern "C"` functions are `unsafe`. Null-pointer guards are required on all exported functions before dereferencing.
- **Go side** (`internal/ffi/*_rust.go`): CGo calls use `unsafe.Pointer` for type conversion. Keep the scope of unsafe operations minimal.
- **Opaque handle pattern**: Simulation state is passed as an opaque `*C.PhysicsSim` pointer. Go cannot read or modify Rust memory directly, which prevents accidental corruption.

When modifying FFI code, verify:
1. All pointer arguments are null-checked before use
2. Buffer sizes match between Go allocation and Rust read/write
3. `_free()` is called exactly once per handle (no double-free, no leak)

## Build Integrity

- **Go dependencies**: Verified by `go.sum` checksums. Run `go mod verify` to check.
- **Rust dependencies**: Locked by `Cargo.lock`. Run `cargo audit` periodically to check for known vulnerabilities.
- **Build from source**: Preferred method. See [INSTALL.md](INSTALL.md) for instructions.

## Asset Handling

- Texture and mesh files are loaded from the local `assets/` directory only
- No assets are fetched from the network at runtime
- Asset integrity can be verified with `solar-sim assets verify` (see [CLI.md](CLI.md))
- The asset validator checks file existence, format, and minimum sizes

## Dependency Auditing

Periodically check for known vulnerabilities:

```bash
# Go dependencies
go list -m all
go mod verify

# Rust dependencies (requires cargo-audit)
cd crates/physics_core && cargo audit
cd crates/render_core && cargo audit
```

## Reporting Vulnerabilities

If you discover a security issue, please open a GitHub issue with the `security` label. For sensitive disclosures, contact the maintainer directly before public disclosure.
