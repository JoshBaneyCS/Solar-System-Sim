package ui

import (
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showAboutWindow() {
	w := a.fyneApp.NewWindow("About Solar System Simulator")

	// Show app logo if available
	var logoContainer *fyne.Container
	if icon := a.fyneApp.Icon(); icon != nil {
		logo := canvas.NewImageFromResource(icon)
		logo.SetMinSize(fyne.NewSize(128, 128))
		logo.FillMode = canvas.ImageFillContain
		logoContainer = container.New(layout.NewCenterLayout(), logo)
	}

	title := widget.NewLabel("Solar System Simulator")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	version := widget.NewLabel("Version 0.1.2")
	version.Alignment = fyne.TextAlignCenter

	author := widget.NewLabel("Author: Joshua Baney")
	author.Alignment = fyne.TextAlignCenter

	repoURL, _ := url.Parse("https://github.com/JoshBaneyCS/Solar-System-Sim")
	repoLink := widget.NewHyperlink("GitHub Repository", repoURL)
	repoLink.Alignment = fyne.TextAlignCenter

	donateURL, _ := url.Parse("https://www.paypal.com/donate/?business=HWM2DENMWG4K2&no_recurring=0&item_name=TO+continue+funding+ongoing+development+to+Solar+System+Simulator+-+An+open+source+Physics+application&currency_code=USD")
	donateLink := widget.NewHyperlink("Sponsor / Donate", donateURL)
	donateLink.Alignment = fyne.TextAlignCenter

	// System information
	ri := a.runtimeInfo
	sysText := fmt.Sprintf(
		"System Information:\n\n"+
			"Platform: %s/%s\n"+
			"CPU Cores: %d\n"+
			"Go Version: %s\n"+
			"GPU Backend: %s",
		ri.OS, ri.Arch, ri.NumCPU, ri.GoVersion, ri.GPUBackend())

	if ri.GPUDevice != "" {
		sysText += fmt.Sprintf("\nGPU: %s", ri.GPUDevice)
		sysText += fmt.Sprintf("\nGPU Vendor: %s", ri.GPUVendor)
		sysText += fmt.Sprintf("\nGPU Type: %s", ri.GPUDeviceType)
		if ri.GPUTier != "" {
			sysText += fmt.Sprintf("\nPerformance Tier: %s", ri.GPUTier)
		}
		if ri.GPUMaxTexture > 0 {
			sysText += fmt.Sprintf("\nMax Texture Size: %d", ri.GPUMaxTexture)
		}
	}

	if ri.IsAppleSilicon {
		sysText += "\nApple Silicon: Yes (Metal supported)"
	}

	sysInfo := widget.NewLabel(sysText)
	sysInfo.Wrapping = fyne.TextWrapWord

	credits := widget.NewLabel(
		"Credits & Acknowledgements:\n\n" +
			"Fyne - Cross-platform GUI toolkit for Go\n" +
			"wgpu - WebGPU implementation for GPU rendering\n" +
			"NASA/JPL - Planetary ephemeris data\n" +
			"Go - Programming language\n" +
			"Rust - GPU acceleration backends\n\n" +
			"License: MIT")
	credits.Wrapping = fyne.TextWrapWord

	contentItems := []fyne.CanvasObject{}
	if logoContainer != nil {
		contentItems = append(contentItems, logoContainer)
	}
	contentItems = append(contentItems,
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
	content := container.NewVBox(contentItems...)

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(380, 450))

	w.SetContent(scroll)
	w.Resize(fyne.NewSize(400, 550))
	w.SetFixedSize(true)
	w.Show()
}
