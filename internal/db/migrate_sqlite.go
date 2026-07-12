package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// MigrateUpSQLite applies the Postgres migrations to a SQLite database for the
// browser demo, translating each to the SQLite dialect. The migrations are read
// from an fs.FS (a real directory in tests, an embedded FS in the browser).
// Statements are split on ";" because database/sql executes one statement per
// call (the migrations are DDL with no embedded semicolons).
func MigrateUpSQLite(ctx context.Context, handle *sql.DB, migrations fs.FS) error {
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".sql" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	if err := execSQLiteStatements(ctx, handle, translateSQLiteDDL(`
		create table if not exists schema_migrations (
			name text primary key,
			applied_at timestamptz not null default now()
		)
	`)); err != nil {
		return err
	}

	for _, name := range names {
		var existing string
		err := handle.QueryRowContext(ctx, "select name from schema_migrations where name = $1", name).Scan(&existing)
		if err == nil {
			continue
		}

		sqlBytes, err := fs.ReadFile(migrations, name)
		if err != nil {
			return err
		}

		if err := execSQLiteStatements(ctx, handle, translateSQLiteDDL(string(sqlBytes))); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}

		if _, err := handle.ExecContext(ctx, "insert into schema_migrations (name) values ($1)", name); err != nil {
			return err
		}
	}

	return nil
}

func execSQLiteStatements(ctx context.Context, handle *sql.DB, script string) error {
	for _, statement := range strings.Split(script, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if index, converted := rewriteSQLiteConstraintToIndex(statement); converted {
			if _, err := handle.ExecContext(ctx, index); err != nil {
				return fmt.Errorf("statement %q: %w", strings.TrimSpace(index), err)
			}
			continue
		}
		if sqliteUnsupportedDDL(statement) {
			continue
		}
		for _, single := range expandSQLiteAddColumns(statement) {
			if _, err := handle.ExecContext(ctx, single); err != nil {
				if sqliteIsAddColumn(single) && strings.Contains(err.Error(), "duplicate column name") {
					continue
				}
				return fmt.Errorf("statement %q: %w", strings.TrimSpace(single), err)
			}
		}
	}
	return nil
}
