package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/go-fitz"
	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type pdfService struct{ pool pdfium.Pool }

const fontCacheControl = "public, max-age=31536000, immutable"

// MuPDF/go-fitz is backed by native code and is not safe for concurrent EPUB/MOBI
// parse/render operations in this app. Serialize all fitz entry points so a
// reader image request cannot race the background parser and crash the process.
var fitzMu sync.Mutex

type tocItem struct {
	Title    string    `json:"title"`
	Page     int       `json:"page"`
	Level    int       `json:"level"`
	Children []tocItem `json:"children,omitempty"`
}

func flattenBookmarks(bookmarks []responses.GetBookmarksBookmark, level int) []tocItem {
	items := make([]tocItem, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		page := 1
		if bookmark.DestInfo != nil && bookmark.DestInfo.PageIndex >= 0 {
			page = bookmark.DestInfo.PageIndex + 1
		} else if bookmark.ActionInfo != nil && bookmark.ActionInfo.DestInfo != nil && bookmark.ActionInfo.DestInfo.PageIndex >= 0 {
			page = bookmark.ActionInfo.DestInfo.PageIndex + 1
		}
		items = append(items, tocItem{
			Title:    bookmark.Title,
			Page:     page,
			Level:    level,
			Children: flattenBookmarks(bookmark.Children, level+1),
		})
	}
	return items
}

func renderDPI() int {
	value := os.Getenv("PDF_RENDER_DPI")
	if value == "" {
		return 220
	}
	dpi, err := strconv.Atoi(value)
	if err != nil || dpi < 72 {
		return 220
	}
	if dpi > 360 {
		return 360
	}
	return dpi
}

func newPDFService() (*pdfService, error) {
	pool, err := webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 2, ReuseWorkers: true})
	if err != nil {
		return nil, err
	}
	return &pdfService{pool: pool}, nil
}

func (s *pdfService) close() {
	if s != nil && s.pool != nil {
		_ = s.pool.Close()
	}
}

func (s *pdfService) withDocument(pdfBytes []byte, fn func(pdfium.Pdfium, *requests.OpenDocument, int) error) error {
	instance, err := s.pool.GetInstance(30 * time.Second)
	if err != nil {
		return err
	}
	defer instance.Close()
	doc, err := instance.OpenDocument(&requests.OpenDocument{File: &pdfBytes})
	if err != nil {
		return err
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	count, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return err
	}
	return fn(instance, &requests.OpenDocument{File: &pdfBytes}, count.PageCount)
}

func recordPDFBytes(app core.App, record *core.Record) ([]byte, error) {
	filename := record.GetString("file")
	if filename == "" {
		return nil, fmt.Errorf("missing pdf file")
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()
	reader, err := fsys.GetReader(path.Join(record.BaseFilesPath(), filename))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func zipFileBytes(reader *zip.Reader, name string) ([]byte, bool) {
	name = path.Clean(strings.TrimPrefix(name, "/"))
	for _, file := range reader.File {
		if path.Clean(file.Name) != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, false
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		return data, err == nil
	}
	return nil, false
}

func attrValue(start xml.StartElement, localName string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == localName {
			return attr.Value
		}
	}
	return ""
}

func resolveEPUBPath(baseFile, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if idx := strings.Index(href, "#"); idx >= 0 {
		href = href[:idx]
	}
	return path.Clean(path.Join(path.Dir(baseFile), href))
}

func epubRootfilePath(reader *zip.Reader) string {
	data, ok := zipFileBytes(reader, "META-INF/container.xml")
	if !ok {
		return ""
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "rootfile" {
			continue
		}
		return path.Clean(attrValue(start, "full-path"))
	}
}

func parseEPUBManifestAndSpine(opfPath string, data []byte) (map[string]string, []string, string, string) {
	manifest := map[string]string{}
	var spineIDs []string
	navPath := ""
	ncxPath := ""
	spineTOCID := ""
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "item":
			id := attrValue(start, "id")
			href := attrValue(start, "href")
			if id == "" || href == "" {
				continue
			}
			fullPath := path.Clean(path.Join(path.Dir(opfPath), href))
			manifest[id] = fullPath
			properties := strings.ToLower(attrValue(start, "properties"))
			mediaType := strings.ToLower(attrValue(start, "media-type"))
			lowerHref := strings.ToLower(href)
			if strings.Contains(properties, "nav") {
				navPath = fullPath
			}
			if strings.Contains(mediaType, "application/x-dtbncx+xml") || strings.HasSuffix(lowerHref, ".ncx") {
				ncxPath = fullPath
			}
		case "spine":
			spineTOCID = attrValue(start, "toc")
		case "itemref":
			idref := attrValue(start, "idref")
			if idref != "" {
				spineIDs = append(spineIDs, idref)
			}
		}
	}
	if spineTOCID != "" {
		if href := manifest[spineTOCID]; href != "" {
			ncxPath = href
		}
	}
	return manifest, spineIDs, navPath, ncxPath
}

