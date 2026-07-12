package sqlitex

import (
	"context"
	"testing"

	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

// TestSnapshotRestore confirms a database serialized by Snapshot can be restored
// into a fresh database by Restore — the browser persistence round-trip.
func TestSnapshotRestore(t *testing.T) {
	ctx := context.Background()

	source, err := Open("file:/snap-source?vfs=memdb")
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer func() { _ = source.Close() }()
	source.SetMaxOpenConns(1)

	if _, err := source.ExecContext(ctx, "create table t(id text, n integer)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := source.ExecContext(ctx, "insert into t values('a', 10), ('b', 32)"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	snapshot, err := Snapshot(ctx, source)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snapshot) == 0 {
		t.Fatalf("snapshot is empty")
	}

	target, err := Open("file:/snap-target?vfs=memdb")
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	defer func() { _ = target.Close() }()
	target.SetMaxOpenConns(1)

	if err := Restore(ctx, target, snapshot); err != nil {
		t.Fatalf("restore: %v", err)
	}

	var total int
	if err := target.QueryRowContext(ctx, "select sum(n) from t").Scan(&total); err != nil {
		t.Fatalf("query restored: %v", err)
	}
	if total != 42 {
		t.Fatalf("restored sum = %d, want 42", total)
	}

	// The encode() compatibility function is registered on the restored handle.
	var encoded string
	if err := target.QueryRowContext(ctx, "select encode(?, 'base64')", []byte("hi")).Scan(&encoded); err != nil {
		t.Fatalf("encode on restored handle: %v", err)
	}
	if encoded != "aGk=" {
		t.Fatalf("encode = %q, want aGk=", encoded)
	}
}

// TestLoadSnapshotBeforeOpen confirms the browser's persistence path: a snapshot
// pre-loaded into a shared memdb (before opening) is visible to the opened
// handle, so a reloaded page restores its state.
func TestLoadSnapshotBeforeOpen(t *testing.T) {
	ctx := context.Background()

	source, err := Open("file:/reload-source?vfs=memdb")
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer func() { _ = source.Close() }()
	if _, err := source.ExecContext(ctx, "create table t(n integer)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := source.ExecContext(ctx, "insert into t values(11),(31)"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	snapshot, err := Snapshot(ctx, source)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	// Simulate a page reload: pre-load the snapshot into a fresh shared memdb
	// name, then open a handle against it.
	LoadSnapshot("reload-target", snapshot)
	reloaded, err := Open("file:/reload-target?vfs=memdb")
	if err != nil {
		t.Fatalf("open reloaded: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	var total int
	if err := reloaded.QueryRowContext(ctx, "select sum(n) from t").Scan(&total); err != nil {
		t.Fatalf("query reloaded: %v", err)
	}
	if total != 42 {
		t.Fatalf("reloaded sum = %d, want 42", total)
	}
}
