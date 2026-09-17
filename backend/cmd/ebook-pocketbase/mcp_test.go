package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestFlattenTOC(t *testing.T) {
	tree := []tocItem{
		{Title: "A", Page: 1, Level: 1, Children: []tocItem{
			{Title: "A.1", Page: 2, Level: 2},
			{Title: "A.2", Page: 5, Level: 2, Children: []tocItem{
				{Title: "A.2.1", Page: 6, Level: 3},
			}},
		}},
		{Title: "B", Page: 9, Level: 1},
	}
	flat := flattenTOC(tree)
	want := []string{"A", "A.1", "A.2", "A.2.1", "B"}
	if len(flat) != len(want) {
		t.Fatalf("flattenTOC len = %d, want %d", len(flat), len(want))
	}
	for i, title := range want {
		if flat[i].Title != title {
			t.Errorf("flat[%d] = %q, want %q", i, flat[i].Title, title)
		}
	}
}

func TestResolveSectionRange(t *testing.T) {
	// A p1 L1, A.1 p2 L2, A.2 p2 L2, B p5 L1, B.1 p7 L2, C p22 L1; page_count 25.
	flat := flattenTOC([]tocItem{
		{Title: "A", Page: 1, Level: 1, Children: []tocItem{
			{Title: "A.1", Page: 2, Level: 2},
			{Title: "A.2", Page: 2, Level: 2},
		}},
		{Title: "B", Page: 5, Level: 1, Children: []tocItem{
			{Title: "B.1", Page: 7, Level: 2},
		}},
		{Title: "C", Page: 22, Level: 1},
	})
	cases := []struct {
		name      string
		title     string
		from, to  int
		isPartial bool
	}{
		{"top-level stops at next same-level", "A", 1, 4, false},
		{"same-page sibling yields partial single page", "A.1", 2, 2, true},
		{"nested ends before parent section end", "A.2", 2, 4, false},
		{"long section spans to next top-level", "B", 5, 21, false},
		{"last section runs to page_count", "C", 22, 25, false},
	}
	for _, c := range cases {
		idx := findTOCIndex(flat, c.title)
		if idx < 0 {
			t.Fatalf("%s: not found", c.title)
		}
		from, to, isPartial := resolveSectionRange(flat, idx, 25)
		if from != c.from || to != c.to || isPartial != c.isPartial {
			t.Errorf("%s: got [%d,%d] partial=%v, want [%d,%d] partial=%v",
				c.name, from, to, isPartial, c.from, c.to, c.isPartial)
		}
	}
}

// withStaticDir points PUBLIC_DIR at a temp dir so registerRoutes mounts the
// GET /{path...} static catch-all exactly like production (where ../dist
// exists). Registration-order conflicts between /mcp and the catch-all only
// surface when this route exists — tests must not run without it.
func withStaticDir(t *testing.T) {
	t.Helper()
	t.Setenv("PUBLIC_DIR", t.TempDir())
}

