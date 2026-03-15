package ui

import (
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showAboutWindow() {
	w := a.fyneApp.NewWindow("About Solar System Simulator")

	title := widget.NewLabel("Solar System Simulator")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	version := widget.NewLabel("Version 1.1")
	version.Alignment = fyne.TextAlignCenter

	author := widget.NewLabel("Author: Joshua Baney")
	author.Alignment = fyne.TextAlignCenter

	repoURL, _ := url.Parse("https://github.com/joshbaney/solar-system-simulator")
	repoLink := widget.NewHyperlink("GitHub Repository", repoURL)
	repoLink.Alignment = fyne.TextAlignCenter

	donateURL, _ := url.Parse("https://github.com/sponsors/joshbaney")
	donateLink := widget.NewHyperlink("Sponsor / Donate", donateURL)
	donateLink.Alignment = fyne.TextAlignCenter

	// System information
	ri := a.runtimeInfo
	sysInfo := widget.NewLabel(fmt.Sprintf(
		"System Information:\n\n"+
			"Platform: %s/%s\n"+
			"CPU Cores: %d\n"+
			"Go Version: %s\n"+
			"GPU Backend: %s",
		ri.OS, ri.Arch, ri.NumCPU, ri.GoVersion, ri.GPUBackend()))
	sysInfo.Wrapping = fyne.TextWrapWord

	if ri.IsAppleSilicon {
		sysInfo.SetText(sysInfo.Text + "\nApple Silicon: Yes (Metal supported)")
	}

	credits := widget.NewLabel(
		"Credits & Acknowledgements:\n\n" +
			"Fyne - Cross-platform GUI toolkit for Go\n" +
			"wgpu - WebGPU implementation for GPU rendering\n" +
			"NASA/JPL - Planetary ephemeris data\n" +
			"Go - Programming language\n" +
			"Rust - GPU acceleration backends\n\n" +
			"License: MIT")
	credits.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		title,
		version,
		widget.NewSeparator(),
		author,
		repoLink,
		donateLink,
		widget.NewSeparator(),
		sysInfo,
		widget.NewSeparator(),
		credits,
	)

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(380, 450))

	w.SetContent(scroll)
	w.Resize(fyne.NewSize(400, 550))
	w.SetFixedSize(true)
	w.Show()
}
