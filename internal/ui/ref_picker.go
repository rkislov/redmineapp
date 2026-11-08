// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/redmine"
	"strconv"
	"strings"
)

type refPicker struct {
	all, filtered []redmine.Ref
	selected      int
	search        *widget.Entry
	list          *widget.List
	count, empty  *widget.Label
	confirm       *widget.Button
	content       *fyne.Container
	choose        func(redmine.Ref)
}

func newRefPicker(refs []redmine.Ref, current int, choose func(redmine.Ref), cancel func()) *refPicker {
	p := &refPicker{all: append([]redmine.Ref(nil), refs...), selected: -1, choose: choose}
	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("Поиск по названию или #номеру")
	p.count = widget.NewLabel("")
	p.empty = widget.NewLabel("Ничего не найдено")
	p.empty.Alignment = fyne.TextAlignCenter
	p.confirm = widget.NewButton("Выбрать", p.accept)
	p.confirm.Importance = widget.HighImportance
	p.confirm.Disable()
	p.list = widget.NewList(func() int { return len(p.filtered) }, func() fyne.CanvasObject {
		title := widget.NewLabel("Статус")
		title.Truncation = fyne.TextTruncateEllipsis
		return container.NewBorder(nil, nil, nil, widget.NewLabel("#123"), title)
	}, func(i widget.ListItemID, o fyne.CanvasObject) {
		row := o.(*fyne.Container)
		row.Objects[0].(*widget.Label).SetText(p.filtered[i].Name)
		row.Objects[1].(*widget.Label).SetText(fmt.Sprintf("#%d", p.filtered[i].ID))
	})
	p.list.OnSelected = func(i widget.ListItemID) { p.selected = i; p.confirm.Enable() }
	p.search.OnChanged = func(string) { p.filter() }
	p.search.OnSubmitted = func(string) {
		if p.selected < 0 && len(p.filtered) == 1 {
			p.list.Select(0)
		}
		p.accept()
	}
	header := container.NewVBox(p.search, p.count)
	footer := container.NewHBox(layout.NewSpacer(), widget.NewButton("Отмена", cancel), p.confirm)
	p.content = container.NewBorder(header, footer, nil, nil, container.NewStack(container.NewThemeOverride(p.list, projectListTheme{}), container.NewCenter(p.empty)))
	p.filter()
	for i, v := range p.filtered {
		if v.ID == current {
			p.list.Select(i)
			break
		}
	}
	return p
}
func (p *refPicker) filter() {
	terms := strings.Fields(normalizeProjectText(p.search.Text))
	p.filtered = nil
	for _, v := range p.all {
		hay := normalizeProjectText(v.Name) + " " + strconv.Itoa(v.ID)
		match := true
		for _, term := range terms {
			if !strings.Contains(hay, strings.TrimPrefix(term, "#")) {
				match = false
				break
			}
		}
		if match {
			p.filtered = append(p.filtered, v)
		}
	}
	p.selected = -1
	p.list.UnselectAll()
	p.list.Refresh()
	p.list.ScrollToTop()
	p.confirm.Disable()
	p.count.SetText(fmt.Sprintf("Найдено: %d", len(p.filtered)))
	if len(p.filtered) == 0 {
		p.empty.Show()
		p.list.Hide()
	} else {
		p.empty.Hide()
		p.list.Show()
	}
}
func (p *refPicker) accept() {
	if p.selected >= 0 && p.selected < len(p.filtered) {
		p.choose(p.filtered[p.selected])
	}
}
func (u *UI) searchableRef(title string, refs []redmine.Ref, current int) (fyne.CanvasObject, func() int) {
	selected := current
	text := widget.NewLabel("Не выбрано")
	text.Truncation = fyne.TextTruncateEllipsis
	for _, v := range refs {
		if v.ID == current {
			text.SetText(v.Name)
			break
		}
	}
	button := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), func() {
		var d *dialog.CustomDialog
		p := newRefPicker(refs, selected, func(v redmine.Ref) { selected = v.ID; text.SetText(v.Name); d.Hide() }, func() { d.Hide() })
		d = dialog.NewCustomWithoutButtons(title, p.content, u.win)
		size := u.win.Canvas().Size()
		d.Resize(fyne.NewSize(min(480, max(280, size.Width-80)), min(400, max(280, size.Height-100))))
		d.Show()
		u.win.Canvas().Focus(p.search)
	})
	button.Alignment = widget.ButtonAlignTrailing
	// The empty button supplies keyboard and pointer interaction; the ellipsized
	// label above it keeps long names from expanding the edit form.
	field := container.NewStack(button, container.NewBorder(nil, nil, nil, container.NewGridWrap(fyne.NewSize(32, 32), layout.NewSpacer()), container.NewPadded(text)))
	return field, func() int { return selected }
}
func issueStatusChoices(v redmine.Issue, fallback []redmine.Ref) []redmine.Ref {
	refs := v.AllowedStatuses
	if refs == nil {
		refs = fallback
	}
	refs = append([]redmine.Ref(nil), refs...)
	for _, r := range refs {
		if r.ID == v.Status.ID {
			return refs
		}
	}
	if v.Status.ID > 0 {
		refs = append([]redmine.Ref{v.Status}, refs...)
	}
	return refs
}
