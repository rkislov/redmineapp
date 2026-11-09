// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package preview

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"sync"
	"time"

	pdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const MaxFileSize int64 = 50 << 20
const MaxImagePixels = 25_000_000

// LimitedWriter bounds downloads even if a server omits or lies about filesize.
type LimitedWriter struct {
	Writer    io.Writer
	Remaining int64
}

func (w *LimitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.Remaining {
		return 0, fmt.Errorf("для предпросмотра доступны файлы до 50 МБ; используйте «Скачать»")
	}
	n, e := w.Writer.Write(p)
	w.Remaining -= int64(n)
	return n, e
}
func Image(path string) (image.Image, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	config, _, e := image.DecodeConfig(f)
	if e != nil {
		return nil, fmt.Errorf("изображение повреждено или формат не поддерживается: %w", e)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > MaxImagePixels {
		return nil, fmt.Errorf("изображение слишком большое для предпросмотра (максимум 25 млн пикселей)")
	}
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return nil, e
	}
	img, _, e := image.Decode(f)
	return img, e
}

type PDF struct {
	mu       sync.Mutex
	pool     pdfium.Pool
	instance pdfium.Pdfium
	document references.FPDF_DOCUMENT
	Pages    int
	closed   bool
}

// OpenPDF mounts no host filesystem and does not execute document JavaScript.
// Cancellation terminates the WASM runtime, including an in-flight render.
func OpenPDF(ctx context.Context, path string) (*PDF, error) {
	data, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	if int64(len(data)) > MaxFileSize {
		return nil, fmt.Errorf("PDF превышает 50 МБ")
	}
	cfg := wazero.NewRuntimeConfig().WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling).WithMemoryLimitPages(8192).WithCloseOnContextDone(true)
	pool, e := webassembly.Init(webassembly.Config{Context: ctx, MinIdle: 0, MaxIdle: 1, MaxTotal: 1, FSConfig: wazero.NewFSConfig(), RuntimeConfig: cfg, Stdout: io.Discard, Stderr: io.Discard})
	if e != nil {
		return nil, fmt.Errorf("не удалось запустить просмотр PDF: %w", e)
	}
	p := &PDF{pool: pool}
	p.instance, e = pool.GetInstance(30 * time.Second)
	if e != nil {
		pool.Close()
		return nil, e
	}
	doc, e := p.instance.OpenDocument(&requests.OpenDocument{File: &data})
	if e != nil {
		p.Close()
		return nil, fmt.Errorf("не удалось открыть PDF (файл повреждён или защищён паролем): %w", e)
	}
	p.document = doc.Document
	count, e := p.instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if e != nil || count.PageCount < 1 {
		p.Close()
		return nil, fmt.Errorf("не удалось прочитать страницы PDF")
	}
	p.Pages = count.PageCount
	return p, nil
}
func (p *PDF) Render(ctx context.Context, page int) (image.Image, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, fmt.Errorf("документ закрыт")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if page < 0 || page >= p.Pages {
		return nil, fmt.Errorf("страница вне документа")
	}
	result, e := p.instance.RenderPageInPixels(&requests.RenderPageInPixels{Page: requests.Page{ByIndex: &requests.PageByIndex{Document: p.document, Index: page}}, Width: 2200, Height: 2200})
	if e != nil {
		return nil, fmt.Errorf("не удалось отобразить страницу: %w", e)
	}
	defer result.Cleanup()
	// WASM owns the source pixels; copy them before releasing the bitmap.
	src := result.Result.RenderedImage
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	return dst, nil
}
func (p *PDF) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	if p.instance != nil {
		if p.document != "" {
			_, _ = p.instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: p.document})
		}
		_ = p.instance.Close()
	}
	if p.pool != nil {
		_ = p.pool.Close()
	}
}
