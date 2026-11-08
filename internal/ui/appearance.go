// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
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
	for i := range []string{"light", "auto", "dark"} {
		t.buttons[i].Importance = widget.LowImportance
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
	track := newGlassSurface()
	track.radius = 22
	knob := newGlassSurface()
	knob.radius = 17
	knob.selected = true
	content := container.NewPadded(container.NewGridWithColumns(3, objects...))
	return &appearanceRenderer{toggle: t, track: track, knob: knob, content: content}
}

type appearanceRenderer struct {
	toggle      *appearanceToggle
	track, knob *glassSurface
	content     *fyne.Container
}

func (r *appearanceRenderer) Layout(s fyne.Size) {
	r.track.Resize(s)
	r.content.Resize(s)
	index := 0
	switch r.toggle.Selected {
	case "auto":
		index = 1
	case "dark":
		index = 2
	}
	pad := float32(8)
	gap := float32(8)
	w := max(0, (s.Width-pad*2-gap*2)/3)
	r.knob.Move(fyne.NewPos(pad+float32(index)*(w+gap), pad))
	r.knob.Resize(fyne.NewSize(w, max(0, s.Height-pad*2)))
}
func (r *appearanceRenderer) MinSize() fyne.Size { return r.content.MinSize() }
func (r *appearanceRenderer) Refresh() {
	r.Layout(r.toggle.Size())
	r.track.Refresh()
	r.knob.Refresh()
	r.content.Refresh()
}
func (r *appearanceRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.knob, r.content}
}
func (r *appearanceRenderer) Destroy() {}

// Original vector symbols; themed resources follow foreground colors in both modes.
var dayIcon = theme.NewThemedResource(fyne.NewStaticResource("appearance-day.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#000" d="M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8ZM11 1h2v4h-2zM11 19h2v4h-2zM1 11h4v2H1zM19 11h4v2h-4zM3.5 4.9l1.4-1.4 2.8 2.8-1.4 1.4zM16.3 17.7l1.4-1.4 2.8 2.8-1.4 1.4zM3.5 19.1l2.8-2.8 1.4 1.4-2.8 2.8zM16.3 6.3l2.8-2.8 1.4 1.4-2.8 2.8z"/></svg>`)))
var autoIcon = theme.NewThemedResource(fyne.NewStaticResource("appearance-auto.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#000" fill-rule="evenodd" d="M4 3h16a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2h-7v3h4v2H7v-2h4v-3H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Zm0 2v10h16V5H4Z"/></svg>`)))
var nightIcon = theme.NewThemedResource(fyne.NewStaticResource("appearance-night.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#000" d="M20.6 14.1A9 9 0 0 1 9.9 3.4a9 9 0 1 0 10.7 10.7Z"/></svg>`)))
