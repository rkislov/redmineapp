// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type miniProgress struct {
	widget.BaseWidget
	value float64
}

func newProgress() *miniProgress           { p := &miniProgress{}; p.ExtendBaseWidget(p); return p }
func (p *miniProgress) SetValue(v float64) { p.value = max(0, min(1, v)); p.Refresh() }
func (p *miniProgress) CreateRenderer() fyne.WidgetRenderer {
	pcolors := currentColors()
	r := &progressRenderer{p: p, track: canvas.NewRectangle(pcolors.track), fill: canvas.NewRectangle(pcolors.accent), text: canvas.NewText("0 %", pcolors.muted)}
	r.track.CornerRadius = 3
	r.fill.CornerRadius = 3
	r.text.TextSize = 11
	return r
}

type progressRenderer struct {
	p           *miniProgress
	track, fill *canvas.Rectangle
	text        *canvas.Text
}

func (r *progressRenderer) Layout(s fyne.Size) {
	w := max(float32(0), s.Width-40)
	r.track.Move(fyne.NewPos(0, (s.Height-6)/2))
	r.track.Resize(fyne.NewSize(w, 6))
	r.fill.Move(r.track.Position())
	r.fill.Resize(fyne.NewSize(w*float32(r.p.value), 6))
	r.text.Move(fyne.NewPos(w+5, 0))
	r.text.Resize(fyne.NewSize(35, s.Height))
}
func (r *progressRenderer) MinSize() fyne.Size { return fyne.NewSize(140, 18) }
func (r *progressRenderer) Refresh() {
	pcolors := currentColors()
	r.track.FillColor = pcolors.track
	r.fill.FillColor = pcolors.accent
	r.text.Color = pcolors.muted
	r.text.Text = fmt.Sprintf("%d %%", int(r.p.value*100))
	r.Layout(r.p.Size())
	canvas.Refresh(r.p)
}
func (r *progressRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.fill, r.text}
}
func (r *progressRenderer) Destroy() {}
