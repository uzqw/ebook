package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// mcpSectionPageChunk caps how many pages one get_section_text response covers.
	mcpSectionPageChunk = 10
	// mcpMaxTextBytes caps tool text payloads (~100KB) before an explicit marker.
	mcpMaxTextBytes     = 100 * 1024
	mcpSearchPerPage    = 3
	mcpSearchMaxResults = 20
)

// mcpUserContextKey carries the authenticated user's record id from the /mcp
// route wrapper (where HTTP status codes are still writable) into the MCP
// tool handlers. The wrapper validates the token; handlers only read the id.
type mcpUserContextKey struct{}

// flattenTOC returns a pre-order DFS flattening of the TOC tree.
func flattenTOC(items []tocItem) []tocItem {
	flat := make([]tocItem, 0, len(items))
	for _, item := range items {
		flat = append(flat, item)
		flat = append(flat, flattenTOC(item.Children)...)
	}
	return flat
}

// resolveSectionRange computes the page range of flat[idx]: from the item's
// own page to the page before the next entry at the same or a higher level
// (nested children belong to the section), or pageCount for the last section.
// Same-page sections yield an empty range; the caller serves that single page
// and reports isPartial.
func resolveSectionRange(flat []tocItem, idx, pageCount int) (from, to int, isPartial bool) {
	item := flat[idx]
	from = item.Page
	if from < 1 {
		from = 1
	}
	to = pageCount
	for j := idx + 1; j < len(flat); j++ {
		if flat[j].Level <= item.Level {
			to = flat[j].Page - 1
			break
		}
	}
	if to > pageCount {
		to = pageCount
	}
	if to < from {
		to = from
		isPartial = true
	}
	return from, to, isPartial
}

// findTOCIndex locates a flattened TOC entry by title: exact (case-insensitive)
// match first, then first entry whose title contains the query.
func findTOCIndex(flat []tocItem, title string) int {
	want := strings.ToLower(strings.TrimSpace(title))
	if want == "" {
		return -1
	}
	for i, item := range flat {
		if strings.ToLower(strings.TrimSpace(item.Title)) == want {
			return i
		}
	}
	for i, item := range flat {
		if strings.Contains(strings.ToLower(item.Title), want) {
			return i
		}
	}
	return -1
}

// mcpTruncateText caps a response at ~100KB, marking the cut explicitly
// instead of silently dropping the remainder.
func mcpTruncateText(s string) (string, bool) {
	if len(s) <= mcpMaxTextBytes {
		return s, false
	}
	cut := s[:mcpMaxTextBytes]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut + "\n\n[truncated: response exceeded 100KB; use start_page/from_page to continue]", true
}

// mcpPagesText joins stored page text for [from,to] with --- p.N ---
// separators, skipping pages that have no stored text.
func mcpPagesText(app core.App, bookID string, from, to int) (string, error) {
	records, err := app.FindRecordsByFilter("book_pages",
		"book = {:book} && page_number >= {:from} && page_number <= {:to}",
		"page_number", 0, 0,
		dbx.Params{"book": bookID, "from": from, "to": to})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, rec := range records {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "--- p.%d ---\n\n%s", rec.GetInt("page_number"), rec.GetString("text"))
	}
	return b.String(), nil
}

// mcpOwnedBook loads a book scoped to the user; a miss surfaces as a plain
// "not found" tool error so handlers never leak another user's book ids.
func mcpOwnedBook(app core.App, ctx context.Context, bookID string) (*core.Record, *mcp.CallToolResult) {
	userID, _ := ctx.Value(mcpUserContextKey{}).(string)
	if userID == "" {
		return nil, mcp.NewToolResultError("unauthenticated")
	}
	book, err := app.FindFirstRecordByFilter("books", "id = {:id} && user = {:user}",
		dbx.Params{"id": bookID, "user": userID})
	if err != nil {
		return nil, mcp.NewToolResultError("book not found")
	}
	return book, nil
}

func mcpJSONResult(v any) *mcp.CallToolResult {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultText(string(data))
}

func mcpListBooks(app core.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, _ := ctx.Value(mcpUserContextKey{}).(string)
		if userID == "" {
			return mcp.NewToolResultError("unauthenticated"), nil
		}
		records, err := app.FindRecordsByFilter("books", "user = {:user}", "-created", 0, 0,
			dbx.Params{"user": userID})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		type bookRow struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Author      string `json:"author"`
			PageCount   int    `json:"page_count"`
			CurrentPage int    `json:"current_page"`
			ParseStatus string `json:"parse_status"`
		}
		rows := make([]bookRow, 0, len(records))
		for _, rec := range records {
			rows = append(rows, bookRow{
				ID:          rec.Id,
				Title:       rec.GetString("title"),
				Author:      rec.GetString("author"),
				PageCount:   rec.GetInt("page_count"),
				CurrentPage: rec.GetInt("current_page"),
				ParseStatus: rec.GetString("parse_status"),
			})
		}
		return mcpJSONResult(rows), nil
	}
}

