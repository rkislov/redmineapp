// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strings"
)

type badge struct {
	widget.BaseWidget
	text *canvas.Text
	bg   *canvas.Rectangle
}

func newBadge() *badge {
	b := &badge{text: canvas.NewText("В работе", currentColors().foreground), bg: canvas.NewRectangle(color.Transparent)}
	b.text.TextSize = 12
	b.text.TextStyle.Bold = true
	b.bg.CornerRadius = 8
	b.ExtendBaseWidget(b)
	return b
}
func (b *badge) Set(s string) { b.text.Text = s; b.Refresh() }
func (b *badge) Refresh() {
	p := currentColors()
	name := strings.ToLower(b.text.Text)
	fg, bg := p.accent, p.selected
	dark := false
	if t, ok := fyne.CurrentApp().Settings().Theme().(appTheme); ok {
		dark = t.isDark(fyne.CurrentApp().Settings().ThemeVariant())
	}
	switch {
	case strings.Contains(name, "закры"), strings.Contains(name, "решен"), strings.Contains(name, "решён"), strings.Contains(name, "closed"), strings.Contains(name, "resolved"):
		fg = color.NRGBA{76, 109, 91, 255}
		bg = color.NRGBA{230, 237, 232, 255}
		if dark {
			fg = color.NRGBA{158, 188, 168, 255}
			bg = color.NRGBA{43, 58, 49, 255}
		}
	case strings.Contains(name, "отклон"), strings.Contains(name, "reject"):
		fg = color.NRGBA{143, 98, 98, 255}
		bg = color.NRGBA{242, 233, 232, 255}
		if dark {
			fg = color.NRGBA{207, 167, 165, 255}
			bg = color.NRGBA{64, 47, 47, 255}
		}
	case strings.Contains(name, "нов"), strings.Contains(name, "new"):
		fg = p.muted
		bg = p.raised
	}
	b.text.Color = fg
	b.bg.FillColor = bg
	b.text.Refresh()
	b.bg.Refresh()
	b.BaseWidget.Refresh()
}
func (b *badge) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(b.bg, container.NewPadded(b.text)))
}
