// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/rkislov/redmineapp/internal/redmine"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestCompactStatusPickerSearchAndCancel(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(appTheme{})
	refs := []redmine.Ref{{ID: 1, Name: "Новая заявка"}, {ID: 2, Name: "В работе"}, {ID: 3, Name: "Заявка согласована"}}
	for i := 4; i <= 300; i++ {
		refs = append(refs, redmine.Ref{ID: i, Name: fmt.Sprintf("Статус %d %s", i, strings.Repeat("очень длинное название ", 20))})
	}
	chosen := 0
	p := newRefPicker(refs, 2, func(v redmine.Ref) { chosen = v.ID }, func() {})
	w := a.NewWindow("redmineapp — выбор статуса")
	w.Resize(fyne.NewSize(480, 400))
	w.SetContent(container.NewPadded(p.content))
	if p.content.MinSize().Width > 480 {
		t.Fatal("long names expanded the picker")
	}
	if p.selected != 1 {
		t.Fatal("current status not selected")
	}
	p.search.SetText("ЗАЯВКА согласована")
	if len(p.filtered) != 1 || p.filtered[0].ID != 3 {
		t.Fatal("name search failed")
	}
	p.search.OnSubmitted(p.search.Text)
	if chosen != 3 {
		t.Fatal("Enter chose wrong status")
	}
	p.search.SetText("#280")
	if len(p.filtered) != 1 || p.filtered[0].ID != 280 {
		t.Fatal("ID search failed")
	}
	p.search.SetText("нет такого статуса")
	if len(p.filtered) != 0 || !p.confirm.Disabled() {
		t.Fatal("empty search accepted stale selection")
	}
	p.search.SetText("")
	p.list.Select(1)
	if file := os.Getenv("REDMINE_STATUS_SCREENSHOT"); file != "" {
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
func TestIssueStatusChoicesRespectWorkflow(t *testing.T) {
	v := redmine.Issue{Status: redmine.Ref{ID: 2, Name: "В работе"}, AllowedStatuses: []redmine.Ref{{ID: 3, Name: "Решена"}}}
	global := []redmine.Ref{{ID: 99, Name: "Другой процесс"}}
	refs := issueStatusChoices(v, global)
	if len(refs) != 2 || refs[0].ID != 2 || refs[1].ID != 3 {
		t.Fatal("unrelated global statuses were included")
	}
	v.AllowedStatuses = []redmine.Ref{}
	if refs = issueStatusChoices(v, global); len(refs) != 1 || refs[0].ID != 2 {
		t.Fatal("empty allowed list must not fall back to unrelated statuses")
	}
	v.AllowedStatuses = nil
	if refs = issueStatusChoices(v, global); len(refs) != 2 || refs[1].ID != 99 {
		t.Fatal("legacy server fallback missing")
	}
}