func parseEPUBNavTOC(navPath string, data []byte, pageByHref map[string]int) []tocItem {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var items []tocItem
	inTOC := false
	navDepth := 0
	olDepth := 0
	captureHref := ""
	captureLevel := 1
	var capture strings.Builder

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "nav" {
				navDepth++
				if strings.Contains(attrValue(t, "type"), "toc") {
					inTOC = true
				}
				continue
			}
			if !inTOC {
				continue
			}
			if t.Name.Local == "ol" {
				olDepth++
			} else if t.Name.Local == "a" {
				captureHref = resolveEPUBPath(navPath, attrValue(t, "href"))
				captureLevel = olDepth
				if captureLevel < 1 {
					captureLevel = 1
				}
				capture.Reset()
			}
		case xml.CharData:
			if inTOC && captureHref != "" {
				capture.Write([]byte(t))
			}
		case xml.EndElement:
			if inTOC && t.Name.Local == "a" && captureHref != "" {
				title := strings.Join(strings.Fields(capture.String()), " ")
				page := pageByHref[path.Clean(captureHref)]
				if page < 1 {
					page = 1
				}
				if title != "" {
					items = append(items, tocItem{Title: title, Page: page, Level: captureLevel})
				}
				captureHref = ""
			} else if inTOC && t.Name.Local == "ol" {
				olDepth--
				if olDepth < 0 {
					olDepth = 0
				}
			} else if t.Name.Local == "nav" {
				if inTOC {
					return items
				}
				navDepth--
				if navDepth < 0 {
					navDepth = 0
				}
			}
		}
	}
	return items
}

func parseEPUBNCXTOC(ncxPath string, data []byte, pageByHref map[string]int) []tocItem {
	type ncxPoint struct {
		title    strings.Builder
		href     string
		level    int
		inText   bool
		appended bool
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var items []tocItem
	var stack []*ncxPoint

	appendTop := func() {
		if len(stack) == 0 {
			return
		}
		point := stack[len(stack)-1]
		if point.appended {
			return
		}
		point.appended = true
		title := strings.Join(strings.Fields(point.title.String()), " ")
		if title == "" {
			return
		}
		href := resolveEPUBPath(ncxPath, point.href)
		page := pageByHref[path.Clean(href)]
		if page < 1 {
			page = 1
		}
		level := point.level
		if level < 1 {
			level = 1
		}
		items = append(items, tocItem{Title: title, Page: page, Level: level})
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "navPoint":
				appendTop()
				stack = append(stack, &ncxPoint{level: len(stack) + 1})
			case "content":
				if len(stack) > 0 {
					stack[len(stack)-1].href = attrValue(t, "src")
				}
			case "text":
				if len(stack) > 0 {
					stack[len(stack)-1].inText = true
				}
			}
		case xml.CharData:
			if len(stack) > 0 && stack[len(stack)-1].inText {
				stack[len(stack)-1].title.Write([]byte(t))
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "text":
				if len(stack) > 0 {
					stack[len(stack)-1].inText = false
				}
			case "navPoint":
				appendTop()
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}
	return items
}

func parseEPUBTOC(bookBytes []byte) []tocItem {
	reader, err := zip.NewReader(bytes.NewReader(bookBytes), int64(len(bookBytes)))
	if err != nil {
		return nil
	}
	opfPath := epubRootfilePath(reader)
	if opfPath == "" {
		return nil
	}
	opfData, ok := zipFileBytes(reader, opfPath)
	if !ok {
		return nil
	}
	manifest, spineIDs, navPath, ncxPath := parseEPUBManifestAndSpine(opfPath, opfData)
	pageByHref := map[string]int{}
	for index, idref := range spineIDs {
		if href := manifest[idref]; href != "" {
			pageByHref[path.Clean(href)] = index + 1
		}
	}
	if navPath != "" {
		if navData, ok := zipFileBytes(reader, navPath); ok {
			if toc := parseEPUBNavTOC(navPath, navData, pageByHref); len(toc) > 0 {
				return toc
			}
		}
	}
	if ncxPath == "" {
		for _, file := range reader.File {
			if strings.HasSuffix(strings.ToLower(file.Name), ".ncx") {
				ncxPath = path.Clean(file.Name)
				break
			}
		}
	}
	if ncxPath != "" {
		if ncxData, ok := zipFileBytes(reader, ncxPath); ok {
			if toc := parseEPUBNCXTOC(ncxPath, ncxData, pageByHref); len(toc) > 0 {
				return toc
			}
		}
	}
	return nil
}

func tocTitleCandidates(title string) []string {
	parts := []string{title}
	for _, sep := range []string{"│", "|", "：", ":", "──", "—", "-"} {
		if idx := strings.Index(title, sep); idx >= 0 && idx+len(sep) < len(title) {
			parts = append(parts, title[idx+len(sep):])
		}
	}
	seen := map[string]bool{}
	candidates := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := cleanString(strings.TrimSpace(part))
		if cleaned == "" || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		candidates = append(candidates, cleaned)
	}
	return candidates
}

