//go:build rust_physics

package physics

// initBackend creates and attaches the Rust physics backend.
func initBackend(s *Simulator) {
	s.Backend = NewRustBackend(s)
}
