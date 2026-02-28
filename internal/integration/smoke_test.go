//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestPostgresMigrationsSmoke(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ISPO_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("ISPO_TEST_DATABASE_URL not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	// Clean schema so migrations are deterministic.
	if _, err := db.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	migrationsDir := filepath.Join(root, "db", "migrations")
	if _, err := os.Stat(migrationsDir); err != nil {
		t.Fatalf("migrations dir: %v", err)
	}

	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	// Critical tables/columns should exist after migrations.
	assertTableExists(t, ctx, db, "system_state")
	assertTableExists(t, ctx, db, "audit_logs")
	assertColumnExists(t, ctx, db, "audit_logs", "request_id")
	assertColumnExists(t, ctx, db, "audit_logs", "ip")
	assertColumnExists(t, ctx, db, "audit_logs", "user_agent")
}

func assertTableExists(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = $1
	)`, table).Scan(&exists)
	if err != nil {
		t.Fatalf("table exists query: %v", err)
	}
	if !exists {
		t.Fatalf("expected table %q to exist", table)
	}
}

func assertColumnExists(t *testing.T, ctx context.Context, db *sql.DB, table, column string) {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
	)`, table, column).Scan(&exists)
	if err != nil {
		t.Fatalf("column exists query: %v", err)
	}
	if !exists {
		t.Fatalf("expected column %q.%q to exist", table, column)
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 12; i++ {
		cand := filepath.Join(dir, "db", "migrations")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not locate repo root from %q", wd)
}