// seedMCPTestApp boots a test app with the /mcp route mounted, one owner book
// (25 pages, multi-level TOC) plus a foreign user's book, and returns the app,
// base URL of a live test server, and the owner's auth token.
func seedMCPTestApp(t *testing.T) (string, string, string) {
	t.Helper()
	withStaticDir(t)
	app := newSchemaTestApp(t)

	owner, err := app.FindAuthRecordByEmail("users", "demo@e.co")
	if err != nil {
		t.Fatalf("find demo user: %v", err)
	}
	token, err := owner.NewAuthToken()
	if err != nil {
		t.Fatalf("auth token: %v", err)
	}

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	other := core.NewRecord(users)
	other.Set("email", "other@e.co")
	other.SetPassword("other1234")
	if err := app.SaveNoValidate(other); err != nil {
		t.Fatalf("save other user: %v", err)
	}

	books, err := app.FindCollectionByNameOrId("books")
	if err != nil {
		t.Fatal(err)
	}
	pages, err := app.FindCollectionByNameOrId("book_pages")
	if err != nil {
		t.Fatal(err)
	}

	toc := []tocItem{
		{Title: "Chapter 1", Page: 1, Level: 1, Children: []tocItem{
			{Title: "Section 1.1", Page: 2, Level: 2},
			{Title: "Section 1.2", Page: 2, Level: 2},
		}},
		{Title: "Chapter 2", Page: 5, Level: 1, Children: []tocItem{
			{Title: "Section 2.1", Page: 7, Level: 2},
		}},
		{Title: "Epilogue", Page: 22, Level: 1},
	}
	tocJSON, err := json.Marshal(toc)
	if err != nil {
		t.Fatal(err)
	}

	book := core.NewRecord(books)
	book.Set("user", owner.Id)
	book.Set("title", "Novel")
	book.Set("author", "Writer")
	book.Set("parse_status", "completed")
	book.Set("page_count", 25)
	book.Set("current_page", 3)
	book.Set("toc", string(tocJSON))
	if err := app.SaveNoValidate(book); err != nil {
		t.Fatalf("save book: %v", err)
	}
	for n := 1; n <= 25; n++ {
		text := fmt.Sprintf("Page %d body\nsecond line of page %d", n, n)
		if n == 8 {
			text = "golang concurrency patterns\ngolang routines\nother line"
		}
		if n == 20 {
			text = "golang channels\ngolang select\nunrelated line"
		}
		p := core.NewRecord(pages)
		p.Set("book", book.Id)
		p.Set("page_number", n)
		p.Set("text", text)
		if err := app.SaveNoValidate(p); err != nil {
			t.Fatalf("save page %d: %v", n, err)
		}
	}

	foreign := core.NewRecord(books)
	foreign.Set("user", other.Id)
	foreign.Set("title", "Foreign")
	foreign.Set("parse_status", "completed")
	foreign.Set("page_count", 2)
	if err := app.SaveNoValidate(foreign); err != nil {
		t.Fatalf("save foreign book: %v", err)
	}
	fp := core.NewRecord(pages)
	fp.Set("book", foreign.Id)
	fp.Set("page_number", 1)
	fp.Set("text", "secret foreign page")
	if err := app.SaveNoValidate(fp); err != nil {
		t.Fatalf("save foreign page: %v", err)
	}

	big := core.NewRecord(books)
	big.Set("user", owner.Id)
	big.Set("title", "Big")
	big.Set("parse_status", "completed")
	big.Set("page_count", 1)
	if err := app.SaveNoValidate(big); err != nil {
		t.Fatalf("save big book: %v", err)
	}
	bp := core.NewRecord(pages)
	bp.Set("book", big.Id)
	bp.Set("page_number", 1)
	bp.Set("text", strings.Repeat("x", 110*1024))
	if err := app.SaveNoValidate(bp); err != nil {
		t.Fatalf("save big page: %v", err)
	}

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts.URL, token, book.Id
}

func newMCPClient(t *testing.T, rawURL, token string) *client.Client {
	t.Helper()
	opts := []transport.StreamableHTTPCOption{}
	if token != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + token,
		}))
	}
	c, err := client.NewStreamableHttpClient(rawURL+"/mcp", opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "mcp-test", Version: "0.0.1"},
		},
	}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func callTool(t *testing.T, c *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res
}

func toolText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty tool result content")
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