func findTOCMatchIndex(cleanText string, candidates []string) int {
	best := -1
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		idx := strings.Index(cleanText, candidate)
		if idx >= 0 && (best == -1 || idx < best) {
			best = idx
		}
	}
	return best
}

func resolveEPUBTOCPages(doc *fitz.Document, items []tocItem) []tocItem {
	if len(items) == 0 {
		return items
	}
	titleCandidates := make([][]string, len(items))
	for i, item := range items {
		titleCandidates[i] = tocTitleCandidates(item.Title)
	}

	pageCount := doc.NumPage()
	pageTexts := make([]string, pageCount)
	tocLikePage := make([]bool, pageCount)
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		text, err := doc.Text(pageIndex)
		if err != nil {
			continue
		}
		cleanText := cleanString(text)
		pageTexts[pageIndex] = cleanText
		matches := 0
		for _, candidates := range titleCandidates {
			if findTOCMatchIndex(cleanText, candidates) >= 0 {
				matches++
			}
			if matches >= 3 {
				tocLikePage[pageIndex] = true
				break
			}
		}
	}

	lastPageIndex := 0
	for itemIndex := range items {
		candidates := titleCandidates[itemIndex]
		bestPage := -1
		bestIndex := 1 << 30
		for pageIndex := lastPageIndex; pageIndex < pageCount; pageIndex++ {
			if tocLikePage[pageIndex] {
				continue
			}
			idx := findTOCMatchIndex(pageTexts[pageIndex], candidates)
			if idx < 0 {
				continue
			}
			if idx < 12 {
				bestPage = pageIndex
				break
			}
			if idx < bestIndex {
				bestPage = pageIndex
				bestIndex = idx
			}
		}
		if bestPage == -1 {
			for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
				if tocLikePage[pageIndex] {
					continue
				}
				idx := findTOCMatchIndex(pageTexts[pageIndex], candidates)
				if idx < 0 {
					continue
				}
				if idx < 12 {
					bestPage = pageIndex
					break
				}
				if idx < bestIndex {
					bestPage = pageIndex
					bestIndex = idx
				}
			}
		}
		if bestPage >= 0 {
			items[itemIndex].Page = bestPage + 1
			lastPageIndex = bestPage
		} else if items[itemIndex].Page < 1 {
			items[itemIndex].Page = 1
		}
	}
	return items
}

