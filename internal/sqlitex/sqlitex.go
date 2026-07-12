// Package sqlitex opens a SQLite database (backed by ncruces/go-sqlite3) with
// the Postgres-compatibility functions the domain stores rely on. It is kept
// separate from internal/db so that production, which imports internal/db but
// never this package, links zero ncruces packages.
package sqlitex

import (
	"database/sql"
	"encoding/base64"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Open opens a SQLite database at the given DSN with the compatibility
// functions registered on every connection.
func Open(dataSourceName string) (*sql.DB, error) {
	return driver.Open(dataSourceName, registerCompatibilityFunctions)
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