func mcpGetTOC(app core.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bookID, err := req.RequireString("book_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		book, failure := mcpOwnedBook(app, ctx, bookID)
		if failure != nil {
			return failure, nil
		}
		raw, err := json.Marshal(book.GetRaw("toc"))
		if err != nil || string(raw) == "null" || len(raw) == 0 {
			raw = []byte("[]")
		}
		return mcp.NewToolResultText(string(raw)), nil
	}
}

func mcpGetSectionText(app core.App) server.ToolHandlerFunc {
	type response struct {
		Title     string `json:"title"`
		FromPage  int    `json:"from_page"`
		ToPage    int    `json:"to_page"`
		NextPage  *int   `json:"next_page"`
		IsPartial bool   `json:"is_partial,omitempty"`
		Note      string `json:"note,omitempty"`
		Truncated bool   `json:"truncated,omitempty"`
		Text      string `json:"text"`
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bookID, err := req.RequireString("book_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := req.RequireString("section_title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		book, failure := mcpOwnedBook(app, ctx, bookID)
		if failure != nil {
			return failure, nil
		}

		var items []tocItem
		if raw, err := json.Marshal(book.GetRaw("toc")); err == nil {
			_ = json.Unmarshal(raw, &items)
		}
		flat := flattenTOC(items)
		idx := findTOCIndex(flat, title)
		if idx < 0 {
			return mcp.NewToolResultErrorf("section %q not found in book TOC", title), nil
		}

		sectionFrom, sectionTo, isPartial := resolveSectionRange(flat, idx, book.GetInt("page_count"))
		from := sectionFrom
		if start := req.GetInt("start_page", 0); start > 0 {
			from = start
		}
		if from < sectionFrom || from > sectionTo {
			return mcp.NewToolResultErrorf("start_page %d out of section range [%d, %d]", from, sectionFrom, sectionTo), nil
		}
		to := sectionFrom + mcpSectionPageChunk - 1
		if from > sectionFrom {
			to = from + mcpSectionPageChunk - 1
		}
		if to > sectionTo {
			to = sectionTo
		}

		text, err := mcpPagesText(app, bookID, from, to)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		text, truncated := mcpTruncateText(text)

		resp := response{
			Title:     flat[idx].Title,
			FromPage:  from,
			ToPage:    to,
			IsPartial: isPartial,
			Truncated: truncated,
			Text:      text,
		}
		if isPartial {
			resp.Note = "section shares its start page with other sections; returning that page's text"
		}
		if to < sectionTo {
			next := to + 1
			resp.NextPage = &next
		}
		return mcpJSONResult(resp), nil
	}
}

func mcpGetPages(app core.App) server.ToolHandlerFunc {
	type response struct {
		FromPage  int    `json:"from_page"`
		ToPage    int    `json:"to_page"`
		Truncated bool   `json:"truncated,omitempty"`
		Text      string `json:"text"`
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bookID, err := req.RequireString("book_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		from, err := req.RequireInt("from_page")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		to, err := req.RequireInt("to_page")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		book, failure := mcpOwnedBook(app, ctx, bookID)
		if failure != nil {
			return failure, nil
		}
		if from < 1 {
			from = 1
		}
		if pageCount := book.GetInt("page_count"); pageCount > 0 && to > pageCount {
			to = pageCount
		}
		if to < from {
			return mcp.NewToolResultError("to_page must be >= from_page"), nil
		}
		text, err := mcpPagesText(app, bookID, from, to)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		text, truncated := mcpTruncateText(text)
		return mcpJSONResult(response{FromPage: from, ToPage: to, Truncated: truncated, Text: text}), nil
	}
}

func mcpSearchInBook(app core.App) server.ToolHandlerFunc {
	type hit struct {
		PageNumber int      `json:"page_number"`
		Snippets   []string `json:"snippets"`
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bookID, err := req.RequireString("book_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		_, failure := mcpOwnedBook(app, ctx, bookID)
		if failure != nil {
			return failure, nil
		}

		records, err := app.FindRecordsByFilter("book_pages", "book = {:book} && text ~ {:q}",
			"page_number", 0, 0, dbx.Params{"book": bookID, "q": query})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		needle := strings.ToLower(query)
		hits := make([]hit, 0, mcpSearchMaxResults)
		total := 0
		for _, rec := range records {
			if total >= mcpSearchMaxResults {
				break
			}
			var snippets []string
			for _, line := range strings.Split(rec.GetString("text"), "\n") {
				if total >= mcpSearchMaxResults || len(snippets) >= mcpSearchPerPage {
					break
				}
				if strings.Contains(strings.ToLower(line), needle) {
					snippets = append(snippets, strings.TrimSpace(line))
					total++
				}
			}
			if len(snippets) > 0 {
				hits = append(hits, hit{PageNumber: rec.GetInt("page_number"), Snippets: snippets})
			}
		}
		return mcpJSONResult(hits), nil
	}
}

// registerMCP mounts a read-only MCP server (streamable HTTP, stateless) at
// /mcp inside the PocketBase process. The route wrapper is the auth gate:
// no/invalid Bearer token → 401 before any MCP processing; a valid token's
// user id is injected into the request context, and every tool additionally
// scopes its queries to that user (books.user = user id) so one user can
// never read another user's books or pages.
//
// The route is registered for POST only (not Any): the app's static frontend
// route is GET /{path...}, and Go's ServeMux panics on an all-methods pattern
// conflicting with it regardless of registration order. MCP streamable HTTP
// clients send JSON-RPC exclusively via POST, so POST-only loses nothing.
func registerMCP(app core.App, e *core.ServeEvent) {
	srv := server.NewMCPServer("ebook-reader", "0.1.0", server.WithToolCapabilities(true))

	srv.AddTool(mcp.NewTool("list_books",
		mcp.WithDescription("List all books owned by the authenticated user (id, title, author, page_count, current_page, parse_status)."),
	), mcpListBooks(app))

	srv.AddTool(mcp.NewTool("get_toc",
		mcp.WithDescription("Get the table of contents of a book as a JSON tree (title/page/level/children)."),
		mcp.WithString("book_id", mcp.Required(), mcp.Description("Book id from list_books.")),
	), mcpGetTOC(app))

	srv.AddTool(mcp.NewTool("get_section_text",
		mcp.WithDescription("Get the text of a TOC section. The section spans its start page up to (but excluding) the next same-or-higher-level TOC entry's page; pass start_page (default: section start) to page through long sections (~10 pages per response). Returns JSON with from_page/to_page/next_page (null when the section is exhausted), text with --- p.N --- separators, and is_partial/note when the section shares its start page with siblings."),
		mcp.WithString("book_id", mcp.Required(), mcp.Description("Book id from list_books.")),
		mcp.WithString("section_title", mcp.Required(), mcp.Description("TOC entry title; exact match first, then substring.")),
		mcp.WithNumber("start_page", mcp.Description("Page to start from (default: the section's start page).")),
	), mcpGetSectionText(app))

	srv.AddTool(mcp.NewTool("get_pages",
		mcp.WithDescription("Get text for an explicit page range [from_page, to_page] of a book, joined with --- p.N --- separators. The range is clamped to the book's page_count; responses over ~100KB are truncated with an explicit marker."),
		mcp.WithString("book_id", mcp.Required(), mcp.Description("Book id from list_books.")),
		mcp.WithNumber("from_page", mcp.Required(), mcp.Description("First page (1-based).")),
		mcp.WithNumber("to_page", mcp.Required(), mcp.Description("Last page (inclusive).")),
	), mcpGetPages(app))

	srv.AddTool(mcp.NewTool("search_in_book",
		mcp.WithDescription("Case-insensitive substring search over a book's stored page text. Returns up to 20 matching line snippets, at most 3 per page."),
		mcp.WithString("book_id", mcp.Required(), mcp.Description("Book id from list_books.")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Text to search for.")),
	), mcpSearchInBook(app))

	h := server.NewStreamableHTTPServer(srv)

	e.Router.POST("/mcp", func(re *core.RequestEvent) error {
		token := authTokenFromHTTPRequest(re.Request)
		if token == "" {
			return re.String(http.StatusUnauthorized, `{"error":"missing auth token"}`)
		}
		auth, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
		if err != nil {
			return re.String(http.StatusUnauthorized, `{"error":"invalid auth token"}`)
		}
		re.Request = re.Request.WithContext(context.WithValue(re.Request.Context(), mcpUserContextKey{}, auth.Id))
		h.ServeHTTP(re.Response, re.Request)
		return nil
	})
}
