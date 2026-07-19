package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MigrateUp(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	names, err := migrationNames(migrationsDir)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		create table if not exists schema_migrations (
			name text primary key,
			applied_at timestamptz not null default now()
		)
	`)
	if err != nil {
		return err
	}

	for _, name := range names {
		var existing string
		err := tx.QueryRow(ctx, "select name from schema_migrations where name = $1", name).Scan(&existing)
		if err == nil {
			continue
		}

		path := filepath.Join(migrationsDir, name)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, "insert into schema_migrations (name) values ($1)", name); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func VerifyMigrationsCurrent(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	names, err := migrationNames(migrationsDir)
	if err != nil {
		return err
	}
	rows, err := pool.Query(ctx, "select name from schema_migrations")
	if err != nil {
		return fmt.Errorf("read applied database migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]struct{}, len(names))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan applied database migration: %w", err)
		}
		applied[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read applied database migrations: %w", err)
	}
	for _, name := range names {
		if _, ok := applied[name]; !ok {
			return fmt.Errorf("database schema is missing migration %s", name)
		}
	}
	return nil
}

func migrationNames(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
