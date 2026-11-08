// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/rkislov/redmineapp/internal/redmine"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestProjectSearchPaginationAndSelection(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(appTheme{})
	projects := []redmine.Ref{}
	for i := 1; i <= 1405; i++ {
		projects = append(projects, redmine.Ref{ID: i, Name: fmt.Sprintf("%04d Компания · Проект", i)})
	}
	projects = append(projects, redmine.Ref{ID: 8000, Name: strings.Repeat("Очень длинное название компании ", 40)})
	chosen := 0
	p := newProjectPicker(projects, func(v redmine.Ref) { chosen = v.ID }, func() {})
	w := a.NewWindow("redmineapp — Проекты")
	w.Resize(fyne.NewSize(740, 620))
	w.SetContent(container.NewPadded(p.content))
	p.search.OnSubmitted(p.search.Text)
	if chosen != 0 {
		t.Fatal("broad search must not choose an arbitrary project")
	}
	if len(p.visible) != 20 || p.pages() != 71 || !p.prev.Disabled() || p.next.Disabled() {
		t.Fatal("initial pagination incorrect")
	}
	if p.content.MinSize().Width > 740 {
		t.Fatalf("long name expanded picker to %v", p.content.MinSize())
	}
	p.goTo(70)
	if len(p.visible) != 6 || !p.next.Disabled() {
		t.Fatal("last page incorrect")
	}
	p.search.SetText("Компания 1395")
	if len(p.filtered) != 1 || p.page != 0 || p.visible[0].ID != 1395 {
		t.Fatal("search must include all pages and match terms")
	}
	p.list.Select(0)
	p.confirm()
	if chosen != 1395 {
		t.Fatal("selected wrong project")
	}
	p.search.SetText("#8000")
	p.search.OnSubmitted(p.search.Text)
	if chosen != 8000 {
		t.Fatal("Enter must open matching project")
	}
	p.search.SetText("нет такого проекта")
	if len(p.visible) != 0 || !p.open.Disabled() || !p.next.Disabled() {
		t.Fatal("empty state incorrect")
	}
	p.search.SetText("")
	p.sizeSelect.SetSelected("100")
	if len(p.visible) != 100 || p.pages() != 15 {
		t.Fatal("page size incorrect")
	}
	p.sortBy.SetSelected("Номер ↓")
	if p.visible[0].ID != 8000 {
		t.Fatal("sort incorrect")
	}
	p.sizeSelect.SetSelected("20")
	p.sortBy.SetSelected("Название А–Я")
	p.list.Select(1)
	if file := os.Getenv("REDMINE_PROJECTS_SCREENSHOT"); file != "" {
		f, e := os.Create(file)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if e = png.Encode(f, w.Canvas().Capture()); e != nil {
			t.Fatal(e)
		}
	}
}
func TestAutoAppearanceTracksSystemVariant(t *testing.T) {
	auto := appTheme{Mode: "auto"}
	if auto.isDark(theme.VariantLight) || !auto.isDark(theme.VariantDark) {
		t.Fatal("auto theme must follow OS")
	}
	for _, v := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		if (appTheme{Mode: "light"}).isDark(v) || !(appTheme{Mode: "dark"}).isDark(v) {
			t.Fatal("manual theme must override OS")
		}
	}
	if auto.Color(theme.ColorNameBackground, theme.VariantLight) == auto.Color(theme.ColorNameBackground, theme.VariantDark) {
		t.Fatal("auto colors did not change")
	}
	a := test.NewApp()
	defer a.Quit()
	toggle := newAppearanceToggle("auto", nil)
	for i, mode := range []string{"light", "auto", "dark"} {
		if toggle.buttons[i].Text != "" || toggle.buttons[i].Icon == nil {
			t.Fatal("appearance switch must use icons")
		}
		test.Tap(toggle.buttons[i])
		if toggle.Selected != mode {
			t.Fatal("wrong switch order")
		}
	}
}

func TestProjectLoadingStatusCenteredAndPartialPagesUsable(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(appTheme{})
	p := newProjectPicker(nil, func(redmine.Ref) {}, func() {})
	w := a.NewWindow("redmineapp — загрузка проектов")
	w.Resize(fyne.NewSize(740, 620))
	w.SetContent(container.NewPadded(p.content))
	p.updateProjects(nil, 0, true, nil)
	w.Content().Refresh()
	if p.empty.Alignment != fyne.TextAlignCenter || p.empty.Size().Width < p.status.Size().Width-1 {
		t.Fatal("loading text must span the centered status area")
	}
	group := p.status.Objects[0]
	if delta := group.Position().Y + group.Size().Height/2 - p.status.Size().Height/2; delta < -1 || delta > 1 {
		t.Fatal("status group is not vertically centered")
	}
	if p.count.Text == "Найдено: 0" {
		t.Fatal("loading was presented as an empty result")
	}
	if file := os.Getenv("REDMINE_LOADING_SCREENSHOT"); file != "" {
		f, e := os.Create(file)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if e = png.Encode(f, w.Canvas().Capture()); e != nil {
			t.Fatal(e)
		}
	}
	refs := []redmine.Ref{}
	for i := 1; i <= 100; i++ {
		refs = append(refs, redmine.Ref{ID: i, Name: fmt.Sprintf("Проект %03d", i)})
	}
	p.updateProjects(refs, 1400, true, nil)
	if !p.status.Hidden || p.list.Hidden || len(p.visible) != 20 {
		t.Fatal("partial list should be usable before all pages finish")
	}
	p.goTo(2)
	p.list.Select(1)
	id := p.visible[1].ID
	refs = append(refs, redmine.Ref{ID: 1001, Name: "А самый первый проект"})
	p.updateProjects(refs, 1400, true, nil)
	if p.page != 2 || p.selected < 0 || p.visible[p.selected].ID != id {
		t.Fatal("incoming pages changed selected project")
	}
	p.search.SetText("самый первый")
	p.updateProjects(refs, 101, false, nil)
	if len(p.visible) != 1 || p.visible[0].ID != 1001 {
		t.Fatal("incoming pages reset search")
	}
}
