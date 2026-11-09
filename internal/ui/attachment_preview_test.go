// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/redmine"
)

func TestImagePreviewAndCloseDuringDownload(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(appTheme{Mode: "light"})
	img := image.NewRGBA(image.Rect(0, 0, 80, 60))
	img.Set(0, 0, color.White)
	var data bytes.Buffer
	png.Encode(&data, img)
	started := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Redmine-API-Key") != "secret" {
			t.Error("download missing authentication")
		}
		if r.URL.Path == "/attachments/download/2/file" {
			close(started)
			<-r.Context().Done()
			return
		}
		w.Write(data.Bytes())
	}))
	defer s.Close()
	c, _ := redmine.New(s.URL, "secret")
	q := make(chan func(), 100)
	u := &UI{app: a, win: a.NewWindow("test"), client: c, post: func(fn func()) { q <- fn }}
	defer u.stopWork()
	p := u.previewAttachment(7, redmine.Attachment{Filename: "picture.png", ContentURL: s.URL + "/attachments/download/1/file"})
	for p.image.Image == nil {
		nextUI(t, q)
	}
	if p.loading.running || len(u.previewWindows) != 1 {
		t.Fatal("preview did not finish")
	}
	if out := os.Getenv("REDMINE_PREVIEW_SCREENSHOT"); out != "" {
		f, e := os.Create(out)
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, p.win.Canvas().Capture())
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	p.zoom.SetSelected("200%")
	if p.image.MinSize().Width != 160 {
		t.Fatal("image zoom did not apply")
	}
	p.win.Close()
	if len(u.previewWindows) != 0 {
		t.Fatal("window not released")
	}
	p = u.previewAttachment(7, redmine.Attachment{Filename: "slow.png", ContentURL: s.URL + "/attachments/download/2/file"})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}
	p.win.Close()
	nextUI(t, q)
	if p.loading.running || len(u.previewWindows) != 0 {
		t.Fatal("closed window kept loading")
	}
}
func TestCommentAndFilesSentFromPreview(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(appTheme{Mode: "light"})
	q := make(chan func(), 100)
	c := redmine.NewDemo()
	u := &UI{app: a, win: a.NewWindow("test"), client: c, post: func(fn func()) { q <- fn }}
	win := a.NewWindow("preview")
	win.Resize(fyne.NewSize(500, 600))
	view, p := u.newPreviewComment(win, 246614)
	win.SetContent(view)
	var send *widget.Button
	var entry *widget.Entry
	for _, o := range test.LaidOutObjects(view) {
		switch w := o.(type) {
		case *widget.Button:
			if w.Text == "Отправить" {
				send = w
			}
		case *widget.Entry:
			entry = w
		}
	}
	if send == nil || entry == nil {
		t.Fatal("composer controls missing")
	}
	path := filepath.Join(t.TempDir(), "note.txt")
	os.WriteFile(path, []byte("attachment"), 0600)
	p.paths = []string{path}
	p.refresh()
	entry.SetText("Комментарий из предпросмотра")
	// Avoid refreshing panes: this standalone test has no task shell.
	old := u.client
	test.Tap(send)
	for p.busy {
		// A successful save refreshes the task list, which requires the shell.
		if u.list == nil {
			u.mode = "Мои"
			u.status = "open"
			u.shell()
		}
		nextUI(t, q)
	}
	// Consume the list refresh before the test app shuts down.
	if u.listLoading.running {
		nextUI(t, q)
	}
	issue, e := old.Issue(context.Background(), 246614)
	if e != nil || len(issue.Attachments) != 1 || issue.Journals[len(issue.Journals)-1].Notes != "Комментарий из предпросмотра" {
		t.Fatalf("comment/files missing: %v", e)
	}
	if entry.Text != "" || len(p.paths) != 0 {
		t.Fatal("successful draft not cleared")
	}
	u.stopWork()
}