func parseFitzOutline(outlines []fitz.Outline) []tocItem {
	if len(outlines) == 0 {
		return nil
	}
	type node struct {
		item     tocItem
		children []*node
	}
	var stack []*node
	var roots []*node

	for _, o := range outlines {
		page := o.Page + 1
		if page < 1 {
			page = 1
		}
		n := &node{
			item: tocItem{
				Title: o.Title,
				Page:  page,
				Level: o.Level,
			},
		}

		for len(stack) > 0 && stack[len(stack)-1].item.Level >= o.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			parent := stack[len(stack)-1]
			parent.children = append(parent.children, n)
		}
		stack = append(stack, n)
	}

	var buildItems func([]*node) []tocItem
	buildItems = func(nodes []*node) []tocItem {
		if len(nodes) == 0 {
			return nil
		}
		items := make([]tocItem, len(nodes))
		for i, n := range nodes {
			items[i] = n.item
			items[i].Children = buildItems(n.children)
		}
		return items
	}
	return buildItems(roots)
}

func cleanString(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || (r >= 0x4e00 && r <= 0x9fff) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func resolveOutlinePages(doc *fitz.Document, outlines []fitz.Outline) []fitz.Outline {
	if len(outlines) == 0 {
		return outlines
	}

	// If there are positive pages, it's likely a PDF and doesn't need resolution
	hasValidPages := false
	for _, o := range outlines {
		if o.Page >= 0 {
			hasValidPages = true
			break
		}
	}
	if hasValidPages {
		return outlines
	}

	lastPage := 1
	numPages := doc.NumPage()

	for idx := range outlines {
		title := strings.TrimSpace(outlines[idx].Title)
		if title == "" {
			outlines[idx].Page = lastPage - 1
			continue
		}

		cleanTitle := cleanString(title)
		if cleanTitle == "" {
			outlines[idx].Page = lastPage - 1
			continue
		}

		if idx == 0 || title == "书名页" || title == "版权页" {
			lastPage = 1
			outlines[idx].Page = 0
			continue
		}

		bestPage := -1
		bestIndex := 999999

		startSearchIdx := lastPage - 1
		if startSearchIdx < 0 {
			startSearchIdx = 0
		}

		for p := startSearchIdx; p < numPages; p++ {
			text, err := doc.Text(p)
			if err != nil {
				continue
			}

			cleanText := cleanString(text)
			index := strings.Index(cleanText, cleanTitle)
			if index != -1 {
				if index < 5 {
					bestPage = p + 1
					bestIndex = index
					break
				}
				if index < bestIndex {
					bestPage = p + 1
					bestIndex = index
				}
			}
		}

		if bestPage == -1 {
			bestIndex = 999999
			for p := 0; p < numPages; p++ {
				text, err := doc.Text(p)
				if err != nil {
					continue
				}

				cleanText := cleanString(text)
				index := strings.Index(cleanText, cleanTitle)
				if index != -1 {
					if index < 5 {
						bestPage = p + 1
						bestIndex = index
						break
					}
					if index < bestIndex {
						bestPage = p + 1
						bestIndex = index
					}
				}
			}
		}

		if bestPage == -1 {
			outlines[idx].Page = lastPage - 1
		} else {
			lastPage = bestPage
			outlines[idx].Page = bestPage - 1
		}
	}

	return outlines
}

func parseFitzBook(app core.App, bookBytes []byte, record *core.Record, ext string) error {
	if ext == ".epub" {
		bookBytes = injectEPUBRenderFont(bookBytes)
	}

	fitzMu.Lock()
	defer fitzMu.Unlock()

	tmpFile, err := os.CreateTemp("", "fitz-book-*"+ext)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(bookBytes); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	doc, err := fitz.New(tmpFile.Name())
	if err != nil {
		return err
	}
	defer doc.Close()

	if ext == ".epub" {
		if toc := parseEPUBTOC(bookBytes); len(toc) > 0 {
			record.Set("toc", resolveEPUBTOCPages(doc, toc))
		}
	}
	if record.Get("toc") == nil {
		outlines, err := doc.ToC()
		if err == nil && len(outlines) > 0 {
			resolvedOutlines := resolveOutlinePages(doc, outlines)
			record.Set("toc", parseFitzOutline(resolvedOutlines))
		}
	}

	pagesCollection, err := app.FindCollectionByNameOrId("book_pages")
	if err != nil {
		return err
	}

	pageCount := doc.NumPage()
	for i := 0; i < pageCount; i++ {
		text, _ := doc.Text(i)
		pageRecord := core.NewRecord(pagesCollection)
		pageRecord.Set("book", record.Id)
		pageRecord.Set("page_number", i+1)
		pageRecord.Set("text", text)
		// Since EPUB/MOBI are reflowable, they don't have intrinsic widths and heights like PDFs.
		// We omit setting width/height, which is perfectly safe for non-PDFs.
		if err := app.Save(pageRecord); err != nil {
			return err
		}
	}

	record.Set("page_count", pageCount)
	record.Set("parse_status", "completed")
	return app.Save(record)
}

func cjkRenderFontPath() string {
	candidates := []string{
		os.Getenv("EPUB_RENDER_FONT"),
		"fonts/DroidSansFallback.ttf",
		"../fonts/DroidSansFallback.ttf",
		"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
		"/usr/share/fonts/truetype/droid/DroidSansFallback.ttf",
		"/usr/share/fonts/droid/DroidSansFallback.ttf",
		"/usr/share/fonts/droid/DroidSansFallbackFull.ttf",
		"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func applyFontCacheHeaders(headers http.Header) {
	headers.Set("Cache-Control", fontCacheControl)
}

func serveCachedFont(re *core.RequestEvent, fontPath string) error {
	info, err := os.Stat(fontPath)
	if err != nil {
		return re.NotFoundError("failed to stat font file", err)
	}

	etag := fmt.Sprintf(`W/"%d-%d"`, info.Size(), info.ModTime().Unix())
	headers := re.Response.Header()
	applyFontCacheHeaders(headers)
	headers.Set("ETag", etag)
	headers.Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	headers.Set("Access-Control-Allow-Origin", "*")

	if inm := re.Request.Header.Get("If-None-Match"); inm != "" && strings.Contains(inm, etag) {
		re.Response.WriteHeader(http.StatusNotModified)
		return nil
	}
	if ims := re.Request.Header.Get("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil && !info.ModTime().After(t) {
			re.Response.WriteHeader(http.StatusNotModified)
			return nil
		}
	}

	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return re.NotFoundError("failed to read font file", err)
	}
	return re.Blob(http.StatusOK, "font/ttf", fontBytes)
}

