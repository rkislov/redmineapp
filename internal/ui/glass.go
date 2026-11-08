// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image"
	"image/color"
	"math"
)

func darkAppearance() bool {
	if a := fyne.CurrentApp(); a != nil {
		if t, ok := a.Settings().Theme().(appTheme); ok {
			return t.isDark(a.Settings().ThemeVariant())
		}
	}
	return false
}
func alpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// glassSurface layers actual translucency, a rounded optical sheen and two fine
// edges. The defocused app backdrop provides a consistent cross-platform material.
type glassSurface struct {
	widget.BaseWidget
	selected, hovered bool
	pointer           float32
	radius            float32
}

func newGlassSurface() *glassSurface {
	s := &glassSurface{pointer: .35, radius: 24}
	s.ExtendBaseWidget(s)
	return s
}
func (s *glassSurface) CreateRenderer() fyne.WidgetRenderer {
	r := &glassRenderer{s: s, base: canvas.NewRectangle(color.Transparent), rim: canvas.NewRectangle(color.Transparent), inner: canvas.NewRectangle(color.Transparent)}
	r.sheen = canvas.NewRaster(s.renderSheen)
	r.Refresh()
	return r
}

type glassRenderer struct {
	s                *glassSurface
	base, rim, inner *canvas.Rectangle
	sheen            *canvas.Raster
}

func (r *glassRenderer) MinSize() fyne.Size { return fyne.NewSize(0, 0) }
func (r *glassRenderer) Layout(size fyne.Size) {
	r.base.Resize(size)
	r.sheen.Resize(size)
	r.rim.Resize(size)
	r.inner.Move(fyne.NewPos(1, 1))
	r.inner.Resize(fyne.NewSize(max(0, size.Width-2), max(0, size.Height-2)))
}
func (r *glassRenderer) Refresh() {
	p := currentColors()
	dark := darkAppearance()
	r.base.CornerRadius = r.s.radius
	r.rim.CornerRadius = r.s.radius
	r.inner.CornerRadius = max(0, r.s.radius-1)
	r.base.FillColor = alpha(p.surface, 182)
	r.rim.StrokeWidth = 1
	r.rim.StrokeColor = color.NRGBA{255, 255, 255, 210}
	r.inner.StrokeWidth = .6
	r.inner.StrokeColor = color.NRGBA{109, 134, 155, 27}
	r.base.Shadow = canvas.Shadow{Color: color.NRGBA{35, 56, 80, 23}, BlurRadius: 19, Offset: fyne.NewPos(0, 6)}
	if dark {
		r.base.FillColor = alpha(p.surface, 188)
		r.rim.StrokeColor = color.NRGBA{211, 228, 245, 65}
		r.inner.StrokeColor = color.NRGBA{8, 14, 24, 115}
		r.base.Shadow.Color = color.NRGBA{0, 4, 12, 95}
	}
	if r.s.selected {
		r.base.FillColor = alpha(p.selected, 216)
		r.rim.StrokeColor = alpha(p.ring, 195)
		r.rim.StrokeWidth = 1.25
		r.base.Shadow.BlurRadius = 24
		r.base.Shadow.Color = alpha(p.accent, 32)
		if dark {
			r.base.Shadow.Color = color.NRGBA{0, 6, 20, 120}
		}
	}
	if r.s.hovered {
		r.base.Shadow.BlurRadius = 25
		r.base.Shadow.Offset = fyne.NewPos(0, 8)
		if !r.s.selected {
			r.rim.StrokeColor = color.NRGBA{255, 255, 255, 245}
			if dark {
				r.rim.StrokeColor = color.NRGBA{222, 237, 255, 108}
			}
		}
	}
	r.Layout(r.s.Size())
	r.base.Refresh()
	r.rim.Refresh()
	r.inner.Refresh()
	r.sheen.Refresh()
	canvas.Refresh(r.s)
}
func (r *glassRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.base, r.sheen, r.rim, r.inner}
}
func (r *glassRenderer) Destroy() {}
func (s *glassSurface) renderSheen(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if w <= 0 || h <= 0 {
		return img
	}
	scale := float64(w) / math.Max(1, float64(s.Size().Width))
	radius := math.Min(float64(s.radius)*scale, math.Min(float64(w), float64(h))/2)
	white := color.NRGBA{255, 255, 255, 255}
	if darkAppearance() {
		white = color.NRGBA{211, 229, 250, 255}
	}
	glint := make([]float64, w)
	center := float64(s.pointer)
	strength := 0.
	if s.hovered {
		strength = 28
	}
	for x := range glint {
		u := float64(x) / float64(w)
		glint[x] = strength * math.Exp(-math.Pow((u-center)/.22, 2))
	}
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h)
		base := 26*math.Pow(math.Max(0, 1-v), 2) + 3*math.Max(0, (v-.65)/.35)
		for x := 0; x < w; x++ {
			coverage := 1.
			dx := math.Max(radius-float64(x)-.5, float64(x)+.5-float64(w)+radius)
			dy := math.Max(radius-float64(y)-.5, float64(y)+.5-float64(h)+radius)
			if dx > 0 && dy > 0 {
				coverage = math.Max(0, math.Min(1, radius+.5-math.Hypot(dx, dy)))
			}
			i := img.PixOffset(x, y)
			img.Pix[i] = white.R
			img.Pix[i+1] = white.G
			img.Pix[i+2] = white.B
			img.Pix[i+3] = uint8(math.Min(255, (base+glint[x]*(1-v*.65))*coverage))
		}
	}
	return img
}

