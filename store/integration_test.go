//go:build integration

package store

import (
	"context"
	"os"
	"testing"
	"testing/fstest"
)

// Integration test against a real Postgres. Set TEST_DATABASE_URL (CI provides a
// postgres service container). Behind the `integration` build tag + skipped when
// unset, so it never runs in the default `go test ./...`.
func TestOpenMigrate_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run integration tests")
	}
	ctx := context.Background()

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()

	// Repeatable run.
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS widgets, schema_migrations`); err != nil {
		t.Fatalf("reset: %v", err)
	}

	migrations := fstest.MapFS{
		"0001_init.sql": {Data: []byte(`CREATE TABLE widgets (id serial PRIMARY KEY, name text NOT NULL);`)},
		"0002_more.sql": {Data: []byte(`ALTER TABLE widgets ADD COLUMN qty int NOT NULL DEFAULT 0;`)},
	}
	if err := Migrate(ctx, pool, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	// Idempotent: a second run applies nothing new and must not error.
	if err := Migrate(ctx, pool, migrations); err != nil {
		t.Fatalf("Migrate (rerun): %v", err)
	}

	var applied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if applied != 2 {
		t.Fatalf("applied migrations = %d, want 2", applied)
	}

	// The migrated table (incl. the column from 0002) is usable.
	if _, err := pool.Exec(ctx, `INSERT INTO widgets (name, qty) VALUES ('x', 3)`); err != nil {
		t.Fatalf("insert into migrated table: %v", err)
	}
}
