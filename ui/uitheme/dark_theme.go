// Package uitheme defines the professional dark theme for the application.
// Named "uitheme" to avoid shadowing "fyne.io/fyne/v2/theme".
package uitheme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Palette — Navy Dark + Indigo accent (inspired by Linear / VS Code Dark+).
var (
	ColorBackground = color.NRGBA{R: 0x0F, G: 0x17, B: 0x2A, A: 0xFF} // #0F172A
	ColorSurface    = color.NRGBA{R: 0x1E, G: 0x29, B: 0x3B, A: 0xFF} // #1E293B
	ColorCard       = color.NRGBA{R: 0x27, G: 0x37, B: 0x4F, A: 0xFF} // #27374F
	ColorPrimary    = color.NRGBA{R: 0x63, G: 0x66, B: 0xF1, A: 0xFF} // #6366F1 indigo-500
	ColorForeground = color.NRGBA{R: 0xF1, G: 0xF5, B: 0xF9, A: 0xFF} // #F1F5F9 slate-100
	ColorMuted      = color.NRGBA{R: 0x94, G: 0xA3, B: 0xB8, A: 0xFF} // #94A3B8 slate-400
	ColorSeparator  = color.NRGBA{R: 0x33, G: 0x41, B: 0x55, A: 0xFF} // #334155 slate-700
	ColorError      = color.NRGBA{R: 0xEF, G: 0x44, B: 0x44, A: 0xFF} // #EF4444 red-500
	ColorSuccess    = color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF} // #22C55E green-500
	ColorWarning    = color.NRGBA{R: 0xF5, G: 0x9E, B: 0x0B, A: 0xFF} // #F59E0B amber-500
	ColorHover      = color.NRGBA{R: 0x63, G: 0x66, B: 0xF1, A: 0x28} // indigo 16% alpha
)

// DarkTheme implements fyne.Theme with a professional dark color scheme.
type DarkTheme struct{}

// Compile-time assertion.
var _ fyne.Theme = (*DarkTheme)(nil)

func (t *DarkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorBackground
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return ColorSurface
	case theme.ColorNameInputBackground:
		return ColorSurface
	case theme.ColorNameButton:
		return ColorCard
	case theme.ColorNamePrimary:
		return ColorPrimary
	case theme.ColorNameFocus:
		return ColorPrimary
	case theme.ColorNameHover:
		return ColorHover
	case theme.ColorNameForeground:
		return ColorForeground
	case theme.ColorNamePlaceHolder:
		return ColorMuted
	case theme.ColorNameSeparator:
		return ColorSeparator
	case theme.ColorNameInputBorder:
		return ColorSeparator
	case theme.ColorNameError:
		return ColorError
	case theme.ColorNameSuccess:
		return ColorSuccess
	case theme.ColorNameWarning:
		return ColorWarning
	case theme.ColorNameScrollBar:
		return ColorSeparator
	case theme.ColorNameHeaderBackground:
		return ColorSurface
	case theme.ColorNameSelection:
		return ColorHover
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x40}
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (t *DarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *DarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 6 // era 12 — reduzia botões para 2×12+icon = 56px de altura.
	case theme.SizeNameInlineIcon:
		return 20 // ícones de botão menores.
	case theme.SizeNameText:
		return 13
	case theme.SizeNameSubHeadingText:
		return 15
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameInputBorder:
		return 1.5
	case theme.SizeNameSeparatorThickness:
		return 1
	}
	return theme.DefaultTheme().Size(name)
}
