package ui

import (
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

	version := widget.NewLabel("Version 1.0")
	version.Alignment = fyne.TextAlignCenter

	author := widget.NewLabel("Author: Joshua Baney")
	author.Alignment = fyne.TextAlignCenter

	repoURL, _ := url.Parse("https://github.com/joshbaney/solar-system-simulator")
	repoLink := widget.NewHyperlink("GitHub Repository", repoURL)
	repoLink.Alignment = fyne.TextAlignCenter

	donateURL, _ := url.Parse("https://github.com/sponsors/joshbaney")
	donateLink := widget.NewHyperlink("Sponsor / Donate", donateURL)
	donateLink.Alignment = fyne.TextAlignCenter

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
		credits,
	)

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(380, 400))

	w.SetContent(scroll)
	w.Resize(fyne.NewSize(400, 500))
	w.SetFixedSize(true)
	w.Show()
}
