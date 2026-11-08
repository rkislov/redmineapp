// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/redmine"
)

func fileSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d Б", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f КБ", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f МБ", float64(n)/(1024*1024))
}
func (u *UI) attachmentSection(v redmine.Issue) fyne.CanvasObject {
	rows := container.NewVBox()
	if len(v.Attachments) == 0 {
		rows.Add(label("Нет вложений"))
	}
	for _, a := range v.Attachments {
		name := label(a.Filename)
		name.TextStyle.Bold = true
		metadata := fileSize(a.Size)
		if a.Author.Name != "" {
			metadata += " · " + a.Author.Name
		}
		if a.Created != "" {
			metadata += " · " + formatDate(a.Created)
		}
		text := container.NewVBox(name, label(metadata))
		if a.Description != "" {
			text.Add(label(a.Description))
		}
		rows.Add(newPanel(container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("Скачать", theme.DownloadIcon(), func() { u.downloadAttachment(a) }), text)))
	}
	return container.NewVBox(widget.NewLabelWithStyle(fmt.Sprintf("Вложения (%d)", len(v.Attachments)), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), rows)
}

// Transfer UI is updated on the main thread, at most ten times a second.
func (u *UI) runTransfer(title string, work func(context.Context, func(string, int64, int64)) error, done func(error)) {
	c := u.client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	if u.transferCancels == nil {
		u.transferCancels = map[int]context.CancelFunc{}
	}
	u.transferID++
	id := u.transferID
	u.transferCancels[id] = cancel
	message := label("Подготовка…")
	bar := widget.NewProgressBar()
	d := dialog.NewCustom(title, "Отмена", container.NewVBox(message, bar), u.win)
	d.SetOnClosed(cancel)
	d.Resize(fyne.NewSize(480, 180))
	d.Show()
	go func() {
		var last time.Time
		report := func(name string, n, total int64) {
			if ctx.Err() != nil {
				return
			}
			now := time.Now()
			if now.Sub(last) < 100*time.Millisecond && n != total {
				return
			}
			last = now
			u.enqueue(func() {
				if ctx.Err() != nil || u.client != c {
					return
				}
				message.SetText(name + " · " + fileSize(n))
				if total > 0 {
					bar.SetValue(float64(n) / float64(total))
				}
			})
		}
		err := work(ctx, report)
		if err == nil {
			err = ctx.Err()
		}
		u.enqueue(func() {
			delete(u.transferCancels, id)
			if u.client != c {
				err = context.Canceled
			}
			d.Hide()
			cancel()
			done(err)
			if err != nil && u.client == c && !errors.Is(err, context.Canceled) {
				dialog.ShowError(err, u.win)
			}
		})
	}()
}
func (u *UI) downloadAttachment(a redmine.Attachment) {
	c := u.client
	var temp string
	u.runTransfer("Скачивание вложения", func(ctx context.Context, report func(string, int64, int64)) error {
		f, e := os.CreateTemp("", "redmineapp-attachment-*")
		if e != nil {
			return e
		}
		temp = f.Name()
		e = c.Download(ctx, a, f, func(n, total int64) { report(a.Filename, n, total) })
		closeErr := f.Close()
		if e != nil {
			return e
		}
		return closeErr
	}, func(err error) {
		if err != nil {
			if temp != "" {
				_ = os.Remove(temp)
			}
			return
		}
		d := dialog.NewFileSave(func(dst fyne.URIWriteCloser, e error) {
			if e != nil {
				_ = os.Remove(temp)
				dialog.ShowError(e, u.win)
				return
			}
			if dst == nil {
				_ = os.Remove(temp)
				return
			}
			go func() {
				defer os.Remove(temp)
				src, e := os.Open(temp)
				if e == nil {
					_, e = io.Copy(dst, src)
					_ = src.Close()
				}
				closeErr := dst.Close()
				if e == nil {
					e = closeErr
				}
				u.enqueue(func() {
					if u.client != c {
						return
					}
					if e != nil {
						dialog.ShowError(fmt.Errorf("не удалось сохранить файл: %w", e), u.win)
					}
				})
			}()
		}, u.win)
		// A server-provided filename must never select a directory on the computer.
		name := filepath.Base(strings.ReplaceAll(a.Filename, "\\", "/"))
		if name == "." || name == "/" || name == "" {
			name = "attachment"
		}
		d.SetFileName(name)
		d.Resize(fyne.NewSize(700, 480))
		d.Show()
	})
}

type attachmentPicker struct {
	u     *UI
	paths []string
	view  *fyne.Container
	rows  *fyne.Container
	add   *widget.Button
	busy  bool
}