func injectEPUBRenderFont(bookBytes []byte) []byte {
	fontPath := cjkRenderFontPath()
	if fontPath == "" {
		return bookBytes
	}
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil || len(fontBytes) == 0 {
		return bookBytes
	}

	reader, err := zip.NewReader(bytes.NewReader(bookBytes), int64(len(bookBytes)))
	if err != nil {
		return bookBytes
	}

	const fontName = "OEBPS/Fonts/ebook-reader-cjk.ttf"
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	defer writer.Close()
	seenFont := false

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return bookBytes
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return bookBytes
		}

		if file.Name == fontName {
			seenFont = true
		}
		lowerName := strings.ToLower(file.Name)
		if strings.HasSuffix(lowerName, ".xhtml") || strings.HasSuffix(lowerName, ".html") || strings.HasSuffix(lowerName, ".htm") {
			fontURL, err := filepath.Rel(filepath.Dir(file.Name), fontName)
			if err != nil {
				fontURL = fontName
			}
			fontURL = filepath.ToSlash(fontURL)
			style := `<style id="ebook-reader-cjk-font">@font-face{font-family:'EbookReaderCJK';src:url('` + fontURL + `');}html,body,body *{font-family:'EbookReaderCJK',sans-serif !important;}</style>`
			html := string(data)
			if !strings.Contains(html, "ebook-reader-cjk-font") {
				lowerHTML := strings.ToLower(html)
				if idx := strings.Index(lowerHTML, "</head>"); idx >= 0 {
					html = html[:idx] + style + html[idx:]
				} else {
					html = style + html
				}
				data = []byte(html)
			}
		}

		header := file.FileHeader
		entry, err := writer.CreateHeader(&header)
		if err != nil {
			return bookBytes
		}
		if _, err := entry.Write(data); err != nil {
			return bookBytes
		}
	}

	if !seenFont {
		header := &zip.FileHeader{Name: fontName, Method: zip.Deflate}
		header.SetMode(0644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return bookBytes
		}
		if _, err := entry.Write(fontBytes); err != nil {
			return bookBytes
		}
	}
	if err := writer.Close(); err != nil {
		return bookBytes
	}
	return buffer.Bytes()
}

