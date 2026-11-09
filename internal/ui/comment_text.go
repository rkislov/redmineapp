// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"html"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	htmlparser "golang.org/x/net/html"
)

type textPart struct {
	text string
	pre  bool
}

// Only pre/code wrappers are interpreted. All other markup stays literal.
func preformattedParts(text string) []textPart {
	tokenizer := htmlparser.NewTokenizer(strings.NewReader(text))
	var parts []textPart
	var pending strings.Builder
	inPre := false
	flush := func() {
		if pending.Len() > 0 {
			value := pending.String()
			if inPre {
				value = unwrapCode(value)
				value = html.UnescapeString(value)
			}
			parts = append(parts, textPart{value, inPre})
			pending.Reset()
		}
	}
	for {
		kind := tokenizer.Next()
		if kind == htmlparser.ErrorToken {
			if tokenizer.Err() != io.EOF {
				pending.Write(tokenizer.Raw())
			}
			flush()
			break
		}
		raw := string(tokenizer.Raw())
		if kind == htmlparser.StartTagToken || kind == htmlparser.EndTagToken {
			name, _ := tokenizer.TagName()
			if strings.EqualFold(string(name), "pre") {
				if kind == htmlparser.StartTagToken && !inPre {
					flush()
					inPre = true
					continue
				}
				if kind == htmlparser.EndTagToken && inPre {
					flush()
					inPre = false
					continue
				}
			}
		}
		pending.WriteString(raw)
	}
	return parts
}
func unwrapCode(text string) string {
	trimmed := strings.TrimSpace(text)
	z := htmlparser.NewTokenizer(strings.NewReader(trimmed))
	if z.Next() != htmlparser.StartTagToken {
		return text
	}
	name, _ := z.TagName()
	if string(name) != "code" {
		return text
	}
	opening := len(z.Raw())
	if !strings.HasSuffix(strings.ToLower(trimmed), "</code>") {
		return text
	}
	// Preserve whitespace outside the optional code wrapper as well.
	start := strings.Index(text, trimmed)
	return text[:start] + trimmed[opening:len(trimmed)-len("</code>")] + text[start+len(trimmed):]
}
func (u *UI) formattedText(text string) fyne.CanvasObject {
	parts := preformattedParts(text)
	box := container.NewVBox()
	for _, part := range parts {
		if !part.pre {
			if part.text != "" {
				box.Add(label(part.text))
			}
			continue
		}
		code := part.text
		block := widget.NewRichText(&widget.TextSegment{Text: code, Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}, ColorName: theme.ColorNameForeground}})
		block.Wrapping = fyne.TextWrapOff
		scroll := container.NewScroll(block)
		lines := strings.Count(code, "\n") + 1
		scroll.SetMinSize(fyne.NewSize(240, min(float32(320), max(float32(56), float32(lines)*22))))
		copy := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() { u.win.Clipboard().SetContent(code) })
		heading := container.NewBorder(nil, nil, widget.NewLabelWithStyle("Код", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), copy)
		background := canvas.NewRectangle(alpha(currentColors().accent, 12))
		background.CornerRadius = 12
		box.Add(container.NewStack(background, container.NewPadded(container.NewBorder(heading, nil, nil, nil, scroll))))
	}
	return box
}
