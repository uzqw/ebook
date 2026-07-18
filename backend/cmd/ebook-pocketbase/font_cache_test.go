package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestApplyFontCacheHeaders(t *testing.T) {
	headers := http.Header{}

	applyFontCacheHeaders(headers)

	if got, want := headers.Get("Cache-Control"), fontCacheControl; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}
}

func TestDecorateReaderPageHTMLDoesNotRequestFont(t *testing.T) {
	html := decorateReaderPageHTML("<html><head></head><body><div id=\"page1\"></div></body></html>")

	if strings.Contains(html, "/api/fonts/") {
		t.Fatalf("decorated page should not request external font: %s", html)
	}
	if strings.Contains(html, "@font-face") {
		t.Fatalf("decorated page should leave font injection to the parent reader")
	}
}

func TestDecorateReaderPageHTMLInjectsSelectionStabilizer(t *testing.T) {
	html := decorateReaderPageHTML("<html><head></head><body><div id=\"page1\"></div></body></html>")

	for _, needle := range []string{
		"installSelectionStabilizer",
		"nearestTextRange",
		"caretRangeFromPoint",
		"expandLineHitTargets",
		"data-orig-top",
		"user-select: none",
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("decorated page missing selection stabilizer piece %q", needle)
		}
	}
}
