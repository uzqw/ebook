package main

import (
	"testing"
	"time"
)

// TestFitzBookCache pins the render-cache contract: hit on same
// record+size+mtime, miss after the file's size or mtime changes (replaced
// file must not serve stale renders), and bounded capacity.
func TestFitzBookCache(t *testing.T) {
	fitzBookCache.Lock()
	fitzBookCache.entries = nil
	fitzBookCache.Unlock()

	mtime := time.Now()
	fitzBookCacheStore("b1:f.epub", 100, mtime, []byte("v1"))

	if got := fitzBookCacheLookup("b1:f.epub", 100, mtime); string(got) != "v1" {
		t.Fatalf("expected cache hit, got %q", got)
	}
	if got := fitzBookCacheLookup("b1:f.epub", 200, mtime); got != nil {
		t.Errorf("size change must invalidate, got %q", got)
	}
	if got := fitzBookCacheLookup("b1:f.epub", 100, mtime.Add(time.Second)); got != nil {
		t.Errorf("mtime change must invalidate, got %q", got)
	}
	if got := fitzBookCacheLookup("b2:f.epub", 100, mtime); got != nil {
		t.Errorf("different record must miss, got %q", got)
	}

	// Bound: more stores than capacity evict the oldest.
	for i := 0; i < fitzBookCacheSize+2; i++ {
		fitzBookCacheStore(string(rune('a'+i))+":f", 1, mtime, []byte{byte(i)})
	}
	fitzBookCache.Lock()
	n := len(fitzBookCache.entries)
	fitzBookCache.Unlock()
	if n > fitzBookCacheSize {
		t.Errorf("cache holds %d entries, want <= %d", n, fitzBookCacheSize)
	}
}
