package wasmseed

import (
	"context"
	"os"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/sqlitex"
)

// TestSeedOnSQLite seeds the demo scenario over the real stores backed by
// SQLite (the same path the browser demo uses) and confirms it succeeds end to
// end, exercising every service the seed touches.
func TestSeedOnSQLite(t *testing.T) {
	ctx := context.Background()
	handle, err := sqlitex.Open("file:" + t.TempDir() + "/seed.db?_pragma=foreign_keys(off)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	if err := db.MigrateUpSQLite(ctx, handle, os.DirFS("../../migrations")); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}

	secretResult := auth.NewAccessTokenSecret("seed-test-secret-0123456789abcdef0123456789abcdef")
	secret, matched := secretResult.(auth.AccessTokenSecretAccepted)
	if !matched {
		t.Fatalf("access token secret rejected")
	}

	stores := StoresFromHandle(db.NewSQLite(handle))
	result := Seed(ctx, secret.Value, stores)
	if result.Err != "" {
		t.Fatalf("seed failed: %s", result.Err)
	}
	if result.AdminRefreshToken == nil {
		t.Fatalf("seed did not return an admin refresh token")
	}

	// The seed writes mara (the admin) a submission-review notification, so the
	// notification store should list at least one for her.
	page, ok := core.NewPage(50, 0).(core.PageAccepted)
	if !ok {
		t.Fatalf("page rejected")
	}
	listed, matched := db.NewNotificationStoreFromHandle(db.NewSQLite(handle)).
		List(ctx, result.AdminUserID, page.Value).(notification.ListStoreAccepted)
	if !matched {
		t.Fatalf("list admin notifications rejected")
	}
	if len(listed.Values) == 0 {
		t.Fatalf("seed did not create the admin's inbox notification")
	}
}
