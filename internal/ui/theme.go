package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// SpaceTheme is a custom dark theme for the solar system simulator.
type SpaceTheme struct{}

var _ fyne.Theme = (*SpaceTheme)(nil)

func (t *SpaceTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{10, 10, 26, 255} // #0A0A1A
	case theme.ColorNameButton:
		return color.RGBA{26, 26, 46, 255} // #1A1A2E
	case theme.ColorNameDisabledButton:
		return color.RGBA{20, 20, 35, 255}
	case theme.ColorNamePrimary:
		return color.RGBA{0, 180, 216, 255} // #00B4D8 cyan accent
	case theme.ColorNameFocus:
		return color.RGBA{0, 180, 216, 128}
	case theme.ColorNameHover:
		return color.RGBA{30, 30, 55, 255}
	case theme.ColorNameInputBackground:
		return color.RGBA{20, 20, 38, 255}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{120, 120, 140, 255}
	case theme.ColorNameScrollBar:
		return color.RGBA{60, 60, 80, 255}
	case theme.ColorNameShadow:
		return color.RGBA{0, 0, 0, 100}
	case theme.ColorNameForeground:
		return color.RGBA{220, 220, 230, 255}
	case theme.ColorNameSeparator:
		return color.RGBA{40, 40, 60, 255}
	}
	// Fall back to dark theme for anything not overridden
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (t *SpaceTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *SpaceTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *SpaceTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 4
	case theme.SizeNameText:
		return 13
	case theme.SizeNameSeparatorThickness:
		return 1
	}
	return theme.DefaultTheme().Size(name)
}