func (u *UI) newAttachmentPicker() *attachmentPicker {
	p := &attachmentPicker{u: u, rows: container.NewVBox()}
	p.add = widget.NewButtonWithIcon("Добавить файл…", theme.MailAttachmentIcon(), p.choose)
	scroll := container.NewVScroll(p.rows)
	scroll.SetMinSize(fyne.NewSize(380, 100))
	p.view = container.NewVBox(p.add, scroll)
	p.refresh()
	return p
}
func (p *attachmentPicker) refresh() {
	p.rows.Objects = nil
	if len(p.paths) == 0 {
		p.rows.Add(label("Файлы не выбраны"))
	}
	for i, path := range p.paths {
		text := filepath.Base(path)
		if info, e := os.Stat(path); e == nil {
			text += " · " + fileSize(info.Size())
		}
		name := widget.NewLabel(text)
		name.Truncation = fyne.TextTruncateEllipsis
		remove := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			if p.busy {
				return
			}
			p.paths = append(p.paths[:i], p.paths[i+1:]...)
			p.refresh()
		})
		if p.busy {
			remove.Disable()
		}
		p.rows.Add(container.NewBorder(nil, nil, nil, remove, name))
	}
	if p.busy {
		p.add.Disable()
	} else {
		p.add.Enable()
	}
	p.rows.Refresh()
}
func (p *attachmentPicker) choose() {
	c := p.u.client
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, e error) {
		if e != nil {
			dialog.ShowError(e, p.u.win)
			return
		}
		if r == nil {
			return
		}
		defer r.Close()
		if p.busy || p.u.client != c {
			return
		}
		if r.URI().Scheme() != "file" {
			dialog.ShowError(fmt.Errorf("выберите локальный файл"), p.u.win)
			return
		}
		path := r.URI().Path()
		for _, v := range p.paths {
			if v == path {
				return
			}
		}
		info, e := os.Stat(path)
		if e != nil {
			dialog.ShowError(e, p.u.win)
			return
		}
		if !info.Mode().IsRegular() {
			dialog.ShowError(fmt.Errorf("выберите обычный файл"), p.u.win)
			return
		}
		p.paths = append(p.paths, path)
		p.refresh()
	}, p.u.win)
	d.Resize(fyne.NewSize(700, 480))
	d.Show()
}
func (p *attachmentPicker) upload(ctx context.Context, c *redmine.Client, report func(string, int64, int64)) ([]redmine.Upload, error) {
	uploads := make([]redmine.Upload, 0, len(p.paths))
	for i, path := range p.paths {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		name := fmt.Sprintf("%d/%d · %s", i+1, len(p.paths), filepath.Base(path))
		item, e := c.UploadFile(ctx, path, func(n, total int64) { report(name, n, total) })
		if e != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), e)
		}
		uploads = append(uploads, item)
	}
	return uploads, nil
}

// Keep the draft and file list available after a server validation failure.
func (u *UI) attachmentForm(title, submit string, items []*widget.FormItem, p *attachmentPicker, fields func() (map[string]any, error), save func(context.Context, *redmine.Client, map[string]any) error, done func()) {
	c := u.client
	form := widget.NewForm(items...)
	form.Append("Вложения", p.view)
	var d *dialog.CustomDialog
	cancel := widget.NewButton("Отмена", func() { d.Hide() })
	send := widget.NewButton(submit, nil)
	send.Importance = widget.HighImportance
	send.OnTapped = func() {
		if p.busy || c != u.client {
			return
		}
		f, e := fields()
		if e != nil {
			dialog.ShowError(e, u.win)
			return
		}
		p.busy = true
		p.refresh()
		send.Disable()
		cancel.Disable()
		u.runTransfer(submit, func(ctx context.Context, report func(string, int64, int64)) error {
			uploads, e := p.upload(ctx, c, report)
			if e != nil {
				return e
			}
			if e = ctx.Err(); e != nil {
				return e
			}
			if len(uploads) > 0 {
				f["uploads"] = uploads
			}
			report("Сохраняем задачу…", 0, 0)
			return save(ctx, c, f)
		}, func(e error) {
			p.busy = false
			p.refresh()
			send.Enable()
			cancel.Enable()
			if e == nil {
				d.Hide()
				done()
			}
		})
	}
	d = dialog.NewCustomWithoutButtons(title, container.NewBorder(nil, container.NewHBox(layout.NewSpacer(), cancel, send), nil, nil, container.NewVScroll(form)), u.win)
	d.Resize(fyne.NewSize(640, min(590, form.MinSize().Height+110)))
	d.Show()
}
