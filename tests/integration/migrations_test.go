//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
