// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

type appTheme struct {
	Mode string
	Dark bool
}

func (t appTheme) isDark(v fyne.ThemeVariant) bool {
	if t.Mode == "auto" {
		return v == theme.VariantDark
	}
	return t.Mode == "dark" || t.Mode == "" && t.Dark
}

type palette struct{ background, surface, raised, border, foreground, muted, accent, selected, ring, track, shadow color.NRGBA }

func colors(dark bool) palette {
	if dark {
		return palette{background: color.NRGBA{24, 26, 29, 255}, surface: color.NRGBA{33, 36, 40, 255}, raised: color.NRGBA{43, 47, 52, 255}, border: color.NRGBA{58, 63, 69, 255}, foreground: color.NRGBA{226, 229, 232, 255}, muted: color.NRGBA{157, 165, 174, 255}, accent: color.NRGBA{126, 155, 170, 255}, selected: color.NRGBA{40, 49, 56, 255}, ring: color.NRGBA{126, 155, 170, 255}, track: color.NRGBA{56, 62, 69, 255}, shadow: color.NRGBA{0, 0, 0, 65}}
	}
	return palette{background: color.NRGBA{242, 243, 244, 255}, surface: color.NRGBA{253, 253, 252, 255}, raised: color.NRGBA{233, 236, 238, 255}, border: color.NRGBA{221, 225, 228, 255}, foreground: color.NRGBA{48, 54, 60, 255}, muted: color.NRGBA{105, 114, 122, 255}, accent: color.NRGBA{80, 107, 122, 255}, selected: color.NRGBA{237, 242, 244, 255}, ring: color.NRGBA{125, 150, 164, 255}, track: color.NRGBA{223, 228, 231, 255}, shadow: color.NRGBA{33, 42, 48, 17}}
}
func currentColors() palette {
	if a := fyne.CurrentApp(); a != nil {
		if t, ok := a.Settings().Theme().(appTheme); ok {
			return colors(t.isDark(a.Settings().ThemeVariant()))
		}
	}
	return colors(false)
}
func (t appTheme) Color(n fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	dark := t.isDark(variant)
	p := colors(dark)
	switch n {
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return p.accent
	case theme.ColorNameBackground:
		return p.background
	case theme.ColorNameButton, theme.ColorNameHeaderBackground:
		return p.raised
	case theme.ColorNameForeground:
		return p.foreground
	case theme.ColorNameForegroundOnPrimary:
		if dark {
			return color.NRGBA{24, 30, 34, 255}
		}
		return color.NRGBA{250, 251, 252, 255}
	case theme.ColorNameShadow:
		return p.shadow
	case theme.ColorNameHover:
		if dark {
			return color.NRGBA{220, 230, 240, 12}
		}
		return color.NRGBA{40, 60, 75, 8}
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return p.border
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
		return p.muted
	case theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground, theme.ColorNameInputBackground:
		return p.surface
	case theme.ColorNameDisabledButton:
		return p.raised
	case theme.ColorNameScrollBar:
		return p.muted
	case theme.ColorNameSelection:
		return color.Transparent
	}
	v := theme.VariantLight
	if dark {
		v = theme.VariantDark
	}
	return theme.DefaultTheme().Color(n, v)
}
func (appTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (appTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }
func (appTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius, theme.SizeNameSelectionRadius:
		return 10
	case theme.SizeNameCardRadius, theme.SizeNameDialogRadius:
		return 16
	case theme.SizeNameSplitThickness:
		return 3
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 8
	}
	return theme.DefaultTheme().Size(n)
}

var starIcon = theme.NewThemedResource(fyne.NewStaticResource("star.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#000" d="m12 2 3 6.2 7 1-5 4.9 1.2 6.9L12 17.7 5.8 21 7 14.1 2 9.2l7-1z"/></svg>`)))
