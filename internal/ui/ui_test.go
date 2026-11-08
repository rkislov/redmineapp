// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/rkislov/redmineapp/internal/redmine"
	"github.com/rkislov/redmineapp/internal/settings"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestThreePaneIssueView(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	c, e := redmine.New("https://redmine.example.org", "demo")
	if e != nil {
		t.Fatal(e)
	}
	u := &UI{app: a, win: a.NewWindow("redmineapp"), cfg: settings.Settings{Favorites: map[string]bool{}}, client: c, mode: "Слежу", status: "open", me: redmine.User{ID: 1, Firstname: "Иван", Lastname: "Сергеев"}}
	a.Settings().SetTheme(appTheme{})
	u.win.Resize(fyne.NewSize(1240, 820))
	u.shell()
	v := redmine.Issue{ID: 246614, Subject: "Пуско-наладочные работы по тестированию инфраструктуры", Description: "Требуется подготовить всё для тестирования инфраструктуры.\n\nПосле коммита в репозиторий должен запуститься процесс автоматизированного тестирования с отчётом в задачу.", Project: redmine.Ref{ID: 1, Name: "Инфраструктура"}, Status: redmine.Ref{ID: 2, Name: "В работе"}, Priority: redmine.Ref{ID: 2, Name: "Нормальный"}, Tracker: redmine.Ref{Name: "Тестирование"}, AssignedTo: redmine.Ref{Name: "Иванов Сергей"}, Author: redmine.Ref{Name: "Антон Палгунов"}, Start: "2026-07-28", Updated: "2026-10-23T18:57:00+03:00", Spent: 55.5, Journals: []redmine.Journal{{User: redmine.Ref{Name: "Антон Палгунов"}, Notes: "Окружение подготовлено. Можно начинать проверку.", Created: "2026-10-24T10:00:00+03:00"}}}
	u.issues = []redmine.Issue{{ID: 230375, Subject: "Выпуск инфраструктуры", Project: redmine.Ref{Name: "Выпуск релизов"}, Status: redmine.Ref{Name: "В работе"}, AssignedTo: redmine.Ref{Name: "Семёнов Андрей"}, Done: 62}, {ID: 258719, Subject: "План разработки 2027", Project: redmine.Ref{Name: "Продукты"}, Status: redmine.Ref{Name: "В работе"}, AssignedTo: redmine.Ref{Name: "Щавелёв Александр"}, Done: 10}, v, {ID: 261099, Subject: "Интеграция процессов в сервисах", Project: redmine.Ref{Name: "Бизнес-процессы"}, Status: redmine.Ref{Name: "Новая"}, Done: 20}}
	u.selected = v.ID
	u.list.Refresh()
	u.showIssue(v)
	if len(u.detail.Objects) != 1 {
		t.Fatal("detail missing")
	}
	// Set REDMINE_SCREENSHOT to export a deterministic software-rendered preview.
	if p := os.Getenv("REDMINE_SCREENSHOT"); p != "" {
		if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			t.Fatal(e)
		}
		f, e := os.Create(p)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if e = png.Encode(f, u.win.Canvas().Capture()); e != nil {
			t.Fatal(e)
		}
	}
	if p := os.Getenv("REDMINE_SCREENSHOT_DARK"); p != "" {
		u.appearanceSelect.Selected = "dark"
		u.appearanceSelect.updateButtons()
		u.appearanceSelect.Refresh()
		a.Settings().SetTheme(appTheme{Dark: true})
		u.win.Content().Refresh()
		f, e := os.Create(p)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if e = png.Encode(f, u.win.Canvas().Capture()); e != nil {
			t.Fatal(e)
		}
	}

}

func TestAppearancePersistsAcrossLaunches(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("APPDATA", d)
	t.Setenv("XDG_CONFIG_HOME", d)
	a := test.NewApp()
	defer a.Quit()
	u := New(a)
	u.appearanceSelect.SetSelected("dark")
	saved, e := settings.Load()
	if e != nil || saved.Appearance != "dark" {
		t.Fatalf("theme not saved: %v", e)
	}
	if !a.Settings().Theme().(appTheme).Dark {
		t.Fatal("night mode not applied")
	}
	// Reload the window using the saved setting without a Redmine connection.
	next := New(a)
	if next.appearanceSelect.Selected != "dark" || !a.Settings().Theme().(appTheme).Dark {
		t.Fatal("night mode lost on reload")
	}
	next.appearanceSelect.SetSelected("light")
	saved, e = settings.Load()
	if e != nil || saved.Appearance != "light" || a.Settings().Theme().(appTheme).Dark {
		t.Fatal("day mode not restored")
	}

	next.appearanceSelect.SetSelected("auto")
	saved, e = settings.Load()
	if e != nil || saved.Appearance != "auto" || a.Settings().Theme().(appTheme).Mode != "auto" {
		t.Fatal("auto mode not saved")
	}
}
