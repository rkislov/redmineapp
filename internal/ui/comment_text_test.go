// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func TestPreformattedComments(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []textPart
	}{
		{"Обычный <b>текст</b>", []textPart{{"Обычный <b>текст</b>", false}}},
		{"До<pre>  x\n\ty &lt; z &amp; a</pre>После", []textPart{{"До", false}, {"  x\n\ty < z & a", true}, {"После", false}}},
		{`<PRE class="code"><code class="sh">echo &quot;test&quot;</code></PRE>`, []textPart{{`echo "test"`, true}}},
		{"<pre><server>\n  value\n</server></pre>", []textPart{{"<server>\n  value\n</server>", true}}},
		{"<pre>незакрытый\n  блок", []textPart{{"незакрытый\n  блок", true}}},
		{"&lt;pre&gt;обычный текст", []textPart{{"&lt;pre&gt;обычный текст", false}}},
	} {
		got := preformattedParts(tc.text)
		if len(got) != len(tc.want) {
			t.Fatalf("%q: %+v", tc.text, got)
		}
		for i, p := range got {
			if p != tc.want[i] {
				t.Fatalf("%q: %+v want %+v", tc.text, got, tc.want)
			}
		}
	}
}

func TestFormattedTextCopiesSelectedText(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(appTheme{})
	win := app.NewWindow("Copy")
	defer win.Close()
	ui := &UI{win: win}

	for _, sample := range []struct{ input, want string }{
		{"текст дальше", "текст"},
		{"<pre>текст дальше</pre>", "текст"},
	} {
		content := ui.formattedText(sample.input).(*container.ThemeOverride)
		win.SetContent(container.NewVBox(content))
		win.Resize(fyne.NewSize(640, 360))

		box := content.Content.(*fyne.Container)
		var textLabel *widget.Label
		if sample.input == "текст дальше" {
			textLabel = box.Objects[0].(*widget.Label)
		} else {
			padded := box.Objects[0].(*fyne.Container).Objects[1].(*fyne.Container)
			border := padded.Objects[0].(*fyne.Container)
			textLabel = border.Objects[0].(*container.Scroll).Content.(*widget.Label)
		}
		position := app.Driver().AbsolutePositionForObject(textLabel)
		padding := theme.InnerPadding()
		word := fyne.MeasureText(sample.want, theme.TextSize(), textLabel.TextStyle)
		app.Clipboard().SetContent("")
		test.Drag(win.Canvas(), position.Add(fyne.NewPos(padding+word.Width, padding+word.Height/2)), word.Width, 0)
		focused, ok := win.Canvas().Focused().(fyne.Shortcutable)
		if !ok {
			t.Fatalf("%q did not focus a copy target", sample.input)
		}
		focused.TypedShortcut(&fyne.ShortcutCopy{Clipboard: app.Clipboard()})
		if got := app.Clipboard().Content(); got != sample.want {
			t.Fatalf("copy from %q = %q, want %q", sample.input, got, sample.want)
		}
	}
}
