package ledger

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*/*.sql
var migrationsFS embed.FS

type DBDriver string

const (
	DBSQLite   DBDriver = "sqlite"
	DBPostgres DBDriver = "postgres"
)

// Migrate applies embedded migrations in order, recording each migration in a migrations table.
// This is intentionally "small and boring": sequential SQL files + a single table.
func Migrate(db *sql.DB, driver DBDriver) error {
	if db == nil {
		return fmt.Errorf("missing db")
	}
	dir, table, err := migrationConfig(driver)
	if err != nil {
		return err
	}
	if err := ensureMigrationsTable(db, driver, table); err != nil {
		return err
	}

	files, err := listMigrationFiles(dir)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".sql")
		contents, err := migrationsFS.ReadFile(file)
		if err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		applied, err := tryInsertMigration(tx, driver, table, version, now)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if !applied {
			_ = tx.Rollback()
			continue
		}

		if _, err := tx.Exec(string(contents)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func migrationConfig(driver DBDriver) (dir string, table string, err error) {
	switch driver {
	case DBSQLite:
		return "migrations/sqlite", "schema_migrations", nil
	case DBPostgres:
		return "migrations/postgres", "relia_schema_migrations", nil
	default:
		return "", "", fmt.Errorf("unsupported db driver: %s", driver)
	}
}

func ensureMigrationsTable(db *sql.DB, driver DBDriver, table string) error {
	switch driver {
	case DBSQLite:
		_, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
)`, table))
		return err
	case DBPostgres:
		_, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL
)`, table))
		return err
	default:
		return fmt.Errorf("unsupported db driver: %s", driver)
	}
}

func tryInsertMigration(tx *sql.Tx, driver DBDriver, table string, version string, now time.Time) (bool, error) {
	var (
		res sql.Result
		err error
	)
	switch driver {
	case DBSQLite:
		res, err = tx.Exec(fmt.Sprintf(`INSERT INTO %s(version, applied_at) VALUES(?, ?) ON CONFLICT(version) DO NOTHING`, table), version, now.Format(time.RFC3339))
	case DBPostgres:
		res, err = tx.Exec(fmt.Sprintf(`INSERT INTO %s(version, applied_at) VALUES($1, $2) ON CONFLICT(version) DO NOTHING`, table), version, now)
	default:
		return false, fmt.Errorf("unsupported db driver: %s", driver)
	}
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func listMigrationFiles(dir string) ([]string, error) {
	entries, err := migrationsFS.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}