func TestMCPAuthGate(t *testing.T) {
	rawURL, _, _ := seedMCPTestApp(t)

	cases := []struct {
		name   string
		header map[string]string
	}{
		{"no token", nil},
		{"invalid token", map[string]string{"Authorization": "Bearer bogus"}},
	}
	for _, c := range cases {
		req, err := http.NewRequest(http.MethodPost, rawURL+"/mcp", strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range c.header {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: POST /mcp → %d, want 401", c.name, resp.StatusCode)
		}
	}
}

func TestMCPTools(t *testing.T) {
	rawURL, token, bookID := seedMCPTestApp(t)
	c := newMCPClient(t, rawURL, token)

	t.Run("list_books returns only own books", func(t *testing.T) {
		res := callTool(t, c, "list_books", nil)
		if res.IsError {
			t.Fatalf("list_books error: %s", toolText(t, res))
		}
		var rows []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Author      string `json:"author"`
			PageCount   int    `json:"page_count"`
			CurrentPage int    `json:"current_page"`
			ParseStatus string `json:"parse_status"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("got %d books, want 2 (Novel + Big): %s", len(rows), toolText(t, res))
		}
		for _, row := range rows {
			if row.Title == "Foreign" {
				t.Fatal("list_books leaked another user's book")
			}
			if row.Title == "Novel" && (row.PageCount != 25 || row.CurrentPage != 3 || row.Author != "Writer" || row.ParseStatus != "completed") {
				t.Errorf("Novel row fields wrong: %+v", row)
			}
		}
	})

	t.Run("get_toc returns the JSON tree", func(t *testing.T) {
		res := callTool(t, c, "get_toc", map[string]any{"book_id": bookID})
		if res.IsError {
			t.Fatalf("get_toc error: %s", toolText(t, res))
		}
		var tree []tocItem
		if err := json.Unmarshal([]byte(toolText(t, res)), &tree); err != nil {
			t.Fatal(err)
		}
		if len(tree) != 3 || tree[0].Title != "Chapter 1" || len(tree[0].Children) != 2 {
			t.Fatalf("unexpected TOC: %s", toolText(t, res))
		}
	})

	t.Run("get_section_text chunks with cursor", func(t *testing.T) {
		type response struct {
			FromPage int    `json:"from_page"`
			ToPage   int    `json:"to_page"`
			NextPage *int   `json:"next_page"`
			Text     string `json:"text"`
		}
		res := callTool(t, c, "get_section_text", map[string]any{"book_id": bookID, "section_title": "Chapter 2"})
		if res.IsError {
			t.Fatalf("get_section_text error: %s", toolText(t, res))
		}
		var first response
		if err := json.Unmarshal([]byte(toolText(t, res)), &first); err != nil {
			t.Fatal(err)
		}
		if first.FromPage != 5 || first.ToPage != 14 || first.NextPage == nil || *first.NextPage != 15 {
			t.Fatalf("first chunk = %+v, want [5,14] next 15", first)
		}
		if !strings.Contains(first.Text, "--- p.5 ---") {
			t.Errorf("chunk missing page separator: %.80q", first.Text)
		}

		res = callTool(t, c, "get_section_text", map[string]any{"book_id": bookID, "section_title": "Chapter 2", "start_page": 15})
		var second response
		if err := json.Unmarshal([]byte(toolText(t, res)), &second); err != nil {
			t.Fatal(err)
		}
		if second.FromPage != 15 || second.ToPage != 21 || second.NextPage != nil {
			t.Fatalf("second chunk = %+v, want [15,21] next null", second)
		}
		if !strings.Contains(second.Text, "--- p.21 ---") || strings.Contains(second.Text, "--- p.22 ---") {
			t.Errorf("second chunk boundaries wrong: %.80q", second.Text)
		}
	})

	t.Run("get_section_text same-page section is partial, not empty", func(t *testing.T) {
		res := callTool(t, c, "get_section_text", map[string]any{"book_id": bookID, "section_title": "Section 1.1"})
		if res.IsError {
			t.Fatalf("get_section_text error: %s", toolText(t, res))
		}
		var resp struct {
			IsPartial bool   `json:"is_partial"`
			Note      string `json:"note"`
			FromPage  int    `json:"from_page"`
			ToPage    int    `json:"to_page"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.IsPartial || resp.Note == "" {
			t.Errorf("same-page section must be flagged partial with a note: %+v", resp)
		}
		if resp.FromPage != 2 || resp.ToPage != 2 || !strings.Contains(resp.Text, "Page 2 body") {
			t.Errorf("partial section must return its single page text: %+v", resp)
		}
	})

	t.Run("get_section_text fuzzy title match", func(t *testing.T) {
		res := callTool(t, c, "get_section_text", map[string]any{"book_id": bookID, "section_title": "1.2"})
		if res.IsError {
			t.Fatalf("get_section_text error: %s", toolText(t, res))
		}
		var resp struct {
			Title    string `json:"title"`
			FromPage int    `json:"from_page"`
			ToPage   int    `json:"to_page"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Title != "Section 1.2" || resp.FromPage != 2 || resp.ToPage != 4 {
			t.Errorf("fuzzy match = %+v, want Section 1.2 [2,4]", resp)
		}
	})

	t.Run("get_section_text last section runs to page_count", func(t *testing.T) {
		res := callTool(t, c, "get_section_text", map[string]any{"book_id": bookID, "section_title": "Epilogue"})
		var resp struct {
			FromPage int  `json:"from_page"`
			ToPage   int  `json:"to_page"`
			NextPage *int `json:"next_page"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.FromPage != 22 || resp.ToPage != 25 || resp.NextPage != nil {
			t.Errorf("Epilogue = %+v, want [22,25] next null", resp)
		}
	})

	t.Run("foreign book is rejected", func(t *testing.T) {
		// The foreign book id is unknown here on purpose: list must not leak
		// it, and any id must fail closed. Use the owner's book id handling as
		// control and a fabricated-but-valid-shape lookup via search on the
		// foreign title path is covered by the ownership filter itself.
		res := callTool(t, c, "get_toc", map[string]any{"book_id": "nonexistent"})
		if !res.IsError {
			t.Errorf("get_toc on unknown book must fail, got: %s", toolText(t, res))
		}
	})

	t.Run("get_pages joins with separators and clamps", func(t *testing.T) {
		res := callTool(t, c, "get_pages", map[string]any{"book_id": bookID, "from_page": 1, "to_page": 99})
		if res.IsError {
			t.Fatalf("get_pages error: %s", toolText(t, res))
		}
		var resp struct {
			FromPage int    `json:"from_page"`
			ToPage   int    `json:"to_page"`
			Text     string `json:"text"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.ToPage != 25 || !strings.Contains(resp.Text, "--- p.1 ---") || !strings.Contains(resp.Text, "--- p.25 ---") {
			t.Errorf("get_pages clamp wrong: %+v", resp)
		}
	})

	t.Run("get_pages marks truncation", func(t *testing.T) {
		res := callTool(t, c, "list_books", nil)
		var rows []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &rows); err != nil {
			t.Fatal(err)
		}
		var bigID string
		for _, row := range rows {
			if row.Title == "Big" {
				bigID = row.ID
			}
		}
		if bigID == "" {
			t.Fatal("Big book not listed")
		}
		res = callTool(t, c, "get_pages", map[string]any{"book_id": bigID, "from_page": 1, "to_page": 1})
		var resp struct {
			Truncated bool   `json:"truncated"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Truncated || !strings.Contains(resp.Text, "[truncated") || len(resp.Text) > mcpMaxTextBytes+200 {
			t.Errorf("truncation not marked: truncated=%v len=%d", resp.Truncated, len(resp.Text))
		}
	})

	t.Run("search_in_book finds hits with snippets", func(t *testing.T) {
		res := callTool(t, c, "search_in_book", map[string]any{"book_id": bookID, "query": "golang"})
		if res.IsError {
			t.Fatalf("search_in_book error: %s", toolText(t, res))
		}
		var hits []struct {
			PageNumber int      `json:"page_number"`
			Snippets   []string `json:"snippets"`
		}
		if err := json.Unmarshal([]byte(toolText(t, res)), &hits); err != nil {
			t.Fatal(err)
		}
		pages := map[int][]string{}
		for _, hit := range hits {
			pages[hit.PageNumber] = hit.Snippets
		}
		if len(pages[8]) != 2 || len(pages[20]) != 2 {
			t.Errorf("want pages 8 and 20 with 2 snippets each, got %+v", pages)
		}
		if _, ok := pages[1]; ok {
			t.Errorf("page 1 must not match: %+v", pages)
		}

		res = callTool(t, c, "search_in_book", map[string]any{"book_id": bookID, "query": "concurrency"})
		hits = nil
		if err := json.Unmarshal([]byte(toolText(t, res)), &hits); err != nil {
			t.Fatal(err)
		}
		if len(hits) != 1 || hits[0].PageNumber != 8 || !strings.Contains(hits[0].Snippets[0], "concurrency") {
			t.Errorf("concurrency search = %+v", hits)
		}
	})
}

// TestMCPForeignBookScoped pins the per-tool ownership filter: a valid token
// for user A used against a book owned by user B fails closed even when the
// foreign book id is known.
func TestMCPForeignBookScoped(t *testing.T) {
	withStaticDir(t)
	app := newSchemaTestApp(t)

	owner, err := app.FindAuthRecordByEmail("users", "demo@e.co")
	if err != nil {
		t.Fatalf("find demo user: %v", err)
	}
	token, err := owner.NewAuthToken()
	if err != nil {
		t.Fatalf("auth token: %v", err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	other := core.NewRecord(users)
	other.Set("email", "other@e.co")
	other.SetPassword("other1234")
	if err := app.SaveNoValidate(other); err != nil {
		t.Fatalf("save other user: %v", err)
	}
	books, err := app.FindCollectionByNameOrId("books")
	if err != nil {
		t.Fatal(err)
	}
	foreign := core.NewRecord(books)
	foreign.Set("user", other.Id)
	foreign.Set("title", "Foreign")
	foreign.Set("parse_status", "completed")
	foreign.Set("page_count", 1)
	if err := app.SaveNoValidate(foreign); err != nil {
		t.Fatalf("save foreign book: %v", err)
	}

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	c := newMCPClient(t, ts.URL, token)
	for _, name := range []string{"get_toc", "get_section_text", "get_pages", "search_in_book"} {
		args := map[string]any{"book_id": foreign.Id}
		switch name {
		case "get_section_text":
			args["section_title"] = "Chapter 1"
		case "get_pages":
			args["from_page"] = 1
			args["to_page"] = 1
		case "search_in_book":
			args["query"] = "secret"
		}
		res := callTool(t, c, name, args)
		if !res.IsError {
			t.Errorf("%s on a foreign book must fail closed, got: %s", name, toolText(t, res))
		}
	}
}
