// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/preview"
	"github.com/rkislov/redmineapp/internal/redmine"
)

func canPreview(a redmine.Attachment) bool {
	switch strings.ToLower(filepath.Ext(a.Filename)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff":
		return true
	}
	return strings.EqualFold(a.ContentType, "application/pdf") || strings.HasPrefix(strings.ToLower(a.ContentType), "image/")
}
func isPDF(a redmine.Attachment) bool {
	return strings.EqualFold(filepath.Ext(a.Filename), ".pdf") || strings.EqualFold(a.ContentType, "application/pdf")
}

type attachmentPreview struct {
	downloadCancel context.CancelFunc
	u              *UI
	client         *redmine.Client
	win            fyne.Window
	ctx            context.Context
	cancel         context.CancelFunc
	doc            *preview.PDF
	view           *fyne.Container
	toolbar        *fyne.Container
	loading        *loadingPanel
	image          *canvas.Image
	scroll         *container.Scroll
	previous, next *widget.Button
	pageLabel      *widget.Label
	zoom           *widget.Select
	page           int
	closed, busy   bool
	picker         *attachmentPicker
}

func (u *UI) previewAttachment(issueID int, a redmine.Attachment) *attachmentPreview {
	if u.client == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &attachmentPreview{u: u, client: u.client, win: u.app.NewWindow(a.Filename + " — redmineapp"), ctx: ctx, cancel: cancel, loading: newLoadingPanel("Загружаем вложение…")}
	if u.previewWindows == nil {
		u.previewWindows = map[fyne.Window]context.CancelFunc{}
	}
	u.previewWindows[p.win] = cancel
	p.view = container.New(previewLayout{p}, p.loading.view)
	p.image = canvas.NewImageFromImage(nil)
	p.image.FillMode = canvas.ImageFillContain
	p.scroll = container.NewScroll(container.NewCenter(p.image))
	p.previous = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { p.render(p.page - 1) })
	p.next = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { p.render(p.page + 1) })
	p.pageLabel = widget.NewLabel("1 / …")
	if !isPDF(a) {
		p.pageLabel.SetText("Изображение")
		p.previous.Hide()
		p.next.Hide()
	}
	p.previous.Disable()
	p.next.Disable()
	p.zoom = widget.NewSelect([]string{"По ширине", "50%", "75%", "100%", "150%", "200%"}, func(string) { p.resizeImage() })
	p.zoom.SetSelected("По ширине")
	toolbar := container.NewBorder(nil, nil, container.NewHBox(p.previous, p.pageLabel, p.next), widget.NewButtonWithIcon("Скачать", theme.DownloadIcon(), func() { p.downloadCancel = u.downloadAttachmentAt(p.win, a) }), p.zoom)
	p.toolbar = toolbar
	composer, picker := u.newPreviewComment(p.win, issueID)
	p.picker = picker
	split := container.NewHSplit(container.NewBorder(toolbar, nil, nil, nil, p.view), newPanel(container.NewVScroll(composer)))
	split.Offset = 0.68
	p.win.SetContent(container.NewStack(newGlassBackdrop(), split))
	p.win.Resize(fyne.NewSize(1180, 800))
	p.win.SetOnClosed(func() {
		p.closed = true
		cancel()
		p.loading.Stop()
		delete(u.previewWindows, p.win)
		if p.downloadCancel != nil {
			p.downloadCancel()
		}
		if p.picker.cancel != nil {
			p.picker.cancel()
		}
		if p.doc != nil {
			go p.doc.Close()
		}
	})
	p.loading.Start("Загружаем вложение…")
	p.win.Show()
	go p.load(a)
	return p
}
func (p *attachmentPreview) load(a redmine.Attachment) {
	if a.Size > preview.MaxFileSize {
		p.fail(fmt.Errorf("для предпросмотра доступны файлы до 50 МБ; используйте «Скачать»"))
		return
	}
	f, e := os.CreateTemp("", "redmineapp-preview-*")
	if e != nil {
		p.fail(e)
		return
	}
	defer os.Remove(f.Name())
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Minute)
	defer cancel()
	last := time.Time{}
	e = p.client.Download(ctx, a, &preview.LimitedWriter{Writer: f, Remaining: preview.MaxFileSize}, func(n, total int64) {
		if time.Since(last) < 150*time.Millisecond {
			return
		}
		last = time.Now()
		p.u.enqueue(func() {
			if !p.closed {
				p.loading.caption.SetText("Загружаем " + fileSize(n) + "…")
			}
		})
	})
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		p.fail(e)
		return
	}
	if p.ctx.Err() != nil {
		return
	}
	p.u.enqueue(func() {
		if !p.closed {
			p.loading.caption.SetText("Подготавливаем предпросмотр…")
		}
	})
	if isPDF(a) {
		doc, e := preview.OpenPDF(p.ctx, f.Name())
		if e != nil {
			p.fail(e)
			return
		}
		p.u.enqueue(func() {
			if p.closed {
				go doc.Close()
				return
			}
			p.doc = doc
			p.render(0)
		})
	} else {
		img, e := preview.Image(f.Name())
		if e != nil {
			p.fail(e)
			return
		}
		p.u.enqueue(func() {
			if p.closed {
				return
			}
			p.pageLabel.SetText("Изображение")
			p.toolbar.Refresh()
			p.showImage(img)
		})
	}
}
func (p *attachmentPreview) fail(e error) {
	p.u.enqueue(func() {
		if p.closed {
			return
		}
		p.busy = false
		p.loading.Message(e.Error())
		p.view.Objects = []fyne.CanvasObject{p.loading.view}
		p.view.Refresh()
		p.updatePages()
	})
}
func (p *attachmentPreview) render(page int) {
	if p.closed || p.busy || p.doc == nil || page < 0 || page >= p.doc.Pages {
		return
	}
	p.busy = true
	p.page = page
	p.updatePages()
	p.view.Objects = []fyne.CanvasObject{p.loading.view}
	p.loading.Start(fmt.Sprintf("Открываем страницу %d…", page+1))
	p.view.Refresh()
	go func() {
		img, e := p.doc.Render(p.ctx, page)
		if e != nil {
			p.fail(e)
			return
		}
		p.u.enqueue(func() {
			if p.closed {
				return
			}
			p.busy = false
			p.showImage(img)
			p.updatePages()
		})
	}()
}
func (p *attachmentPreview) updatePages() {
	p.previous.Disable()
	p.next.Disable()
	if p.doc == nil {
		return
	}
	p.pageLabel.SetText(fmt.Sprintf("%d / %d", p.page+1, p.doc.Pages))
	p.toolbar.Refresh()
	if !p.busy {
		if p.page > 0 {
			p.previous.Enable()
		}
		if p.page+1 < p.doc.Pages {
			p.next.Enable()
		}
	}
}
func (p *attachmentPreview) showImage(img image.Image) {
	p.loading.Stop()
	p.image.Image = img
	p.image.Refresh()
	p.resizeImage()
	p.scroll.Offset = fyne.NewPos(0, 0)
	p.scroll.Refresh()
	p.view.Objects = []fyne.CanvasObject{p.scroll}
	p.view.Refresh()
	// The scroll has its final size only after it has been installed in the view.
	p.resizeImage()
}
func (p *attachmentPreview) resizeImage() {
	if p.image.Image == nil {
		return
	}
	bounds := p.image.Image.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	if p.zoom.Selected == "По ширине" {
		width := p.view.Size().Width - 32
		if width < 200 {
			width = 700
		}
		h = h * width / w
		w = width
	} else {
		scale := map[string]float32{"50%": 0.5, "75%": 0.75, "100%": 1, "150%": 1.5, "200%": 2}[p.zoom.Selected]
		w *= scale
		h *= scale
	}
	p.image.SetMinSize(fyne.NewSize(w, h))
	p.scroll.Refresh()
}

