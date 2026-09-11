package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// TestPageRouteAuthGate pins the shared gate semantics for the three
// /api/books/{id}/pages/{page}/* routes after their boilerplate was
// consolidated into authorizedBookPage: missing/invalid token → 401, unknown
// book → 404, foreign book → 403, bad page → 400, and (for image/html)
// unfinished parsing → 400.
func TestPageRouteAuthGate(t *testing.T) {
	app := newSchemaTestApp(t)

	owner, err := app.FindAuthRecordByEmail("users", "demo@e.co")
	if err != nil {
		t.Fatalf("find demo user: %v", err)
	}
	rawToken, err := owner.NewAuthToken()
	if err != nil {
		t.Fatalf("auth token: %v", err)
	}
	token := "Bearer " + rawToken

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
	newBook := func(userID, status string) string {
		b := core.NewRecord(books)
		b.Set("user", userID)
		b.Set("title", "T")
		b.Set("parse_status", status)
		if err := app.SaveNoValidate(b); err != nil {
			t.Fatalf("save book: %v", err)
		}
		return b.Id
	}
	bookID := newBook(owner.Id, "completed")
	foreignBookID := newBook(other.Id, "completed")
	pendingBookID := newBook(owner.Id, "pending")

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

	serve := func(url, authHeader string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, url, nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		mux.ServeHTTP(rec, req)
		return rec.Result().StatusCode
	}

	suffixes := []string{"image", "illustrations", "html"}
	for _, suffix := range suffixes {
		page1 := "/api/books/" + bookID + "/pages/1/" + suffix
		cases := []struct {
			name string
			url  string
			tok  string
			want int
		}{
			{"missing token", page1, "", http.StatusUnauthorized},
			{"invalid token", page1, "bogus", http.StatusUnauthorized},
			{"unknown book", "/api/books/missing/pages/1/" + suffix, token, http.StatusNotFound},
			{"foreign book", "/api/books/" + foreignBookID + "/pages/1/" + suffix, token, http.StatusForbidden},
			{"bad page", "/api/books/" + bookID + "/pages/abc/" + suffix, token, http.StatusBadRequest},
		}
		if suffix != "illustrations" {
			cases = append(cases, struct {
				name string
				url  string
				tok  string
				want int
			}{"parsing not completed", "/api/books/" + pendingBookID + "/pages/1/" + suffix, token, http.StatusBadRequest})
		}
		for _, c := range cases {
			if got := serve(c.url, c.tok); got != c.want {
				t.Errorf("%s: GET %s → %d, want %d", c.name, c.url, got, c.want)
			}
		}
	}
}
