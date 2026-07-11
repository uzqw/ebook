package main

import (
	"archive/zip"
	"bytes"
	"testing"
)

func testEPUBZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, contents := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func TestParseEPUBTOCFallsBackToNCX(t *testing.T) {
	book := testEPUBZip(t, map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="c1" href="Text/chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="Text/chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="c1"/>
    <itemref idref="c2"/>
  </spine>
</package>`,
		"OEBPS/toc.ncx": `<?xml version="1.0"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/">
  <navMap>
    <navPoint id="chapter-1" playOrder="1">
      <navLabel><text>Chapter One</text></navLabel>
      <content src="Text/chapter1.xhtml"/>
      <navPoint id="section-1-1" playOrder="2">
        <navLabel><text>Section 1.1</text></navLabel>
        <content src="Text/chapter1.xhtml#section-1-1"/>
      </navPoint>
    </navPoint>
    <navPoint id="chapter-2" playOrder="3">
      <navLabel><text>Chapter Two</text></navLabel>
      <content src="Text/chapter2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`,
		"OEBPS/Text/chapter1.xhtml": "<html><body>one</body></html>",
		"OEBPS/Text/chapter2.xhtml": "<html><body>two</body></html>",
	})

	toc := parseEPUBTOC(book)
	if len(toc) != 3 {
		t.Fatalf("toc len = %d, want 3: %#v", len(toc), toc)
	}
	assertTOCItem(t, toc[0], "Chapter One", 1, 1)
	assertTOCItem(t, toc[1], "Section 1.1", 1, 2)
	assertTOCItem(t, toc[2], "Chapter Two", 2, 1)
}

func assertTOCItem(t *testing.T, item tocItem, title string, page, level int) {
	t.Helper()
	if item.Title != title || item.Page != page || item.Level != level {
		t.Fatalf("toc item = %#v, want title=%q page=%d level=%d", item, title, page, level)
	}
}
