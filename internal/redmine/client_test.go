// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package redmine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWorkflow(t *testing.T) {
	var updates int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Redmine-API-Key") != "secret" {
			t.Error("missing key")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /redmine/users/current.json":
			fmt.Fprint(w, `{"user":{"id":7,"firstname":"Иван"}}`)
		case "GET /redmine/issues.json":
			if r.URL.Query().Get("subject") != "~тест & UI" {
				t.Error("query escaped incorrectly")
			}
			fmt.Fprint(w, `{"issues":[{"id":42,"subject":"Тест"}],"total_count":1}`)
		case "GET /redmine/issues/42.json":
			if r.URL.Query().Get("include") == "" {
				t.Error("no journals")
			}
			fmt.Fprint(w, `{"issue":{"id":42,"journals":[{"notes":"Привет"}]}}`)
		case "PUT /redmine/issues/42.json":
			var b struct {
				Issue map[string]any `json:"issue"`
			}
			if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
				t.Fatal(e)
			}
			if b.Issue["notes"] != "Заметка" {
				t.Error("wrong update")
			}
			updates++
			w.WriteHeader(204)
		case "POST /redmine/issues.json":
			w.WriteHeader(201)
			fmt.Fprint(w, `{"issue":{"id":43}}`)
		case "POST /redmine/time_entries.json":
			var b map[string]map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["time_entry"]["hours"] != 1.5 {
				t.Error("hours")
			}
			w.WriteHeader(201)
			fmt.Fprint(w, `{"time_entry":{"id":1}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(404)
		}
	}))
	defer s.Close()
	c, e := New(s.URL+"/redmine/", "secret")
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	if u, e := c.Me(ctx); e != nil || u.ID != 7 {
		t.Fatalf("me: %v %v", u, e)
	}
	if p, e := c.Issues(ctx, url.Values{"subject": {"~тест & UI"}}); e != nil || p.Total != 1 {
		t.Fatalf("issues: %v %v", p, e)
	}
	if v, e := c.Issue(ctx, 42); e != nil || len(v.Journals) != 1 {
		t.Fatalf("issue: %v %v", v, e)
	}
	if e = c.Update(ctx, 42, map[string]any{"notes": "Заметка"}); e != nil {
		t.Fatal(e)
	}
	if updates != 1 {
		t.Fatal("update missing")
	}
	if v, e := c.Create(ctx, map[string]any{"subject": "Новая"}); e != nil || v.ID != 43 {
		t.Fatalf("create: %v %v", v, e)
	}
	if e = c.LogTime(ctx, map[string]any{"issue_id": 42, "hours": 1.5}); e != nil {
		t.Fatal(e)
	}
}
func TestRedirectDoesNotLeakKey(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer s.Close()
	c, _ := New(s.URL, "secret")
	_, e := c.Me(context.Background())
	if e == nil || called {
		t.Fatal("redirect followed")
	}
}
func TestValidationErrors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		fmt.Fprint(w, `{"errors":["Тема обязательна"]}`)
	}))
	defer s.Close()
	c, _ := New(s.URL, "secret")
	e := c.Update(context.Background(), 1, map[string]any{})
	if e == nil || !strings.Contains(e.Error(), "Тема обязательна") {
		t.Fatalf("error: %v", e)
	}
}
func TestURLValidation(t *testing.T) {
	for _, raw := range []string{"", "file:///etc/passwd", "https://user:password@host", "https://host?api_key=secret", "https://host/#fragment"} {
		if _, e := New(raw, "key"); e == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
func TestProjectPagination(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") == "0" {
			fmt.Fprint(w, `{"projects":[{"id":1,"name":"A"}],"total_count":2}`)
		} else {
			fmt.Fprint(w, `{"projects":[{"id":2,"name":"B"}],"total_count":2}`)
		}
	}))
	defer s.Close()
	c, _ := New(s.URL, "secret")
	v, e := c.Refs(context.Background(), "projects.json", "projects")
	if e != nil || len(v) != 2 {
		t.Fatalf("%v %v", v, e)
	}
}

func TestDemoEditsRemainInMemory(t *testing.T) {
	c := NewDemo()
	ctx := context.Background()
	before, e := c.Issue(ctx, 246614)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Update(ctx, 246614, map[string]any{"notes": "Тестовый комментарий", "done_ratio": 70}); e != nil {
		t.Fatal(e)
	}
	if e = c.LogTime(ctx, map[string]any{"issue_id": 246614, "hours": 1.5}); e != nil {
		t.Fatal(e)
	}
	after, e := c.Issue(ctx, 246614)
	if e != nil {
		t.Fatal(e)
	}
	if after.Done != 70 || len(after.Journals) != len(before.Journals)+1 || after.Spent != before.Spent+1.5 {
		t.Fatalf("edit not applied: %+v", after)
	}
	fresh, e := NewDemo().Issue(ctx, 246614)
	if e != nil || fresh.Done != before.Done {
		t.Fatal("demo leaked between sessions")
	}
}

func TestBasicAuthentication(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		login, password, ok := r.BasicAuth()
		if !ok || login != "roman" || password != " p@ss:тест " {
			t.Error("basic credentials were not preserved")
		}
		if r.Header.Get("X-Redmine-API-Key") != "" {
			t.Error("mixed authentication modes")
		}
		if r.URL.RawQuery != "" || r.URL.User != nil {
			t.Error("credentials in URL")
		}
		fmt.Fprint(w, `{"user":{"id":9}}`)
	}))
	defer s.Close()
	c, e := NewBasic(s.URL+"/redmine", " roman ", " p@ss:тест ")
	if e != nil {
		t.Fatal(e)
	}
	u, e := c.Me(context.Background())
	if e != nil || u.ID != 9 {
		t.Fatalf("login failed: %v", e)
	}
	for _, pair := range [][2]string{{"", "pass"}, {"user:name", "pass"}, {"user", ""}} {
		if _, e := NewBasic(s.URL, pair[0], pair[1]); e == nil {
			t.Error("accepted invalid credentials")
		}
	}
}
func TestBasicRedirectDoesNotLeakCredentials(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer s.Close()
	c, _ := NewBasic(s.URL, "user", "password")
	if _, e := c.Me(context.Background()); e == nil || called {
		t.Fatal("redirect followed with credentials")
	}
}
