package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestServeCachedFontReturnsNotModified(t *testing.T) {
	dir := t.TempDir()
	fontPath := filepath.Join(dir, "font.ttf")
	if err := os.WriteFile(fontPath, []byte("font-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	modTime := time.Unix(12345, 0).UTC()
	if err := os.Chtimes(fontPath, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/fonts/DroidSansFallback.ttf", nil)
	req.Header.Set("If-None-Match", `W/"10-12345"`)
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{App: pocketbase.New(), Event: router.Event{Request: req, Response: rec}}

	if err := serveCachedFont(e, fontPath); err != nil {
		t.Fatalf("serveCachedFont: %v", err)
	}

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNotModified)
	}
	if got := res.Header.Get("Cache-Control"); got != fontCacheControl {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := res.Header.Get("ETag"); got == "" || !strings.Contains(got, "12345") {
		t.Fatalf("ETag = %q, want weak etag containing modtime", got)
	}
}
