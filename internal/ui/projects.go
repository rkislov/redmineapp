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
	"image/color"
	"sort"
	"strconv"
	"strings"
)

type projectPicker struct {
	total                         int
	activity                      *widget.Activity
	status                        *fyne.Container
	progress                      *widget.Label
	loading                       bool
	loadError                     error
	all, filtered, visible        []redmine.Ref
	page, pageSize, selected      int
	search                        *widget.Entry
	sortBy, sizeSelect            *widget.Select
	list                          *widget.List
	count, selection, empty       *widget.Label
	first, prev, next, last, open *widget.Button
	content                       *fyne.Container
	choose                        func(redmine.Ref)
}

func normalizeProjectText(s string) string { return strings.ReplaceAll(strings.ToLower(s), "ё", "е") }
func newProjectPicker(projects []redmine.Ref, choose func(redmine.Ref), cancel func()) *projectPicker {
	p := &projectPicker{all: append([]redmine.Ref(nil), projects...), pageSize: 20, selected: -1, choose: choose}
	p.search = widget.NewEntry()
	p.search.SetPlaceHolder("Название проекта или #номер")
	p.count = widget.NewLabel("")
	p.selection = widget.NewLabel("Выберите проект в списке")
	p.selection.Wrapping = fyne.TextWrapWord
	p.empty = label("Проекты не найдены. Попробуйте другой запрос.")
	p.empty.Alignment = fyne.TextAlignCenter
	p.activity = widget.NewActivity()
	p.activity.Hide()
	p.progress = widget.NewLabel("")
	p.progress.Wrapping = fyne.TextWrapWord
	p.progress.Alignment = fyne.TextAlignCenter
	p.status = container.New(centeredStatusLayout{}, container.NewVBox(container.NewCenter(p.activity), p.empty))
	p.list = widget.NewList(func() int { return len(p.visible) }, func() fyne.CanvasObject {
		title := widget.NewLabel("Проект")
		title.Truncation = fyne.TextTruncateEllipsis
		number := widget.NewLabel("#12345")
		number.TextStyle.Monospace = true
		return container.NewBorder(nil, nil, widget.NewIcon(theme.FolderIcon()), number, title)
	}, func(i widget.ListItemID, o fyne.CanvasObject) {
		v := p.visible[i]
		row := o.(*fyne.Container)
		row.Objects[0].(*widget.Label).SetText(v.Name)
		row.Objects[2].(*widget.Label).SetText(fmt.Sprintf("#%d", v.ID))
	})
	p.list.OnSelected = func(i widget.ListItemID) { p.selected = i; p.selection.SetText(p.visible[i].Name); p.open.Enable() }
	p.open = widget.NewButton("Открыть проект", p.confirm)
	p.open.Importance = widget.HighImportance
	p.open.Disable()
	p.first = widget.NewButton("«", func() { p.goTo(0) })
	p.prev = widget.NewButton("Назад", func() { p.goTo(p.page - 1) })
	p.next = widget.NewButton("Вперёд", func() { p.goTo(p.page + 1) })
	p.last = widget.NewButton("»", func() { p.goTo(p.pages() - 1) })
	p.sortBy = widget.NewSelect([]string{"Название А–Я", "Номер ↑", "Номер ↓"}, func(string) { p.refilter() })
	p.sortBy.Selected = "Название А–Я"
	p.sizeSelect = widget.NewSelect([]string{"20", "50", "100"}, func(s string) { n, _ := strconv.Atoi(s); p.pageSize = n; p.goTo(0) })
	p.sizeSelect.Selected = "20"
	p.search.OnChanged = func(string) { p.refilter() }
	p.search.OnSubmitted = func(string) {
		if p.selected < 0 {
			if len(p.filtered) != 1 {
				return
			}
			p.list.Select(0)
		}
		p.confirm()
	}
	searchRow := container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.CancelIcon(), func() { p.search.SetText("") }), p.search)
	header := container.NewVBox(searchRow, container.NewBorder(nil, nil, widget.NewLabel("Сортировка"), container.NewGridWrap(fyne.NewSize(190, 36), p.sortBy), layout.NewSpacer()))
	pager := container.NewBorder(nil, nil, container.NewHBox(p.first, p.prev), container.NewHBox(p.next, p.last), container.NewCenter(p.count))
	selectionScroll := container.NewVScroll(p.selection)
	selectionScroll.SetMinSize(fyne.NewSize(0, 44))
	bottom := container.NewVBox(p.progress, selectionScroll, container.NewHBox(widget.NewLabel("На странице"), p.sizeSelect), pager, widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), widget.NewButton("Отмена", cancel), p.open))
	p.content = container.NewBorder(header, bottom, nil, nil, container.NewStack(container.NewThemeOverride(p.list, projectListTheme{}), p.status))
	p.refilter()
	return p
}
func (p *projectPicker) pages() int { return max(1, (len(p.filtered)+p.pageSize-1)/p.pageSize) }
func (p *projectPicker) refilter() {
	terms := strings.Fields(normalizeProjectText(p.search.Text))
	p.filtered = nil
	for _, v := range p.all {
		haystack := normalizeProjectText(v.Name) + " " + strconv.Itoa(v.ID)
		match := true
		for _, term := range terms {
			if !strings.Contains(haystack, strings.TrimPrefix(term, "#")) {
				match = false
				break
			}
		}
		if match {
			p.filtered = append(p.filtered, v)
		}
	}
	sort.SliceStable(p.filtered, func(i, j int) bool {
		a, b := p.filtered[i], p.filtered[j]
		switch p.sortBy.Selected {
		case "Номер ↑":
			return a.ID < b.ID
		case "Номер ↓":
			return a.ID > b.ID
		}
		an, bn := normalizeProjectText(a.Name), normalizeProjectText(b.Name)
		if an == bn {
			return a.ID < b.ID
		}
		return an < bn
	})
	p.goTo(0)
}
func (p *projectPicker) goTo(page int) {
	p.page = max(0, min(page, p.pages()-1))
	start := p.page * p.pageSize
	end := min(start+p.pageSize, len(p.filtered))
	p.visible = p.filtered[start:end]
	p.selected = -1
	p.list.UnselectAll()
	p.list.Refresh()
	p.list.ScrollToTop()
	p.open.Disable()
	p.selection.SetText("Выберите проект в списке")
	if len(p.filtered) == 0 {
		p.list.Hide()
		p.status.Show()
		p.empty.SetText("Проекты не найдены. Попробуйте другой запрос.")
		if p.loading {
			p.empty.SetText("Загружаем список проектов…")
		} else if p.loadError != nil {
			p.empty.SetText("Не удалось загрузить проекты: " + p.loadError.Error())
		}
		p.count.SetText("Найдено: 0")
		if p.loading {
			p.count.SetText("Загрузка…")
		}
	} else {
		p.status.Hide()
		p.list.Show()
		p.count.SetText(fmt.Sprintf("%d–%d из %d · %d / %d", start+1, end, len(p.filtered), p.page+1, p.pages()))
	}
	p.first.Disable()
	p.prev.Disable()
	p.next.Disable()
	p.last.Disable()
	p.activity.Stop()
	p.activity.Hide()
	p.progress.Hide()
	if p.loading {
		if len(p.filtered) == 0 {
			p.activity.Show()
			p.activity.Start()
		}
		if len(p.all) > 0 {
			p.progress.Show()
		}
		if p.total > 0 {
			p.progress.SetText(fmt.Sprintf("Загружено %d из %d · Можно выбирать проект", len(p.all), p.total))
		} else {
			p.progress.SetText("Подключаемся к серверу…")
		}
	}
	if p.loadError != nil && len(p.all) > 0 {
		p.progress.Show()
		p.progress.SetText("Показана загруженная часть списка. " + p.loadError.Error())
	}
	if p.page > 0 {
		p.first.Enable()
		p.prev.Enable()
	}
	if p.page+1 < p.pages() {
		p.next.Enable()
		p.last.Enable()
	}
}
func (p *projectPicker) confirm() {
	if p.selected >= 0 && p.selected < len(p.visible) {
		p.choose(p.visible[p.selected])
	}
}
func (u *UI) showProjectPicker(choose func(redmine.Ref)) {
	var d *dialog.CustomDialog
	p := newProjectPicker(u.projects, func(v redmine.Ref) { d.Hide(); choose(v) }, func() { d.Hide() })
	u.activeProjects = p
	p.updateProjects(u.projects, u.projectsTotal, u.projectsLoading, u.projectsError)
	d = dialog.NewCustomWithoutButtons("Проекты", p.content, u.win)
	d.SetOnClosed(func() {
		if u.activeProjects == p {
			u.activeProjects = nil
		}
		p.activity.Stop()
	})
	size := u.win.Canvas().Size()
	d.Resize(fyne.NewSize(min(740, max(320, size.Width-64)), min(620, max(360, size.Height-64))))
	d.Show()
	u.win.Canvas().Focus(p.search)
}
func (u *UI) chooseProject() {
	u.showProjectPicker(func(p redmine.Ref) { u.mode = p.Name; u.project = p.ID; u.offset = 0; u.refresh() })
}

