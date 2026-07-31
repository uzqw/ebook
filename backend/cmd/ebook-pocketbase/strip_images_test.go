package main

import (
	"strings"
	"testing"
)

func TestStripEmbeddedImageData(t *testing.T) {
	html := `<html><body><p style="position:absolute;top:10pt">Hello</p>` +
		`<img style="position:absolute" src="data:image/png;base64,iVBORw0KGgo=">` +
		`<img src="data:image/jpeg;base64,/9j/4AAQ" style="left:0">` +
		`<p>World</p></body></html>`
	got := stripEmbeddedImageData(html)
	if len(got) >= len(html) {
		t.Fatalf("expected output shorter than input (%d >= %d)", len(got), len(html))
	}
	for _, want := range []string{
		`<p style="position:absolute;top:10pt">Hello</p>`,
		`<img style="position:absolute">`,
		`<img style="left:0">`,
		`<p>World</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "data:image") {
		t.Fatalf("data URLs remain: %q", got)
	}
}
