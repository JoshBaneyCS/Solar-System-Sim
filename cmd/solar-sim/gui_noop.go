//go:build nogui

package main

import (
	"fmt"
	"os"
)

func runGUI() {
	fmt.Fprintln(os.Stderr, "Error: GUI not available in this build (built with -tags nogui)")
	os.Exit(1)
}