// The task list paints selection on its own tiles; plain project rows need a visible selection background.
type projectListTheme struct{}

func (projectListTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameSelection {
		return currentColors().raised
	}
	return fyne.CurrentApp().Settings().Theme().Color(n, v)
}
func (projectListTheme) Size(n fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(n)
}
func (projectListTheme) Font(s fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(s)
}
func (projectListTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(n)
}

func (p *projectPicker) updateProjects(projects []redmine.Ref, total int, loading bool, err error) {
	page := p.page
	selectedID := 0
	if p.selected >= 0 && p.selected < len(p.visible) {
		selectedID = p.visible[p.selected].ID
	}
	p.all = append([]redmine.Ref(nil), projects...)
	p.total = total
	p.loading = loading
	p.loadError = err
	p.refilter()
	if selectedID != 0 {
		for i, v := range p.filtered {
			if v.ID == selectedID {
				page = i / p.pageSize
				break
			}
		}
	}
	p.goTo(page)
	if selectedID != 0 {
		for i, v := range p.visible {
			if v.ID == selectedID {
				p.list.Select(i)
				break
			}
		}
	}
}

// Fill the status area's width for wrapping/alignment, and center the whole group vertically.
type centeredStatusLayout struct{}

func (centeredStatusLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 0) }
func (centeredStatusLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		h := o.MinSize().Height
		o.Resize(fyne.NewSize(size.Width, h))
		h = o.MinSize().Height
		o.Resize(fyne.NewSize(size.Width, h))
		o.Move(fyne.NewPos(0, max(0, (size.Height-h)/2)))
	}
}
