//go:build !nogui

package main

import "solar-system-sim/internal/ui"

func runGUI() {
	app := ui.NewApp()
	app.Run()
}
