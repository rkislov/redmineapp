// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/redmine"
)

func TestAttachmentDraftSurvivesSaveFailureAndCanRetry(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(appTheme{Mode: "light"})
	q := make(chan func(), 100)
	u := &UI{app: a, win: a.NewWindow("attachments"), client: redmine.NewDemo(), post: func(fn func()) { q <- fn }}
	u.win.Resize(fyne.NewSize(900, 720))
	u.win.SetContent(widget.NewLabel("Задача"))
	defer u.stopWork()
	p := u.newAttachmentPicker()
	for _, name := range []string{"one.txt", "two.txt"} {
		path := filepath.Join(t.TempDir(), name)
		os.WriteFile(path, []byte(name), 0600)
		p.paths = append(p.paths, path)
	}
	p.refresh()
	entry := widget.NewMultiLineEntry()
	entry.SetMinRowsVisible(6)
	entry.SetText("Черновик комментария")
	attempts := 0
	finished := false
	u.attachmentForm("Комментарий", "Отправить", []*widget.FormItem{widget.NewFormItem("Текст", entry)}, p, func() (map[string]any, error) { return map[string]any{"notes": entry.Text}, nil }, func(ctx context.Context, c *redmine.Client, f map[string]any) error {
		uploads := f["uploads"].([]redmine.Upload)
		if len(uploads) != 2 {
			t.Error("file queue lost")
		}
		attempts++
		if attempts == 1 {
			return fmt.Errorf("тестовая ошибка сервера")
		}
		return c.Update(ctx, 246614, f)
	}, func() { finished = true })
	var send *widget.Button
	for _, o := range test.LaidOutObjects(u.win.Canvas().Overlays().Top()) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Отправить" {
			send = b
			break
		}
	}
	if send == nil {
		t.Fatal("submit button missing")
	}
	if out := os.Getenv("REDMINE_ATTACHMENT_SCREENSHOT"); out != "" {
		f, e := os.Create(out)
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, u.win.Canvas().Capture())
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	test.Tap(send)
	for p.busy {
		nextUI(t, q)
	}
	if finished || send.Disabled() || entry.Text != "Черновик комментария" || len(p.paths) != 2 {
		t.Fatal("draft lost or cannot retry")
	}
	// Close the error overlay, leaving the original draft open.
	top := u.win.Canvas().Overlays().Top()
	top.Hide()
	test.Tap(send)
	for p.busy {
		nextUI(t, q)
	}
	if !finished {
		t.Fatal("retry did not complete")
	}
	v, e := u.client.Issue(context.Background(), 246614)
	if e != nil || len(v.Attachments) != 2 {
		t.Fatalf("files not attached after retry: %v", e)
	}
}
