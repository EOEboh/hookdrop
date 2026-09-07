package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// SPIKE SCAFFOLDING — not for merge as-is.
//
// Lets the existing test suite run unchanged against a remote libSQL server
// (Turso, or sqld locally) instead of a temp file, so the suite itself
// answers whether the driver swap is safe:
//
//	docker run -d -p 8081:8080 ghcr.io/tursodatabase/libsql-server
//	HOOKDROP_TEST_DB_URL=http://localhost:8081 go test ./...

// TestDSN returns the DSN a test should open. With HOOKDROP_TEST_DB_URL set it
// returns that URL; otherwise a fresh file inside the test's temp directory.
func TestDSN(tempDir string) string {
	if u := os.Getenv("HOOKDROP_TEST_DB_URL"); u != "" {
		return u
	}
	return filepath.Join(tempDir, "test.db")
}

// ResetForTest drops every table, so a shared remote database behaves like the
// fresh temp file each test would otherwise get. A no-op for local files,
// which are already per-test.
func (s *Store) ResetForTest() error {
	if !s.remote {
		return nil
	}
	rows, err := s.db.Query(
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, t := range tables {
		if _, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", t)); err != nil {
			return fmt.Errorf("drop %s: %w", t, err)
		}
	}
	return s.migrate()
}
