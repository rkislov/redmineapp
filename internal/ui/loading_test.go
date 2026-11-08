// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fmt"
	"fyne.io/fyne/v2/test"
	"github.com/rkislov/redmineapp/internal/redmine"
	"github.com/rkislov/redmineapp/internal/settings"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBothPaneAnimationsStopOnSuccessAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			a := test.NewApp()
			defer a.Quit()
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if fail {
					w.WriteHeader(500)
					return
				}
				if r.URL.Path == "/issues.json" {
					fmt.Fprint(w, `{"issues":[{"id":1,"subject":"Задача"}],"total_count":1}`)
				} else {
					fmt.Fprint(w, `{"issue":{"id":1,"subject":"Задача","project":{"name":"Проект"}}}`)
				}
			}))
			defer s.Close()
			c, _ := redmine.New(s.URL, "test")
			q := make(chan func(), 10)
			u := &UI{app: a, win: a.NewWindow("test"), client: c, status: "open", mode: "Мои", cfg: settings.Settings{Favorites: map[string]bool{}}, post: func(fn func()) { q <- fn }}
			defer u.stopWork()
			u.shell()
			u.refresh()
			if !u.listLoading.running || !u.list.Hidden {
				t.Fatal("task loading animation missing")
			}
			nextUI(t, q)
			if u.listLoading.running {
				t.Fatal("task animation did not stop")
			}
			if !fail && u.list.Hidden {
				t.Fatal("list not restored after success")
			}
			u.openIssue(1)
			if !u.detailLoading.running {
				t.Fatal("description animation missing")
			}
			nextUI(t, q)
			if u.detailLoading.running {
				t.Fatal("description animation did not stop")
			}
		})
	}
}
func TestStaleIssueResponseCannotStopNewLoadingAnimation(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	started := make(chan struct{})
	release := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/issues/1.json" {
			close(started)
			<-r.Context().Done()
			return
		}
		<-release
		fmt.Fprint(w, `{"issue":{"id":2,"subject":"Вторая задача"}}`)
	}))
	defer s.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	c, _ := redmine.New(s.URL, "test")
	q := make(chan func(), 10)
	u := &UI{app: a, win: a.NewWindow("test"), client: c, cfg: settings.Settings{Favorites: map[string]bool{}}, post: func(fn func()) { q <- fn }}
	defer u.stopWork()
	u.shell()
	u.openIssue(1)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first request did not start")
	}
	u.openIssue(2)
	nextUI(t, q)
	if !u.detailLoading.running || u.selected != 2 {
		t.Fatal("stale request stopped the current animation")
	}
	close(release)
	nextUI(t, q)
	if u.detailLoading.running || u.selected != 2 {
		t.Fatal("new issue did not finish loading")
	}
}
