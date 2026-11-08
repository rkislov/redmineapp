// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2/test"
	"github.com/rkislov/redmineapp/internal/settings"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRestoreSavedConnectionSupportsBothAuthModes(t *testing.T) {
	for _, mode := range []string{"api_key", "basic"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if mode == "basic" {
					name, secret, ok := r.BasicAuth()
					if !ok || name != "roman" || secret != "saved-secret" {
						t.Error("wrong basic credentials")
					}
				} else if r.Header.Get("X-Redmine-API-Key") != "saved-secret" {
					t.Error("missing saved key")
				}
				fmt.Fprint(w, `{"user":{"id":7}}`)
			}))
			defer s.Close()
			p := settings.Profile{URL: s.URL, Username: "roman", Auth: mode}
			c, me, e := restoreSavedConnection(context.Background(), p, func(serviceName, id string) (string, error) {
				if serviceName != service || id != p.CredentialID() {
					t.Error("wrong credential identity")
				}
				return "saved-secret", nil
			})
			if e != nil || c == nil || me.ID != 7 || requests.Load() != 1 {
				t.Fatalf("automatic sign-in failed: %v", e)
			}
		})
	}
}
func TestRestoreDoesNotConnectWithoutRememberedCredentials(t *testing.T) {
	no := false
	p := settings.Profile{URL: "https://example.invalid", AutoLogin: &no}
	calls := 0
	getter := func(string, string) (string, error) { calls++; return "", errors.New("missing") }
	if _, _, e := restoreSavedConnection(context.Background(), p, getter); !errors.Is(e, errAutoLoginUnavailable) || calls != 0 {
		t.Fatal("opt-out read credentials")
	}
	p.AutoLogin = nil
	if _, _, e := restoreSavedConnection(context.Background(), p, getter); !errors.Is(e, errAutoLoginUnavailable) {
		t.Fatal("missing credential must return to manual login")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := calls
	if _, _, e := restoreSavedConnection(ctx, p, getter); !errors.Is(e, context.Canceled) || calls != before {
		t.Fatal("cancelled login was not stopped")
	}
}
func nextUI(t *testing.T, q chan func()) {
	t.Helper()
	select {
	case fn := <-q:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("UI result was not delivered")
	}
}
func TestStartupAutomaticallyEntersSavedAccount(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("APPDATA", d)
	t.Setenv("XDG_CONFIG_HOME", d)
	a := test.NewApp()
	defer a.Quit()
	var logins atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/current.json":
			logins.Add(1)
			fmt.Fprint(w, `{"user":{"id":7,"firstname":"Роман"}}`)
		case "/issues.json":
			fmt.Fprint(w, `{"issues":[],"total_count":0}`)
		case "/projects.json":
			fmt.Fprint(w, `{"projects":[],"total_count":0}`)
		case "/issue_statuses.json":
			fmt.Fprint(w, `{"issue_statuses":[]}`)
		case "/trackers.json":
			fmt.Fprint(w, `{"trackers":[]}`)
		case "/enumerations/issue_priorities.json":
			fmt.Fprint(w, `{"issue_priorities":[]}`)
		case "/enumerations/time_entry_activities.json":
			fmt.Fprint(w, `{"time_entry_activities":[]}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	u := newUI(a, false)
	defer u.stopWork()
	q := make(chan func(), 20)
	u.post = func(fn func()) { q <- fn }
	u.secretReader = func(string, string) (string, error) { return "saved-key", nil }
	u.cfg.Profiles = []settings.Profile{{URL: server.URL, Name: "Рабочий сервер", Auth: "api_key"}}
	u.restoreSession()
	if u.authLoading == nil || !u.authLoading.running {
		t.Fatal("startup animation missing")
	}
	nextUI(t, q)
	if u.client == nil || u.me.ID != 7 || u.list == nil || u.authLoading.running || logins.Load() != 1 {
		t.Fatal("startup did not enter the saved account automatically")
	}
}

func TestRejectedSavedCredentialsReturnAnErrorWithoutLeakingSecret(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer s.Close()
	c, _, e := restoreSavedConnection(context.Background(), settings.Profile{URL: s.URL}, func(string, string) (string, error) { return "private-test-secret", nil })
	if e == nil || c != nil {
		t.Fatal("invalid credentials accepted")
	}
	if strings.Contains(e.Error(), "private-test-secret") {
		t.Fatal("credential was included in error message")
	}
}
func TestCancellingAutomaticLoginCannotOpenOldAccount(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("APPDATA", d)
	t.Setenv("XDG_CONFIG_HOME", d)
	a := test.NewApp()
	defer a.Quit()
	started := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer s.Close()
	u := newUI(a, false)
	defer u.stopWork()
	q := make(chan func(), 20)
	u.post = func(fn func()) { q <- fn }
	u.secretReader = func(string, string) (string, error) { return "test", nil }
	u.cfg.Profiles = []settings.Profile{{URL: s.URL, Auth: "api_key"}}
	u.restoreSession()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("automatic login did not start")
	}
	u.login()
	nextUI(t, q)
	nextUI(t, q)
	if u.client != nil || u.authLoading.running || u.appearanceSelect == nil {
		t.Fatal("cancelled automatic login replaced the manual form")
	}
}
