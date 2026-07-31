package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func textField(name string, required bool, max int) *core.TextField {
	return &core.TextField{Name: name, Required: required, Max: max}
}

func numberField(name string, required bool) *core.NumberField {
	return &core.NumberField{Name: name, Required: required}
}

func relationField(name, collectionID string) *core.RelationField {
	return &core.RelationField{Name: name, CollectionId: collectionID, Required: true, MaxSelect: 1}
}

func timestamps(collection *core.Collection) {
	collection.Fields.Add(
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
}

func findOrCreateCollection(app core.App, name string, create func() *core.Collection) (*core.Collection, error) {
	collection, err := app.FindCollectionByNameOrId(name)
	if err == nil {
		return collection, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	collection = create()
	if err := app.Save(collection); err != nil {
		return nil, fmt.Errorf("create collection %s: %w", name, err)
	}
	return collection, nil
}

func ensureSchema(app core.App) error {
	users, err := findOrCreateCollection(app, "users", func() *core.Collection {
		collection := core.NewAuthCollection("users", "_pb_users_auth_")
		ownerRule := "id = @request.auth.id"
		collection.ListRule = types.Pointer(ownerRule)
		collection.ViewRule = types.Pointer(ownerRule)
		collection.CreateRule = types.Pointer("")
		collection.UpdateRule = types.Pointer(ownerRule)
		collection.DeleteRule = types.Pointer(ownerRule)
		collection.AuthRule = types.Pointer("")
		collection.Fields.Add(textField("name", true, 120))
		timestamps(collection)
		return collection
	})
	if err != nil {
		return err
	}

	books, err := findOrCreateCollection(app, "books", func() *core.Collection {
		collection := core.NewBaseCollection("books")
		collection.ListRule = types.Pointer("user = @request.auth.id")
		collection.ViewRule = types.Pointer("user = @request.auth.id")
		collection.CreateRule = types.Pointer(`@request.auth.id != "" && user = @request.auth.id`)
		collection.UpdateRule = types.Pointer("user = @request.auth.id")
		collection.DeleteRule = types.Pointer("user = @request.auth.id")
		collection.Fields.Add(
			relationField("user", users.Id),
			textField("title", true, 240),
			textField("author", false, 180),
			&core.EditorField{Name: "description", MaxSize: 20000},
			&core.FileField{
				Name: "file", Required: true, MaxSelect: 1, MaxSize: 104857600,
				MimeTypes: []string{"application/pdf", "application/epub+zip", "application/x-mobipocket-ebook"},
			},
			numberField("page_count", false),
			&core.SelectField{
				Name: "parse_status", Required: true, MaxSelect: 1,
				Values: []string{"pending", "processing", "completed", "failed"},
			},
			textField("parse_error", false, 1000),
			numberField("current_page", false),
			&core.JSONField{Name: "toc", MaxSize: 200000},
			&core.DateField{Name: "last_read_at"},
		)
		timestamps(collection)
		return collection
	})
	if err != nil {
		return err
	}

	definitions := []struct {
		name   string
		create func() *core.Collection
	}{
		{"book_pages", func() *core.Collection {
			collection := core.NewBaseCollection("book_pages")
			collection.ListRule = types.Pointer("book.user = @request.auth.id")
			collection.ViewRule = types.Pointer("book.user = @request.auth.id")
			collection.Fields.Add(
				relationField("book", books.Id),
				numberField("page_number", true),
				&core.EditorField{Name: "text", MaxSize: 200000},
				numberField("width", false),
				numberField("height", false),
			)
			timestamps(collection)
			return collection
		}},
		{"bookmarks", func() *core.Collection {
			collection := ownedCollection("bookmarks")
			collection.Fields.Add(
				relationField("book", books.Id), relationField("user", users.Id),
				numberField("page_number", true), textField("title", true, 200), textField("note", false, 1000),
			)
			timestamps(collection)
			return collection
		}},
		{"notes", func() *core.Collection {
			collection := ownedCollection("notes")
			collection.Fields.Add(
				relationField("book", books.Id), relationField("user", users.Id),
				numberField("page_number", true), &core.EditorField{Name: "content", Required: true, MaxSize: 50000},
			)
			timestamps(collection)
			return collection
		}},
		{"reading_records", func() *core.Collection {
			collection := ownedCollection("reading_records")
			collection.Fields.Add(
				relationField("book", books.Id), relationField("user", users.Id),
				numberField("page_number", true), numberField("progress", false), numberField("read_seconds", false),
			)
			timestamps(collection)
			return collection
		}},
	}
	for _, definition := range definitions {
		if _, err := findOrCreateCollection(app, definition.name, definition.create); err != nil {
			return err
		}
	}

	if err := ensureAuthRecord(app, core.CollectionNameSuperusers, envOr("PB_SUPERUSER_EMAIL", "admin@e.co"), envOr("PB_SUPERUSER_PASSWORD", "admin123"), ""); err != nil {
		return fmt.Errorf("create default superuser: %w", err)
	}
	if err := ensureAuthRecord(app, "users", envOr("APP_USER_EMAIL", "demo@e.co"), envOr("APP_USER_PASSWORD", "demo1234"), envOr("APP_USER_NAME", "Reader Demo User")); err != nil {
		return fmt.Errorf("create default user: %w", err)
	}
	return nil
}

func ownedCollection(name string) *core.Collection {
	collection := core.NewBaseCollection(name)
	collection.ListRule = types.Pointer("user = @request.auth.id")
	collection.ViewRule = types.Pointer("user = @request.auth.id")
	collection.CreateRule = types.Pointer(`@request.auth.id != "" && user = @request.auth.id && book.user = @request.auth.id`)
	collection.UpdateRule = types.Pointer("user = @request.auth.id")
	collection.DeleteRule = types.Pointer("user = @request.auth.id")
	return collection
}

func ensureAuthRecord(app core.App, collectionName, email, password, name string) error {
	if _, err := app.FindAuthRecordByEmail(collectionName, email); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.SetPassword(password)
	record.SetVerified(true)
	if name != "" {
		record.Set("name", name)
	}
	return app.Save(record)
}
