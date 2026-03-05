//go:build !rust_physics

package physics

// initBackend is a no-op when Rust physics is not enabled.
func initBackend(_ *Simulator) {}