// The same task can be discussed without leaving the attachment preview.
func (u *UI) newPreviewComment(win fyne.Window, issueID int) (fyne.CanvasObject, *attachmentPicker) {
	c := u.client
	entry := widget.NewMultiLineEntry()
	entry.SetMinRowsVisible(6)
	entry.SetPlaceHolder("Комментарий к задаче…")
	files := u.newAttachmentPicker()
	files.win = win
	status := label(fmt.Sprintf("Задача #%d", issueID))
	send := widget.NewButtonWithIcon("Отправить", theme.MailSendIcon(), nil)
	send.Importance = widget.HighImportance
	send.OnTapped = func() {
		if files.busy || u.client != c {
			return
		}
		text := entry.Text
		if strings.TrimSpace(text) == "" && len(files.paths) == 0 {
			dialog.ShowError(fmt.Errorf("введите комментарий или прикрепите файл"), win)
			return
		}
		files.busy = true
		files.refresh()
		entry.Disable()
		send.Disable()
		status.SetText("Отправляем…")
		files.cancel = u.runTransferAt(win, "Отправка комментария", func(ctx context.Context, report func(string, int64, int64)) error {
			uploads, e := files.upload(ctx, c, report)
			if e != nil {
				return e
			}
			if e = ctx.Err(); e != nil {
				return e
			}
			f := map[string]any{"notes": text}
			if len(uploads) > 0 {
				f["uploads"] = uploads
			}
			return c.Update(ctx, issueID, f)
		}, func(e error) {
			files.cancel = nil
			files.busy = false
			files.refresh()
			entry.Enable()
			send.Enable()
			if e != nil {
				status.SetText("Черновик сохранён в этом окне")
				return
			}
			entry.SetText("")
			files.paths = nil
			files.refresh()
			status.SetText("Комментарий отправлен")
			if u.client == c {
				if u.selected == issueID {
					u.openIssue(issueID)
				}
				u.refresh()
			}
		})
	}
	return container.NewVBox(widget.NewLabelWithStyle("Комментарий", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), status, entry, files.view, send), files
}

// Keep fit-to-width responsive without letting a zoomed image resize the window.
type previewLayout struct{ p *attachmentPreview }

func (l previewLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(240, 240) }
func (l previewLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	p := l.p
	if p.image != nil && p.image.Image != nil && p.zoom != nil && p.zoom.Selected == "По ширине" {
		b := p.image.Image.Bounds()
		width := max(float32(1), size.Width-32)
		p.image.SetMinSize(fyne.NewSize(width, float32(b.Dy())*width/float32(b.Dx())))
	}
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}
