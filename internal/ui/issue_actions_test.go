// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/rkislov/redmineapp/internal/redmine"
	"testing"
)

func TestProgressChangesByClickAndIgnoresClicksWhileSaving(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	p := newEditableProgress()
	p.Resize(fyne.NewSize(240, 24))
	p.SetValue(0)
	chosen := -1
	p.OnChanged = func(n int) { chosen = n; p.SetValue(float64(n) / 100) }
	for _, sample := range []struct {
		x    float32
		want int
	}{{100, 50}, {-10, 0}, {199, 100}, {65, 30}, {250, 100}} {
		p.Tapped(&fyne.PointEvent{Position: fyne.NewPos(sample.x, 12)})
		if chosen != sample.want {
			t.Fatalf("tap at %v = %d, want %d", sample.x, chosen, sample.want)
		}
	}
	p.busy = true
	p.Tapped(&fyne.PointEvent{Position: fyne.NewPos(0, 12)})
	if chosen != 100 {
		t.Fatal("click changed value during save")
	}
}
func TestClosedStatusChoicesUseServerFlagAndAllowedTransitions(t *testing.T) {
	global := []redmine.Ref{{ID: 2, Name: "Решена"}, {ID: 3, Name: "Архив", IsClosed: true}, {ID: 4, Name: "Закрыта", IsClosed: true}}
	v := redmine.Issue{Status: global[0], AllowedStatuses: []redmine.Ref{{ID: 3, Name: "Архив"}}}
	if statusIsClosed(v.Status, global) {
		t.Fatal("status name must not imply closed")
	}
	choices := closedStatusChoices(v, global)
	if len(choices) != 1 || choices[0].ID != 3 {
		t.Fatal("must use an allowed status marked closed by the server")
	}
	v.AllowedStatuses = []redmine.Ref{}
	if len(closedStatusChoices(v, global)) != 0 {
		t.Fatal("no transitions allowed")
	}
}
