// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/redmine"
	"github.com/rkislov/redmineapp/internal/settings"
	"github.com/zalando/go-keyring"
	"time"
)

var errAutoLoginUnavailable = errors.New("автоматический вход недоступен")

func restoreSavedConnection(ctx context.Context, p settings.Profile, getSecret func(string, string) (string, error)) (*redmine.Client, redmine.User, error) {
	empty := redmine.User{}
	if p.AutoLogin != nil && !*p.AutoLogin {
		return nil, empty, errAutoLoginUnavailable
	}
	if e := ctx.Err(); e != nil {
		return nil, empty, e
	}
	secret, e := getSecret(service, p.CredentialID())
	if e != nil || secret == "" {
		return nil, empty, errAutoLoginUnavailable
	}
	if e = ctx.Err(); e != nil {
		return nil, empty, e
	}
	var c *redmine.Client
	switch p.Auth {
	case "basic":
		c, e = redmine.NewBasic(p.URL, p.Username, secret)
	case "", "api_key":
		c, e = redmine.New(p.URL, secret)
	default:
		e = fmt.Errorf("неизвестный способ входа")
	}
	if e != nil {
		return nil, empty, e
	}
	me, e := c.Me(ctx)
	if e != nil {
		return nil, empty, e
	}
	return c, me, nil
}
func (u *UI) cancelAuthentication() {
	u.authGeneration++
	if u.authCancel != nil {
		u.authCancel()
	}
	if u.authLoading != nil {
		u.authLoading.Stop()
	}
}
func (u *UI) startSession(c *redmine.Client, me redmine.User) {
	u.cancelAuthentication()
	u.stopPaneRequests()
	u.client = c
	u.me = me
	u.mode = "Мои"
	u.status = "open"
	u.search = ""
	u.project = 0
	u.offset = 0
	u.selected = 0
	u.issues = nil
	u.total = 0
	u.shell()
	u.loadMetadata()
	u.refresh()
}
func (u *UI) restoreSession() {
	if u.cfg.Active < 0 || u.cfg.Active >= len(u.cfg.Profiles) {
		u.login()
		return
	}
	profile := u.cfg.Profiles[u.cfg.Active]
	if profile.AutoLogin != nil && !*profile.AutoLogin {
		u.login()
		return
	}
	u.cancelAuthentication()
	g := u.authGeneration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	u.authCancel = cancel
	u.authLoading = newLoadingPanel("Подключаемся к " + profile.Name + "…")
	u.setContent(container.NewBorder(nil, container.NewCenter(widget.NewButton("Выбрать подключение", func() { u.login() })), nil, nil, u.authLoading.view))
	u.authLoading.Start("Подключаемся к " + profile.Name + "…")
	go func() {
		defer cancel()
		c, me, e := restoreSavedConnection(ctx, profile, u.getSecret)
		u.enqueue(func() {
			if g != u.authGeneration {
				return
			}
			u.authLoading.Stop()
			if e != nil {
				if errors.Is(e, errAutoLoginUnavailable) {
					u.login()
				} else {
					u.login("Автоматический вход не выполнен: " + e.Error())
				}
				return
			}
			u.startSession(c, me)
		})
	}()
}

func (u *UI) getSecret(service, key string) (string, error) {
	if u.secretReader != nil {
		return u.secretReader(service, key)
	}
	return keyring.Get(service, key)
}
func (u *UI) enqueue(fn func()) {
	if u.post != nil {
		u.post(fn)
		return
	}
	fyne.Do(fn)
}
