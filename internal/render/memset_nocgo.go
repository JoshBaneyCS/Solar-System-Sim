//go:build !cgo

package render

// clearPixels zeros a byte slice using a Go loop (fallback when CGO is disabled).
func clearPixels(pix []byte) {
	for i := range pix {
		pix[i] = 0
	}
}