func renderFitzPagePNG(bookBytes []byte, pageNumber int, ext string) ([]byte, error) {
	if ext == ".epub" {
		bookBytes = injectEPUBRenderFont(bookBytes)
	}

	fitzMu.Lock()
	defer fitzMu.Unlock()

	tmpFile, err := os.CreateTemp("", "fitz-render-*"+ext)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(bookBytes); err != nil {
		tmpFile.Close()
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	doc, err := fitz.New(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	if pageNumber < 1 || pageNumber > doc.NumPage() {
		return nil, fmt.Errorf("page %d out of range 1-%d", pageNumber, doc.NumPage())
	}

	dpi := float64(renderDPI())
	return doc.ImagePNG(pageNumber-1, dpi)
}

func renderFitzPageHTML(bookBytes []byte, pageNumber int, ext string) (string, error) {
	if ext == ".epub" {
		bookBytes = injectEPUBRenderFont(bookBytes)
	}

	fitzMu.Lock()
	defer fitzMu.Unlock()

	tmpFile, err := os.CreateTemp("", "fitz-html-*"+ext)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(bookBytes); err != nil {
		tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	doc, err := fitz.New(tmpFile.Name())
	if err != nil {
		return "", err
	}
	defer doc.Close()

	if pageNumber < 1 || pageNumber > doc.NumPage() {
		return "", fmt.Errorf("page %d out of range 1-%d", pageNumber, doc.NumPage())
	}

	return doc.HTML(pageNumber-1, true)
}

func decorateReaderPageHTML(htmlStr string) string {
	injections := `
<style>
/* Clean up html body for iframe rendering */
body {
	margin: 0;
	padding: 0;
	overflow: hidden;
	background-color: transparent !important;
}
div[id^="page"] {
	transform-origin: top left;
	margin: 0 !important;
	box-shadow: none !important;
}
</style>
<script>
function resize() {
	var page = document.querySelector('div[id^="page"]');
	if (!page) return;
	var width = window.innerWidth;
	var widthStr = page.style.width;
	var scale = 1;
	if (widthStr.indexOf('pt') !== -1) {
		scale = width / (parseFloat(widthStr) * 1.33333);
	} else if (widthStr.indexOf('px') !== -1) {
		scale = width / parseFloat(widthStr);
	} else {
		scale = width / parseFloat(widthStr);
	}
	page.style.transform = 'scale(' + scale + ')';
	var heightStr = page.style.height;
	var pageHeight = parseFloat(heightStr);
	if (heightStr.indexOf('pt') !== -1) {
		pageHeight *= 1.33333;
	}
	var computedHeight = pageHeight * scale;
	document.body.style.height = computedHeight + 'px';
	window.parent.postMessage({ type: 'ebook-reader-page-height', height: computedHeight }, '*');
}
window.addEventListener('resize', resize);
window.addEventListener('DOMContentLoaded', resize);
setTimeout(resize, 0);
</script>
`

	if idx := strings.Index(strings.ToLower(htmlStr), "</head>"); idx >= 0 {
		return htmlStr[:idx] + injections + htmlStr[idx:]
	}
	return injections + htmlStr
}

func parseBook(app core.App, svc *pdfService, bookID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in parseBook: %v", r)
			if record, err := app.FindRecordById("books", bookID); err == nil {
				markParseFailed(app, record, fmt.Errorf("panic in parseBook: %v", r))
			}
		}
	}()

	record, err := app.FindRecordById("books", bookID)
	if err != nil {
		return
	}
	record.Set("parse_status", "processing")
	record.Set("parse_error", "")
	_ = app.Save(record)

	bookBytes, err := recordPDFBytes(app, record)
	if err != nil {
		markParseFailed(app, record, err)
		return
	}

	filename := record.GetString("file")
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".epub" || ext == ".mobi" {
		err = parseFitzBook(app, bookBytes, record, ext)
	} else {
		err = svc.withParsedDocument(bookBytes, func(instance pdfium.Pdfium, docRef any, pageCount int) error {
			bookmarks, bookmarksErr := instance.GetBookmarks(&requests.GetBookmarks{Document: docRef.(requests.PageByIndex).Document})
			if bookmarksErr == nil && bookmarks != nil {
				record.Set("toc", flattenBookmarks(bookmarks.Bookmarks, 1))
			}

			pagesCollection, err := app.FindCollectionByNameOrId("book_pages")
			if err != nil {
				return err
			}
			for i := 0; i < pageCount; i++ {
				pageReq := requests.Page{ByIndex: &requests.PageByIndex{Document: docRef.(requests.PageByIndex).Document, Index: i}}
				text, _ := instance.GetPageText(&requests.GetPageText{Page: pageReq})
				size, _ := instance.GetPageSize(&requests.GetPageSize{Page: pageReq})
				pageRecord := core.NewRecord(pagesCollection)
				pageRecord.Set("book", record.Id)
				pageRecord.Set("page_number", i+1)
				if text != nil {
					pageRecord.Set("text", text.Text)
				}
				if size != nil {
					pageRecord.Set("width", size.Width)
					pageRecord.Set("height", size.Height)
				}
				if err := app.Save(pageRecord); err != nil {
					return err
				}
			}
			record.Set("page_count", pageCount)
			record.Set("parse_status", "completed")
			return app.Save(record)
		})
	}
	if err != nil {
		markParseFailed(app, record, err)
	}
}

func (s *pdfService) withParsedDocument(pdfBytes []byte, fn func(pdfium.Pdfium, any, int) error) error {
	instance, err := s.pool.GetInstance(30 * time.Second)
	if err != nil {
		return err
	}
	defer instance.Close()
	doc, err := instance.OpenDocument(&requests.OpenDocument{File: &pdfBytes})
	if err != nil {
		return err
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})
	count, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return err
	}
	return fn(instance, requests.PageByIndex{Document: doc.Document}, count.PageCount)
}

