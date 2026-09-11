package main

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newSchemaTestApp builds a TestApp with the project schema applied and routes
// registered, mirroring main() without the pdfium service (nil is fine: the
// exercised routes don't reach it).
func newSchemaTestApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	t.Setenv("PB_SUPERUSER_EMAIL", "admin@e.co")
	t.Setenv("PB_SUPERUSER_PASSWORD", "admin123")
	t.Setenv("APP_USER_EMAIL", "demo@e.co")
	t.Setenv("APP_USER_PASSWORD", "demo1234")

	if err := ensureSchema(app); err != nil {
		t.Fatal(err)
	}
	registerHooks(app, nil)
	registerRoutes(app, nil)
	return app
}

// TestHealthRoute pins the contract the frontend heartbeat and `task status`
// rely on: GET /api/health answers 2xx. PocketBase's built-in health API
// provides the route; this guards against regression if it is ever removed
// or shadowed.
func TestHealthRoute(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "health probe returns 200",
		Method:          http.MethodGet,
		URL:             "/api/health",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"code":200`},
		TestAppFactory:  newSchemaTestApp,
	}
	scenario.Test(t)
}

// TestDeleteBookCleansChildrenOnce verifies that deleting a book via the API
// removes all child records exactly once — the cleanup must be bound to a
// single hook, not duplicated across OnRecordDeleteRequest and OnRecordDelete.
func TestDeleteBookCleansChildrenOnce(t *testing.T) {
	var bookID, token string

	scenario := tests.ApiScenario{
		Name:   "deleting a book removes its child records",
		Method: http.MethodDelete,
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := newSchemaTestApp(t)

			user, err := app.FindAuthRecordByEmail("users", "demo@e.co")
			if err != nil {
				t.Fatalf("find demo user: %v", err)
			}
			token, err = user.NewAuthToken()
			if err != nil {
				t.Fatalf("auth token: %v", err)
			}

			books, err := app.FindCollectionByNameOrId("books")
			if err != nil {
				t.Fatal(err)
			}
			book := core.NewRecord(books)
			book.Set("user", user.Id)
			book.Set("title", "T")
			book.Set("parse_status", "completed")
			if err := app.SaveNoValidate(book); err != nil {
				t.Fatalf("save book: %v", err)
			}
			bookID = book.Id

			childFields := map[string]map[string]any{
				"book_pages":      {"page_number": 1},
				"bookmarks":       {"page_number": 1, "title": "b"},
				"notes":           {"page_number": 1, "content": "n"},
				"reading_records": {"page_number": 1, "progress": 0.5},
			}
			for collName, fields := range childFields {
				coll, err := app.FindCollectionByNameOrId(collName)
				if err != nil {
					t.Fatal(err)
				}
				rec := core.NewRecord(coll)
				rec.Set("book", book.Id)
				if collName != "book_pages" {
					rec.Set("user", user.Id)
				}
				for k, v := range fields {
					rec.Set(k, v)
				}
				if err := app.SaveNoValidate(rec); err != nil {
					t.Fatalf("save %s: %v", collName, err)
				}
			}
			return app
		},
		ExpectedStatus: http.StatusNoContent,
		ExpectedEvents: map[string]int{
			// book + 4 children, each deleted exactly once
			"OnModelDelete":             5,
			"OnModelDeleteExecute":      5,
			"OnModelAfterDeleteSuccess": 5,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			for _, coll := range []string{"book_pages", "bookmarks", "notes", "reading_records"} {
				rows, err := app.FindRecordsByFilter(coll, `book = "`+bookID+`"`, "", 0, 0)
				if err != nil {
					t.Fatalf("query %s: %v", coll, err)
				}
				if len(rows) != 0 {
					t.Errorf("%s: %d child records left after book delete", coll, len(rows))
				}
			}
		},
	}
	// ApiScenario reads URL/Headers when the request executes — after
	// TestAppFactory has run — so resolve them lazily inside the factory.
	factory := scenario.TestAppFactory
	scenario.TestAppFactory = func(tb testing.TB) *tests.TestApp {
		app := factory(tb)
		scenario.URL = "/api/collections/books/records/" + bookID
		scenario.Headers = map[string]string{"Authorization": token}
		return app
	}
	scenario.Test(t)
}
