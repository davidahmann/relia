package ledger

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateSQLiteIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, DBSQLite); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Migrate(db, DBSQLite); err != nil {
		t.Fatalf("migrate second: %v", err)
	}

	// Ensure the outbox table exists.
	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='slack_outbox'`).Scan(&name); err != nil {
		t.Fatalf("expected slack_outbox table: %v", err)
	}
	if name != "slack_outbox" {
		t.Fatalf("unexpected table name: %s", name)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected at least 2 migrations applied, got %d", count)
	}
}

func TestMigrationHelpers(t *testing.T) {
	if _, _, err := migrationConfig(DBPostgres); err != nil {
		t.Fatalf("expected postgres config, got %v", err)
	}
	if _, _, err := migrationConfig(DBDriver("nope")); err == nil {
		t.Fatalf("expected error for unsupported driver")
	}

	if err := ensureMigrationsTable(&sql.DB{}, DBDriver("nope"), "t"); err == nil {
		t.Fatalf("expected error for unsupported driver")
	}

	if _, err := listMigrationFiles("migrations/sqlite"); err != nil {
		t.Fatalf("list migrations: %v", err)
	}
}