func deleteBookChildren(app core.App, bookID string) error {
	for _, collection := range []string{"book_pages", "bookmarks", "notes", "reading_records"} {
		records, err := app.FindRecordsByFilter(collection, `book = "`+bookID+`"`, "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}
	}
	return nil
}

func markParseFailed(app core.App, record *core.Record, err error) {
	record.Set("parse_status", "failed")
	record.Set("parse_error", err.Error())
	_ = app.Save(record)
}

func renderPagePNG(app core.App, svc *pdfService, book *core.Record, pageNumber int) ([]byte, error) {
	pdfBytes, err := recordPDFBytes(app, book)
	if err != nil {
		return nil, err
	}
	var out []byte
	err = svc.withParsedDocument(pdfBytes, func(instance pdfium.Pdfium, docRef any, pageCount int) error {
		if pageNumber < 1 || pageNumber > pageCount {
			return fmt.Errorf("page %d out of range 1-%d", pageNumber, pageCount)
		}
		resp, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{Page: requests.Page{ByIndex: &requests.PageByIndex{Document: docRef.(requests.PageByIndex).Document, Index: pageNumber - 1}}, DPI: renderDPI(), RenderFlags: enums.FPDF_RENDER_FLAG_REVERSE_BYTE_ORDER})
		if err != nil {
			return err
		}
		defer resp.Cleanup()
		buf := bytes.NewBuffer(nil)
		if err := png.Encode(buf, resp.Result.Image); err != nil {
			return err
		}
		out = buf.Bytes()
		return nil
	})
	return out, err
}

func authTokenFromHTTPRequest(req *http.Request) string {
	authorization := strings.TrimSpace(req.Header.Get("Authorization"))
	if len(authorization) > len("Bearer ") && strings.EqualFold(authorization[:len("Bearer ")], "Bearer ") {
		return strings.TrimSpace(authorization[len("Bearer "):])
	}
	return req.URL.Query().Get("token")
}

func authTokenFromRequest(re *core.RequestEvent) string {
	return authTokenFromHTTPRequest(re.Request)
}

