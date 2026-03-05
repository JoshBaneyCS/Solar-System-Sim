## Description
<!-- What does this PR do? Why is this change needed? -->

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Refactoring (no behavior change)
- [ ] Documentation
- [ ] Build / CI

## Checklist
- [ ] `make test` passes
- [ ] `make vet` passes
- [ ] Code formatted with `gofmt`
- [ ] Rust changes: `cargo clippy` clean and `cargo test` passes
- [ ] FFI changes: C header, Go bindings, and Rust `ffi.rs` updated together
- [ ] Build tag variants tested if applicable (`nogui`, `rust_physics`, `rust_render`)
- [ ] Documentation updated in `docs/` if behavior changed
- [ ] No hardcoded physical constants introduced (use `pkg/constants/` or `internal/launch/constants.go`)

## Testing
<!-- How was this tested? Which make targets were run? -->

## Related Issues
<!-- Link any related issues -->
