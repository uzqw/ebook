package main

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestEnsureSchemaInitializesFreshDatabase(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	t.Setenv("PB_SUPERUSER_EMAIL", "admin@e.co")
	t.Setenv("PB_SUPERUSER_PASSWORD", "admin123")
	t.Setenv("APP_USER_EMAIL", "demo@e.co")
	t.Setenv("APP_USER_PASSWORD", "demo1234")
	t.Setenv("APP_USER_NAME", "Reader Demo User")

	if err := ensureSchema(app); err != nil {
		t.Fatal(err)
	}
	if err := ensureSchema(app); err != nil {
		t.Fatalf("second initialization: %v", err)
	}
	for _, name := range []string{"users", "books", "book_pages", "bookmarks", "notes", "reading_records"} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("collection %s: %v", name, err)
		}
	}
	if _, err := app.FindAuthRecordByEmail("users", "demo@e.co"); err != nil {
		t.Errorf("demo user: %v", err)
	}
	if _, err := app.FindAuthRecordByEmail("_superusers", "admin@e.co"); err != nil {
		t.Errorf("superuser: %v", err)
	}
}
