// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package redmine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DemoTransport keeps sample data in memory and never accesses the network.
type demoTransport struct {
	mu     sync.Mutex
	issues []Issue
	next   int
}

func NewDemo() *Client {
	c, _ := New("https://demo.invalid", "demo")
	names := []string{"Выпуск инфраструктуры", "План разработки 2027", "Пуско-наладочные работы по тестированию инфраструктуры", "Регистрация сделки и запрос цен", "Интеграция процессов в сервисах"}
	d := &demoTransport{next: 246620}
	for n, s := range names {
		d.issues = append(d.issues, Issue{ID: 246612 + n, Subject: s, Description: "Требуется подготовить всё для тестирования инфраструктуры.\n\nПосле коммита в репозиторий должен запуститься процесс автоматизированного тестирования с отчётом в задачу.\n\nЭто демонстрационные данные. Изменения сохраняются только до закрытия приложения.", Project: Ref{1, "Инфраструктура"}, Tracker: Ref{2, "Задача"}, Status: Ref{2, "В работе"}, Priority: Ref{2, "Нормальный"}, Author: Ref{1, "Антон Палгунов"}, AssignedTo: Ref{1, "Иванов Сергей"}, Done: []int{62, 10, 0, 60, 20}[n], Start: "2026-07-28", Updated: "2026-10-23T18:57:00+03:00", Spent: 55.5, Journals: []Journal{{ID: 1, User: Ref{1, "Антон Палгунов"}, Notes: "Окружение подготовлено. Можно начинать проверку.", Created: "2026-10-24T10:00:00+03:00"}}})
	}
	c.HTTP = &http.Client{Transport: d, Timeout: time.Second}
	return c
}
func (d *demoTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var value any
	code := 200
	path := r.URL.Path
	switch {
	case path == "/users/current.json":
		value = map[string]any{"user": User{ID: 1, Firstname: "Сергей", Lastname: "Иванов"}}
	case path == "/projects.json":
		value = map[string]any{"projects": []Ref{{1, "Инфраструктура"}, {2, "Продукты"}}, "total_count": 2}
	case path == "/issue_statuses.json":
		value = map[string]any{"issue_statuses": []Ref{{1, "Новая"}, {2, "В работе"}, {3, "Решена"}, {4, "Закрыта"}}}
	case path == "/trackers.json":
		value = map[string]any{"trackers": []Ref{{1, "Ошибка"}, {2, "Задача"}}}
	case path == "/enumerations/issue_priorities.json":
		value = map[string]any{"issue_priorities": []Ref{{1, "Низкий"}, {2, "Нормальный"}, {3, "Высокий"}}}
	case path == "/enumerations/time_entry_activities.json":
		value = map[string]any{"time_entry_activities": []Ref{{1, "Разработка"}, {2, "Тестирование"}}}
	case path == "/issues.json" && r.Method == "GET":
		issues := []Issue{}
		q := r.URL.Query()
		for _, v := range d.issues {
			if q.Get("project_id") != "" && q.Get("project_id") != fmt.Sprint(v.Project.ID) {
				continue
			}
			if q.Get("issue_id") != "" && !strings.Contains(","+q.Get("issue_id")+",", fmt.Sprintf(",%d,", v.ID)) {
				continue
			}
			if q.Get("subject") != "" && !strings.Contains(strings.ToLower(v.Subject), strings.ToLower(strings.TrimPrefix(q.Get("subject"), "~"))) {
				continue
			}
			closed := v.Status.ID == 3 || v.Status.ID == 4
			if q.Get("status_id") == "open" && closed || q.Get("status_id") == "closed" && !closed {
				continue
			}
			issues = append(issues, v)
		}
		total := len(issues)
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 {
			limit = 50
		}
		offset = min(max(offset, 0), total)
		issues = issues[offset:min(offset+limit, total)]
		value = Page{Issues: issues, Total: total}
	case path == "/issues.json" && r.Method == "POST":
		var b struct {
			Issue Issue `json:"issue"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		b.Issue.ID = d.next
		d.next++
		b.Issue.Project = Ref{1, "Инфраструктура"}
		b.Issue.Status = Ref{1, "Новая"}
		b.Issue.Updated = time.Now().Format(time.RFC3339)
		d.issues = append([]Issue{b.Issue}, d.issues...)
		value = map[string]any{"issue": b.Issue}
		code = 201
	case strings.HasPrefix(path, "/issues/"):
		id, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(path, "/issues/"), ".json"))
		index := -1
		for i := range d.issues {
			if d.issues[i].ID == id {
				index = i
				break
			}
		}
		if index < 0 {
			code = 404
			break
		}
		v := &d.issues[index]
		if r.Method == "PUT" {
			var b struct {
				Issue map[string]json.RawMessage `json:"issue"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			for k, raw := range b.Issue {
				switch k {
				case "subject":
					_ = json.Unmarshal(raw, &v.Subject)
				case "description":
					_ = json.Unmarshal(raw, &v.Description)
				case "done_ratio":
					_ = json.Unmarshal(raw, &v.Done)
				case "due_date":
					_ = json.Unmarshal(raw, &v.Due)
				case "assigned_to_id":
					v.AssignedTo = Ref{1, "Иванов Сергей"}
				case "status_id":
					var n int
					_ = json.Unmarshal(raw, &n)
					names := map[int]string{1: "Новая", 2: "В работе", 3: "Решена", 4: "Закрыта"}
					v.Status = Ref{n, names[n]}
				case "priority_id":
					var n int
					_ = json.Unmarshal(raw, &n)
					v.Priority = Ref{n, map[int]string{1: "Низкий", 2: "Нормальный", 3: "Высокий"}[n]}
				case "notes":
					var notes string
					_ = json.Unmarshal(raw, &notes)
					v.Journals = append(v.Journals, Journal{ID: len(v.Journals) + 1, User: Ref{1, "Иванов Сергей"}, Notes: notes, Created: time.Now().Format(time.RFC3339)})
				}
			}
			v.Updated = time.Now().Format(time.RFC3339)
			code = 204
		} else {
			value = map[string]any{"issue": v}
		}
	case path == "/time_entries.json" && r.Method == "POST":
		var b struct {
			Entry struct {
				ID    int     `json:"issue_id"`
				Hours float64 `json:"hours"`
			} `json:"time_entry"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		for i := range d.issues {
			if d.issues[i].ID == b.Entry.ID {
				d.issues[i].Spent += b.Entry.Hours
			}
		}
		code = 201
		value = map[string]any{"time_entry": map[string]int{"id": 1}}
	default:
		code = 404
	}
	b, _ := json.Marshal(value)
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(b))), Request: r}, nil
}
