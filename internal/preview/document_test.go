// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package preview

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A two-page vector PDF, with no third-party fixture or external font.
func samplePDF() []byte {
	objects := []string{
		`<< /Type /Catalog /Pages 2 0 R >>`,
		`<< /Type /Pages /Kids [3 0 R 5 0 R] /Count 2 >>`,
		`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 100] /Resources << >> /Contents 4 0 R >>`,
		"<< /Length 27 >>\nstream\n1 0 0 rg 0 0 200 100 re f\nendstream",
		`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 200] /Resources << >> /Contents 6 0 R >>`,
		"<< /Length 27 >>\nstream\n0 0 1 rg 0 0 100 200 re f\nendstream",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, o := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, o := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return b.Bytes()
}
func TestPDFRenderingPagesAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "two-pages.pdf")
	os.WriteFile(path, samplePDF(), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, e := OpenPDF(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	if p.Pages != 2 {
		t.Fatalf("pages=%d", p.Pages)
	}
	first, e := p.Render(ctx, 0)
	if e != nil {
		t.Fatal(e)
	}
	second, e := p.Render(ctx, 1)
	if e != nil {
		t.Fatal(e)
	}
	for i, img := range []image.Image{first, second} {
		if img.Bounds().Dx() > 2200 || img.Bounds().Dy() > 2200 {
			t.Fatal("unbounded bitmap")
		}
		r, _, b, _ := img.At(img.Bounds().Dx()/2, img.Bounds().Dy()/2).RGBA()
		if i == 0 && (r < 60000 || b > 1000) || i == 1 && (b < 60000 || r > 1000) {
			t.Fatalf("wrong page colour: %d %d", r, b)
		}
	}
	// Rendering another page must not invalidate the first page's pixels.
	r, _, b, _ := first.At(30, 30).RGBA()
	if r < 60000 || b > 1000 {
		t.Fatal("WASM bitmap was not copied")
	}
	if _, e = p.Render(ctx, 2); e == nil {
		t.Fatal("out of range page accepted")
	}
	if out := os.Getenv("REDMINE_PDF_SCREENSHOT"); out != "" {
		f, e := os.Create(out)
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, second)
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	cancel()
	if _, e = p.Render(ctx, 0); e == nil {
		t.Fatal("cancelled render accepted")
	}
	p.Close()
	p.Close()
}
func TestInvalidPDFAndImageLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.pdf")
	os.WriteFile(path, []byte("not a pdf"), 0600)
	p, e := OpenPDF(context.Background(), path)
	if e == nil {
		p.Close()
		t.Fatal("invalid PDF accepted")
	}
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 60, 40))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	png.Encode(&b, img)
	path = filepath.Join(t.TempDir(), "image.png")
	os.WriteFile(path, b.Bytes(), 0600)
	decoded, e := Image(path)
	if e != nil || decoded.Bounds() != img.Bounds() {
		t.Fatalf("decode: %v", e)
	}
	oversized := b.Bytes()
	binary.BigEndian.PutUint32(oversized[16:20], 50000)
	binary.BigEndian.PutUint32(oversized[20:24], 50000)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	os.WriteFile(path, oversized, 0600)
	if _, e = Image(path); e == nil || !strings.Contains(e.Error(), "слишком большое") {
		t.Fatalf("image limit not enforced: %v", e)
	}
}
func TestDownloadLimitDoesNotTrustMetadata(t *testing.T) {
	var b bytes.Buffer
	w := &LimitedWriter{Writer: &b, Remaining: 5}
	if _, e := w.Write([]byte("1234")); e != nil {
		t.Fatal(e)
	}
	if _, e := w.Write([]byte("56")); e == nil || b.String() != "1234" {
		t.Fatal("limit exceeded")
	}
}
