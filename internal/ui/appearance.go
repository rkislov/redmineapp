// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// appearanceToggle is a three-position switch in the fixed Day / Auto / Night order.
type appearanceToggle struct {
	widget.BaseWidget
	Selected  string
	OnChanged func(string)
	buttons   []*widget.Button
}

func newAppearanceToggle(mode string, onChanged func(string)) *appearanceToggle {
	t := &appearanceToggle{Selected: appearanceMode(mode), OnChanged: onChanged}
	for _, option := range []struct {
		mode string
		icon fyne.Resource
	}{{"light", dayIcon}, {"auto", autoIcon}, {"dark", nightIcon}} {
		option := option
		b := widget.NewButtonWithIcon("", option.icon, func() { t.SetSelected(option.mode) })
		t.buttons = append(t.buttons, b)
	}
	t.ExtendBaseWidget(t)
	t.updateButtons()
	return t
}
func appearanceMode(mode string) string {
	if mode == "light" || mode == "dark" {
		return mode
	}
	return "auto"
}
func (t *appearanceToggle) updateButtons() {
	for i, mode := range []string{"light", "auto", "dark"} {
		t.buttons[i].Importance = widget.LowImportance
		if t.Selected == mode {
			t.buttons[i].Importance = widget.HighImportance
		}
		t.buttons[i].Refresh()
	}
}
func (t *appearanceToggle) SetSelected(mode string) {
	mode = appearanceMode(mode)
	if mode == t.Selected {
		return
	}
	t.Selected = mode
	t.updateButtons()
	t.Refresh()
	if t.OnChanged != nil {
		t.OnChanged(mode)
	}
}
func (t *appearanceToggle) CreateRenderer() fyne.WidgetRenderer {
	objects := []fyne.CanvasObject{}
	for _, b := range t.buttons {
		objects = append(objects, b)
	}
	bg := canvas.NewRectangle(currentColors().raised)
	bg.CornerRadius = 13
	content := container.NewPadded(container.NewGridWithColumns(3, objects...))
	return &appearanceRenderer{toggle: t, bg: bg, content: content}
}

type appearanceRenderer struct {
	toggle  *appearanceToggle
	bg      *canvas.Rectangle
	content *fyne.Container
}

func (r *appearanceRenderer) Layout(s fyne.Size) { r.bg.Resize(s); r.content.Resize(s) }
func (r *appearanceRenderer) MinSize() fyne.Size { return r.content.MinSize() }
func (r *appearanceRenderer) Refresh() {
	r.bg.FillColor = currentColors().raised
	r.bg.Refresh()
	r.content.Refresh()
	canvas.Refresh(r.toggle)
}
func (r *appearanceRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.content}
}
func (r *appearanceRenderer) Destroy() {}

// Original vector symbols; themed resources follow foreground colors in both modes.
var dayIcon = theme.NewThemedResource(fyne.NewStaticResource("appearance-day.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><g fill="none" stroke="#000" stroke-width="1.8" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></g></svg>`)))
var autoIcon = theme.NewThemedResource(fyne.NewStaticResource("appearance-auto.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><g fill="none" stroke="#000" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8m-4-4v4"/></g></svg>`)))
var nightIcon = theme.NewThemedResource(fyne.NewStaticResource("appearance-night.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="none" stroke="#000" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" d="M20.6 14.1A9 9 0 0 1 9.9 3.4a9 9 0 1 0 10.7 10.7Z"/></svg>`)))
