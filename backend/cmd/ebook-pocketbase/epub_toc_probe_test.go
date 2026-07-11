package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gen2brain/go-fitz"
)

func TestDumpEPUBTOCForLocalStorage(t *testing.T) {
	inputs := strings.TrimSpace(os.Getenv("EPUB_TOC_INPUTS"))
	if inputs == "" {
		t.Skip("EPUB_TOC_INPUTS not set")
	}
	outDir := os.Getenv("EPUB_TOC_OUT_DIR")
	if outDir == "" {
		outDir = os.TempDir()
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(inputs, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("bad input %q", line)
		}
		id, filePath := parts[0], parts[1]
		bookBytes, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("read %s: %v", filePath, err)
		}
		bookBytes = injectEPUBRenderFont(bookBytes)
		tmpFile, err := os.CreateTemp("", "epub-toc-probe-*.epub")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tmpFile.Write(bookBytes); err != nil {
			t.Fatal(err)
		}
		if err := tmpFile.Close(); err != nil {
			t.Fatal(err)
		}
		func() {
			defer os.Remove(tmpFile.Name())
			fitzMu.Lock()
			defer fitzMu.Unlock()
			doc, err := fitz.New(tmpFile.Name())
			if err != nil {
				t.Fatalf("fitz %s: %v", id, err)
			}
			defer doc.Close()
			toc := resolveEPUBTOCPages(doc, parseEPUBTOC(bookBytes))
			data, err := json.Marshal(toc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(outDir, id+".json"), data, 0644); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s toc=%d", id, len(toc))
		}()
	}
}
