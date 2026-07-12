// Package migrations embeds the SQL migration files so they can be applied
// without filesystem access — in particular by the browser demo, which runs
// the real stores over SQLite and has no disk to read migrations from.
package migrations

import "embed"

// FS holds every migration file, ordered by name.
//
//go:embed *.sql
var FS embed.FS
