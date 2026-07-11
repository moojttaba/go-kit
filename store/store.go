// Package store provides a pgx connection pool with startup retry/backoff and a
// forward-only migration runner driven by an embedded fs.FS. Each service
// supplies its own migrations directory; go-kit only runs them.
package store

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open creates a pool and pings it, retrying with exponential backoff (capped)
// for up to ~30s so a service may start slightly ahead of its database.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}

	const maxBackoff = 5 * time.Second
	backoff := 250 * time.Millisecond
	deadline := time.Now().Add(30 * time.Second)

	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		pingErr := pool.Ping(pingCtx)
		cancel()
		if pingErr == nil {
			return pool, nil
		}
		if time.Now().After(deadline) {
			pool.Close()
			return nil, fmt.Errorf("db not ready after %d attempts: %w", attempt, pingErr)
		}
		slog.Warn("db not ready, retrying", "attempt", attempt, "backoff", backoff.String(), "err", pingErr)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// Migrate applies every *.sql file at the root of fsys (lexical order) that has
// not run yet, recording applied versions in schema_migrations. Each file runs
// in its own transaction. Migrations are forward-only — never edit a shipped
// file; add a new one.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}

	names, err := sqlFiles(fsys)
	if err != nil {
		return err
	}

	for _, name := range names {
		var applied bool
		if err := pool.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name,
		).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		sqlBytes, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// sqlFiles returns the sorted names of *.sql files at the root of fsys.
func sqlFiles(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
