// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import "testing"

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
