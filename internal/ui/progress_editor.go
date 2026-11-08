// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"math"
)

type editableProgress struct {
	widget.BaseWidget
	bar       *miniProgress
	OnChanged func(int)
	busy      bool
}

func newEditableProgress() *editableProgress {
	p := &editableProgress{bar: newProgress()}
	p.ExtendBaseWidget(p)
	return p
}
func (p *editableProgress) SetValue(v float64) { p.bar.SetValue(v) }
func (p *editableProgress) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.bar)
}
func (p *editableProgress) Cursor() desktop.Cursor {
	if p.busy {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}
func (p *editableProgress) Tapped(e *fyne.PointEvent) {
	if p.busy || p.OnChanged == nil {
		return
	}
	w := p.Size().Width - 40
	if w <= 0 {
		return
	}
	value := int(math.Round(float64(max(0, min(1, e.Position.X/w)))*10)) * 10
	if value == int(math.Round(p.bar.value*100)) {
		return
	}
	p.OnChanged(value)
}
func (u *UI) setIssueProgress(id int, p *editableProgress, value int) {
	if p.busy {
		return
	}
	old := p.bar.value
	p.busy = true
	p.SetValue(float64(value) / 100)
	c := u.client
	go func() {
		e := c.Update(context.Background(), id, map[string]any{"done_ratio": value})
		fyne.Do(func() {
			if u.client != c {
				return
			}
			p.busy = false
			if e != nil {
				p.SetValue(old)
				dialog.ShowError(e, u.win)
				return
			}
			for i := range u.issues {
				if u.issues[i].ID == id {
					u.issues[i].Done = value
				}
			}
			if u.list != nil {
				u.list.Refresh()
			}
			if u.selected == id {
				u.openIssue(id)
			}
		})
	}()
}
