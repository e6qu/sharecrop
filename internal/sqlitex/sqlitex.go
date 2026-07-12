// Package sqlitex opens a SQLite database (backed by ncruces/go-sqlite3) with
// the Postgres-compatibility functions the domain stores rely on. It is kept
// separate from internal/db so that production, which imports internal/db but
// never this package, links zero ncruces packages.
package sqlitex

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/serdes"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

// LoadSnapshot pre-loads a snapshot into the named shared memdb database before
// it is opened, so every pooled connection sees the restored data. The name is
// the memdb database name without the leading "/" (e.g. Open a DSN of
// "file:/demo?vfs=memdb" after LoadSnapshot("demo", ...)). Call before Open.
func LoadSnapshot(name string, snapshot []byte) {
	memdb.Create(name, snapshot)
}

// Open opens a SQLite database at the given DSN with the compatibility
// functions registered on every connection.
func Open(dataSourceName string) (*sql.DB, error) {
	return driver.Open(dataSourceName, registerCompatibilityFunctions)
}

// Snapshot serializes the main database to a byte slice (via SQLite's serialize
// API), so the browser demo can persist it to browser storage and restore it on
// the next page load.
func Snapshot(ctx context.Context, handle *sql.DB) ([]byte, error) {
	conn, err := handle.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	var snapshot []byte
	err = conn.Raw(func(driverConn any) error {
		raw, matched := driverConn.(interface{ Raw() *sqlite3.Conn })
		if !matched {
			return fmt.Errorf("sqlitex: driver connection does not expose a raw sqlite3.Conn")
		}
		bytes, serializeErr := serdes.Serialize(raw.Raw(), "main")
		snapshot = bytes
		return serializeErr
	})
	return snapshot, err
}

// Restore replaces the main database's contents with a snapshot produced by
// Snapshot. The handle should use a single connection (SetMaxOpenConns(1)) so
// every query sees the restored data.
func Restore(ctx context.Context, handle *sql.DB, snapshot []byte) error {
	conn, err := handle.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	return conn.Raw(func(driverConn any) error {
		raw, matched := driverConn.(interface{ Raw() *sqlite3.Conn })
		if !matched {
			return fmt.Errorf("sqlitex: driver connection does not expose a raw sqlite3.Conn")
		}
		return serdes.Deserialize(raw.Raw(), "main", snapshot)
	})
}

func registerCompatibilityFunctions(conn *sqlite3.Conn) error {
	// encode(blob, 'base64') — Postgres has this builtin, SQLite does not. The
	// stores use it to inline attachment content as base64 in aggregated JSON.
	return conn.CreateFunction("encode", 2, sqlite3.DETERMINISTIC, encodeBase64)
}

func encodeBase64(ctx sqlite3.Context, arg ...sqlite3.Value) {
	content := arg[0].Blob(nil)
	ctx.ResultText(base64.StdEncoding.EncodeToString(content))
}
