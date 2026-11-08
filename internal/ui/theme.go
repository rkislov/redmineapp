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
		return palette{background: color.NRGBA{19, 25, 34, 255}, surface: color.NRGBA{38, 48, 62, 255}, raised: color.NRGBA{56, 69, 85, 170}, border: color.NRGBA{184, 206, 228, 44}, foreground: color.NRGBA{232, 238, 246, 255}, muted: color.NRGBA{165, 181, 197, 255}, accent: color.NRGBA{144, 172, 194, 255}, selected: color.NRGBA{55, 77, 98, 255}, ring: color.NRGBA{144, 172, 194, 255}, track: color.NRGBA{124, 151, 177, 60}, shadow: color.NRGBA{0, 0, 0, 65}}
	}
	return palette{background: color.NRGBA{231, 237, 243, 255}, surface: color.NRGBA{251, 253, 255, 255}, raised: color.NRGBA{241, 247, 253, 155}, border: color.NRGBA{255, 255, 255, 210}, foreground: color.NRGBA{36, 51, 67, 255}, muted: color.NRGBA{98, 119, 140, 255}, accent: color.NRGBA{76, 108, 134, 255}, selected: color.NRGBA{216, 233, 247, 255}, ring: color.NRGBA{127, 161, 189, 255}, track: color.NRGBA{144, 166, 187, 60}, shadow: color.NRGBA{33, 42, 48, 17}}
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
	case theme.ColorNameInputBackground:
		return alpha(p.surface, 165)
	case theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		return alpha(p.surface, 248)
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
		return 18
	case theme.SizeNameCardRadius, theme.SizeNameDialogRadius:
		return 24
	case theme.SizeNameSplitThickness:
		return 2
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 8
	}
	return theme.DefaultTheme().Size(n)
}

var starIcon = theme.NewThemedResource(fyne.NewStaticResource("star.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#000" d="m12 2 3 6.2 7 1-5 4.9 1.2 6.9L12 17.7 5.8 21 7 14.1 2 9.2l7-1z"/></svg>`)))
