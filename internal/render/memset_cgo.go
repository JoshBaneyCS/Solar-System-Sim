//go:build cgo

package render

/*
#include <string.h>
*/
import "C"
import "unsafe"

// clearPixels zeros a byte slice using C memset, which leverages SIMD/rep stosb
// instructions for 10-50x faster clearing than a Go byte loop on large buffers.
func clearPixels(pix []byte) {
	if len(pix) == 0 {
		return
	}
	C.memset(unsafe.Pointer(&pix[0]), 0, C.size_t(len(pix)))
}
