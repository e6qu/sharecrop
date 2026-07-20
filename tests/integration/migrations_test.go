//go:build integration

package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
)

func TestVerifyMigrationsCurrentDetectsPendingMigration(t *testing.T) {
	pool := newPool(t)
	migrationsDir := requireEnv(t, "SHARECROP_MIGRATIONS_DIR")
	if err := db.VerifyMigrationsCurrent(context.Background(), pool, migrationsDir); err != nil {
		t.Fatalf("current schema rejected: %v", err)
	}

	pendingDir := t.TempDir()
	pendingName := "999999_pending_identity_schema.sql"
	if err := os.WriteFile(filepath.Join(pendingDir, pendingName), []byte("select 1;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := db.VerifyMigrationsCurrent(context.Background(), pool, pendingDir)
	if err == nil || !strings.Contains(err.Error(), pendingName) {
		t.Fatalf("pending migration error = %v", err)
	}
}

func TestMigrateUpSerializesConcurrentStandaloneTasks(t *testing.T) {
	pool := newPool(t)
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		t.Fatal(err)
	}
	suffix := hex.EncodeToString(randomBytes)
	tableName := "migration_concurrency_probe_" + suffix
	migrationName := "999998_concurrency_probe_" + suffix + ".sql"
	migrationsDir := t.TempDir()
	migrationSQL := "create table " + tableName + " (run_count integer not null);\n" +
		"insert into " + tableName + " (run_count) values (1);\n"
	if err := os.WriteFile(filepath.Join(migrationsDir, migrationName), []byte(migrationSQL), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "drop table if exists "+tableName)
		_, _ = pool.Exec(context.Background(), "delete from schema_migrations where name = $1", migrationName)
	})

	start := make(chan struct{})
	errors := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			errors <- db.MigrateUp(context.Background(), pool, migrationsDir)
		}()
	}
	ready.Wait()
	close(start)

	for range 2 {
		if err := <-errors; err != nil {
			t.Fatalf("concurrent migration failed: %v", err)
		}
	}

	var runCount int
	if err := pool.QueryRow(context.Background(), "select run_count from "+tableName).Scan(&runCount); err != nil {
		t.Fatal(err)
	}
	if runCount != 1 {
		t.Fatalf("migration SQL ran %d times, want exactly once", runCount)
	}
}
