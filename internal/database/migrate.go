package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed all:migrations
var migrationsFS embed.FS

// migrate applies every embedded migration that has not been recorded yet.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY, applied_at INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var seen int
		if err = db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&seen); err != nil {
			return err
		}
		if seen > 0 {
			continue
		}

		body, e := migrationsFS.ReadFile("migrations/" + name)
		if e != nil {
			return e
		}

		tx, e := db.Begin()
		if e != nil {
			return e
		}
		if _, e = tx.Exec(string(body)); e != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, e)
		}
		if _, e = tx.Exec(`INSERT INTO schema_migrations (name, applied_at) VALUES (?, unixepoch())`, name); e != nil {
			_ = tx.Rollback()
			return e
		}
		if e = tx.Commit(); e != nil {
			return e
		}
	}

	return nil
}
