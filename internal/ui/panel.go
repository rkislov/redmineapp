// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// floatingPanel renders an elevated, theme-aware surface.
type floatingPanel struct {
	widget.BaseWidget
	content                        fyne.CanvasObject
	selected, hovered, interactive bool
}

func newPanel(content fyne.CanvasObject) *floatingPanel {
	p := &floatingPanel{content: content}
	p.ExtendBaseWidget(p)
	return p
}
func (p *floatingPanel) MouseIn(*desktop.MouseEvent) {
	if p.interactive {
		p.hovered = true
		p.Refresh()
	}
}
func (p *floatingPanel) MouseMoved(*desktop.MouseEvent) {}
func (p *floatingPanel) MouseOut() {
	if p.interactive {
		p.hovered = false
		p.Refresh()
	}
}
func (p *floatingPanel) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(currentColors().surface)
	bg.CornerRadius = 16
	r := &panelRenderer{p: p, bg: bg}
	r.Refresh()
	return r
}

type panelRenderer struct {
	p  *floatingPanel
	bg *canvas.Rectangle
}

func (r *panelRenderer) MinSize() fyne.Size { return r.p.content.MinSize().Add(fyne.NewSize(36, 32)) }
func (r *panelRenderer) Layout(s fyne.Size) {
	lift := float32(0)
	if r.p.hovered {
		lift = 2
	}
	r.bg.Move(fyne.NewPos(6, 6-lift))
	r.bg.Resize(fyne.NewSize(max(0, s.Width-12), max(0, s.Height-12)))
	r.p.content.Move(fyne.NewPos(18, 16-lift))
	r.p.content.Resize(fyne.NewSize(max(0, s.Width-36), max(0, s.Height-32)))
}
func (r *panelRenderer) Refresh() {
	p := currentColors()
	r.bg.FillColor = p.surface
	r.bg.StrokeWidth = 1
	r.bg.StrokeColor = p.border
	r.bg.Shadow = canvas.Shadow{Color: p.shadow, BlurRadius: 12, Offset: fyne.NewPos(0, 4)}
	if r.p.selected {
		r.bg.FillColor = p.selected
		r.bg.StrokeColor = p.ring
		r.bg.StrokeWidth = 1.5
	}
	if r.p.hovered {
		r.bg.Shadow.BlurRadius = 18
		r.bg.Shadow.Offset = fyne.NewPos(0, 7)
		r.bg.Shadow.Color = p.shadow
	}
	r.Layout(r.p.Size())
	r.bg.Refresh()
	canvas.Refresh(r.p)
}
func (r *panelRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.p.content} }
func (r *panelRenderer) Destroy()                     {}