func registerRoutes(app core.App, svc *pdfService) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/books/{id}/pages/{page}/image", func(re *core.RequestEvent) error {
			token := authTokenFromRequest(re)
			if token == "" {
				return re.UnauthorizedError("missing auth token", nil)
			}
			auth, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
			if err != nil {
				return re.UnauthorizedError("invalid auth token", nil)
			}
			bookID := re.Request.PathValue("id")
			book, err := app.FindRecordById("books", bookID)
			if err != nil {
				return re.NotFoundError("book not found", nil)
			}
			if book.GetString("user") != auth.Id {
				return re.ForbiddenError("not your book", nil)
			}
			pageNumber, err := strconv.Atoi(re.Request.PathValue("page"))
			if err != nil {
				return re.BadRequestError("invalid page", nil)
			}
			if book.GetString("parse_status") != "completed" {
				return re.BadRequestError("book parsing is not completed", nil)
			}

			filename := book.GetString("file")
			ext := strings.ToLower(filepath.Ext(filename))
			var pngBytes []byte
			if ext == ".epub" || ext == ".mobi" {
				bookBytes, err := recordPDFBytes(app, book)
				if err != nil {
					return re.InternalServerError(err.Error(), nil)
				}
				pngBytes, err = renderFitzPagePNG(bookBytes, pageNumber, ext)
				if err != nil {
					return re.InternalServerError(err.Error(), nil)
				}
			} else {
				pngBytes, err = renderPagePNG(app, svc, book, pageNumber)
				if err != nil {
					return re.InternalServerError(err.Error(), nil)
				}
			}
			return re.Blob(http.StatusOK, "image/png", pngBytes)
		})

		e.Router.GET("/api/books/{id}/pages/{page}/html", func(re *core.RequestEvent) error {
			token := authTokenFromRequest(re)
			if token == "" {
				return re.UnauthorizedError("missing auth token", nil)
			}
			auth, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
			if err != nil {
				return re.UnauthorizedError("invalid auth token", nil)
			}
			bookID := re.Request.PathValue("id")
			book, err := app.FindRecordById("books", bookID)
			if err != nil {
				return re.NotFoundError("book not found", nil)
			}
			if book.GetString("user") != auth.Id {
				return re.ForbiddenError("not your book", nil)
			}
			pageNumber, err := strconv.Atoi(re.Request.PathValue("page"))
			if err != nil {
				return re.BadRequestError("invalid page", nil)
			}
			if book.GetString("parse_status") != "completed" {
				return re.BadRequestError("book parsing is not completed", nil)
			}
			filename := book.GetString("file")
			ext := strings.ToLower(filepath.Ext(filename))
			bookBytes, err := recordPDFBytes(app, book)
			if err != nil {
				return re.InternalServerError(err.Error(), nil)
			}

			htmlStr, err := renderFitzPageHTML(bookBytes, pageNumber, ext)
			if err != nil {
				return re.InternalServerError(err.Error(), nil)
			}
			return re.Blob(http.StatusOK, "text/html; charset=utf-8", []byte(decorateReaderPageHTML(htmlStr)))
		})

		e.Router.GET("/api/fonts/{filename}", func(re *core.RequestEvent) error {
			filename := re.Request.PathValue("filename")
			isUnhashed := filename == "DroidSansFallback.ttf"
			isHashed := strings.HasPrefix(filename, "DroidSansFallback.") && strings.HasSuffix(filename, ".ttf")
			if !isUnhashed && !isHashed {
				return re.NotFoundError("font file not found", nil)
			}
			fontPath := cjkRenderFontPath()
			if fontPath == "" {
				return re.NotFoundError("font file not found", nil)
			}
			return serveCachedFont(re, fontPath)
		})

		publicDir := os.Getenv("PUBLIC_DIR")
		if publicDir == "" {
			publicDir = filepath.Join("..", "dist")
		}
		if info, err := os.Stat(publicDir); err == nil && info.IsDir() {
			e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), true))
		}
		return e.Next()
	})
}

func main() {
	app := pocketbase.New()
	svc, err := newPDFService()
	if err != nil {
		log.Fatal(err)
	}
	defer svc.close()

	app.OnRecordAfterCreateSuccess("books").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		go parseBook(app, svc, e.Record.Id)
		return nil
	})

	app.OnRecordDeleteRequest("books").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := deleteBookChildren(app, e.Record.Id); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordDelete("books").BindFunc(func(e *core.RecordEvent) error {
		if err := deleteBookChildren(app, e.Record.Id); err != nil {
			return err
		}
		return e.Next()
	})
	registerRoutes(app, svc)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