// Broad, pre-softened light fields stay behind the glass rather than introducing
// busy detail under text. The background is cached until resize or theme change.
type glassBackdrop struct{ widget.BaseWidget }

func newGlassBackdrop() *glassBackdrop { b := &glassBackdrop{}; b.ExtendBaseWidget(b); return b }
func (b *glassBackdrop) CreateRenderer() fyne.WidgetRenderer {
	r := &backdropRenderer{b: b, raster: canvas.NewRaster(renderBackdrop)}
	return r
}

type backdropRenderer struct {
	b      *glassBackdrop
	raster *canvas.Raster
}

func (r *backdropRenderer) MinSize() fyne.Size           { return fyne.NewSize(0, 0) }
func (r *backdropRenderer) Layout(s fyne.Size)           { r.raster.Resize(s) }
func (r *backdropRenderer) Refresh()                     { r.raster.Refresh(); canvas.Refresh(r.b) }
func (r *backdropRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.raster} }
func (r *backdropRenderer) Destroy()                     {}
func renderBackdrop(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if w <= 0 || h <= 0 {
		return img
	}
	dark := darkAppearance()
	base := currentColors().background
	colors := [][3]float64{{174, 196, 218}, {221, 210, 201}, {179, 206, 209}}
	if dark {
		colors = [][3]float64{{48, 69, 96}, {62, 52, 68}, {36, 67, 73}}
	}
	xFields := make([][3]float64, w)
	yFields := make([][3]float64, h)
	centers := [3][2]float64{{.08, .2}, {.9, .87}, {.82, .08}}
	for x := 0; x < w; x++ {
		u := float64(x) / float64(w)
		for k, c := range centers {
			xFields[x][k] = math.Exp(-math.Pow((u-c[0])/.58, 2))
		}
	}
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h)
		for k, c := range centers {
			yFields[y][k] = math.Exp(-math.Pow((v-c[1])/.63, 2))
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b := float64(base.R), float64(base.G), float64(base.B)
			for k, c := range colors {
				t := xFields[x][k] * yFields[y][k] * .34
				r += (c[0] - r) * t
				g += (c[1] - g) * t
				b += (c[2] - b) * t
			}
			i := img.PixOffset(x, y)
			img.Pix[i] = uint8(r)
			img.Pix[i+1] = uint8(g)
			img.Pix[i+2] = uint8(b)
			img.Pix[i+3] = 255
		}
	}
	return img
}
func (u *UI) setContent(content fyne.CanvasObject) {
	u.win.SetContent(container.NewStack(newGlassBackdrop(), content))
}

// Selected navigation/filter controls use a clear inset pill, keeping primary
// issue actions visibly distinct without painting the whole sidebar an accent.
type glassControlTheme struct{}

func (glassControlTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	p := currentColors()
	switch n {
	case theme.ColorNamePrimary:
		return alpha(p.surface, 225)
	case theme.ColorNameForegroundOnPrimary:
		return p.foreground
	case theme.ColorNameShadow:
		return alpha(p.shadow, 24)
	}
	return fyne.CurrentApp().Settings().Theme().Color(n, v)
}
func (glassControlTheme) Font(s fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(s)
}
func (glassControlTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(n)
}
func (glassControlTheme) Size(n fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(n)
}
