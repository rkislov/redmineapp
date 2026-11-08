// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/rkislov/redmineapp/internal/redmine"
)

func statusIsClosed(status redmine.Ref, all []redmine.Ref) bool {
	if status.IsClosed {
		return true
	}
	for _, s := range all {
		if s.ID == status.ID {
			return s.IsClosed
		}
	}
	return false
}
func closedStatusChoices(v redmine.Issue, all []redmine.Ref) []redmine.Ref {
	result := []redmine.Ref{}
	for _, s := range issueStatusChoices(v, all) {
		if s.ID != v.Status.ID && statusIsClosed(s, all) {
			result = append(result, s)
		}
	}
	return result
}
func (u *UI) closeIssue(v redmine.Issue) {
	if statusIsClosed(v.Status, u.statuses) {
		dialog.ShowInformation("Задача уже закрыта", "Эта задача уже находится в закрытом статусе.", u.win)
		return
	}
	options := closedStatusChoices(v, u.statuses)
	if len(options) == 0 {
		message := "Для этой задачи нет разрешённых переходов в закрытый статус. Проверьте права и настройки рабочего процесса в Redmine."
		if len(u.statuses) == 0 {
			message = "Список статусов ещё недоступен. Повторите действие после его загрузки."
		}
		dialog.ShowInformation("Закрытие недоступно", message, u.win)
		return
	}
	save := func(s redmine.Ref) { u.update(v.ID, map[string]any{"status_id": s.ID}) }
	if len(options) == 1 {
		target := options[0]
		dialog.ShowConfirm("Закрыть задачу?", fmt.Sprintf("#%d · %s\nСтатус: %s", v.ID, v.Subject, target.Name), func(ok bool) {
			if ok {
				save(target)
			}
		}, u.win)
		return
	}
	var d *dialog.CustomDialog
	p := newRefPicker(options, 0, func(s redmine.Ref) { d.Hide(); save(s) }, func() { d.Hide() })
	p.confirm.SetText("Закрыть задачу")
	d = dialog.NewCustomWithoutButtons("Выберите закрытый статус", p.content, u.win)
	d.Resize(fyne.NewSize(480, 400))
	d.Show()
	u.win.Canvas().Focus(p.search)
}
