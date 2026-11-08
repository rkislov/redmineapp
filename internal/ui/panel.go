// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"math"
)

// floatingPanel renders an elevated, theme-aware surface.
type floatingPanel struct {
	widget.BaseWidget
	content                        fyne.CanvasObject
	selected, hovered, interactive bool
	pointer                        float32
}

func newPanel(content fyne.CanvasObject) *floatingPanel {
	p := &floatingPanel{content: content, pointer: .35}
	p.ExtendBaseWidget(p)
	return p
}
func (p *floatingPanel) MouseIn(e *desktop.MouseEvent) {
	if p.interactive {
		p.hovered = true
		if p.Size().Width > 0 {
			p.pointer = e.Position.X / p.Size().Width
		}
		p.Refresh()
	}
}
func (p *floatingPanel) MouseMoved(e *desktop.MouseEvent) {
	if p.interactive && p.Size().Width > 0 {
		v := e.Position.X / p.Size().Width
		if math.Abs(float64(v-p.pointer)) > .035 {
			p.pointer = v
			p.Refresh()
		}
	}
}
func (p *floatingPanel) MouseOut() {
	if p.interactive {
		p.hovered = false
		p.Refresh()
	}
}
func (p *floatingPanel) CreateRenderer() fyne.WidgetRenderer {
	r := &panelRenderer{p: p, surface: newGlassSurface()}
	r.Refresh()
	return r
}

type panelRenderer struct {
	p       *floatingPanel
	surface *glassSurface
}

func (r *panelRenderer) MinSize() fyne.Size { return r.p.content.MinSize().Add(fyne.NewSize(40, 36)) }
func (r *panelRenderer) Layout(s fyne.Size) {
	lift := float32(0)
	if r.p.hovered {
		lift = 2
	}
	r.surface.Move(fyne.NewPos(6, 6-lift))
	r.surface.Resize(fyne.NewSize(max(0, s.Width-12), max(0, s.Height-12)))
	r.p.content.Move(fyne.NewPos(20, 18-lift))
	r.p.content.Resize(fyne.NewSize(max(0, s.Width-40), max(0, s.Height-36)))
}
func (r *panelRenderer) Refresh() {
	r.surface.selected = r.p.selected
	r.surface.hovered = r.p.hovered
	r.surface.pointer = r.p.pointer
	r.Layout(r.p.Size())
	r.surface.Refresh()
}
func (r *panelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.surface, r.p.content}
}
func (r *panelRenderer) Destroy() {}
