package db

import (
	"regexp"
	"strings"
)

// The single RFC3339 form both DB-side (strftime) and Go-provided (RFC3339Nano
// write) timestamps share, so mixed-source values sort consistently and
// ncruces decodes them back into time.Time (given a time-ish column type).
const sqliteNowExpr = `strftime('%Y-%m-%dT%H:%M:%fZ','now')`

var (
	// A Postgres type cast such as id::text or $6::jsonb. SQLite is dynamically
	// typed, so the cast is simply dropped.
	sqliteCastPattern = regexp.MustCompile(`::[a-zA-Z]+`)
	// Row locking has no SQLite equivalent; SQLite serializes writers, so
	// "for update" is a no-op and must be removed to parse.
	sqliteForUpdatePattern = regexp.MustCompile(`(?i)\s+for update`)
	// now() as a function call (not a column named "now").
	sqliteNowPattern = regexp.MustCompile(`(?i)\bnow\(\)`)
	// Postgres $N placeholders bind by number. SQLite treats $1 as a named
	// parameter numbered by appearance, so "$3 ... $1" would bind wrong;
	// rewriting to ?N restores explicit by-number binding.
	sqlitePlaceholder = regexp.MustCompile(`\$(\d+)`)
	// SQL line comments are stripped before splitting DDL on ";", because a
	// comment may itself contain a semicolon and would otherwise split badly.
	sqliteLineComment = regexp.MustCompile(`--[^\n]*`)

	// DDL type names. timestamptz is intentionally kept: SQLite accepts the
	// unknown type name, and ncruces uses the declared type to decide whether
	// to decode a text column back into time.Time.
	sqliteUUIDPattern       = regexp.MustCompile(`(?i)\buuid\b`)
	sqliteJSONBTypePattern  = regexp.MustCompile(`(?i)\bjsonb\b`)
	sqliteJSONTypePattern   = regexp.MustCompile(`(?i)\bjson\b`)
	sqliteDefaultNowPattern = regexp.MustCompile(`(?i)default\s+now\(\)`)
	// SQLite's ADD COLUMN has no IF NOT EXISTS; a fresh sequential apply never
	// re-adds a column, so the guard can be dropped.
	sqliteAddColumnGuard = regexp.MustCompile(`(?i)add\s+column\s+if\s+not\s+exists`)

	// ALTER TABLE forms SQLite cannot execute. The demo builds the final schema
	// fresh and validates data in the domain layer, so dropping these leaves an
	// equivalent-enough schema (looser constraints, all columns still present).
	sqliteUnsupportedAlter = regexp.MustCompile(`(?is)^\s*alter\s+table\s+\w+\s+(drop\s+constraint|add\s+constraint|alter\s+column|drop\s+column)`)
)

// sqliteUnsupportedDDL reports whether a single translated DDL statement is one
// SQLite cannot run and that MigrateUpSQLite skips.
func sqliteUnsupportedDDL(statement string) bool {
	return sqliteUnsupportedAlter.MatchString(statement)
}

var (
	// A single ALTER TABLE that adds several columns at once. SQLite permits
	// only one operation per ALTER, so these are expanded into one statement
	// per column.
	sqliteAlterAddColumns = regexp.MustCompile(`(?is)^(\s*alter\s+table\s+\w+\s+)(add\s+column\s+.*)$`)
	sqliteAddColumnSep    = regexp.MustCompile(`(?i),\s*add\s+column\s+`)
)

var (
	sqliteAddPrimaryKey       = regexp.MustCompile(`(?is)^\s*alter\s+table\s+(\w+)\s+add\s+primary\s+key\s*\(([^)]*)\)`)
	sqliteAddUniqueConstraint = regexp.MustCompile(`(?is)^\s*alter\s+table\s+(\w+)\s+add\s+constraint\s+(\w+)\s+(?:unique|primary\s+key)\s*\(([^)]*)\)`)
)

// rewriteSQLiteConstraintToIndex turns an ALTER TABLE that adds a PRIMARY KEY or
// UNIQUE constraint (which SQLite cannot do) into an equivalent unique index, so
// the columns backing an ON CONFLICT target still get their uniqueness. Returns
// false for other statements. CHECK constraints are not convertible and are
// skipped elsewhere (the domain layer validates instead).
func rewriteSQLiteConstraintToIndex(statement string) (string, bool) {
	if match := sqliteAddPrimaryKey.FindStringSubmatch(statement); match != nil {
		table, columns := match[1], match[2]
		return "create unique index if not exists " + table + "_pkey on " + table + " (" + columns + ")", true
	}
	if match := sqliteAddUniqueConstraint.FindStringSubmatch(statement); match != nil {
		table, name, columns := match[1], match[2], match[3]
		return "create unique index if not exists " + name + " on " + table + " (" + columns + ")", true
	}
	return statement, false
}

var sqliteAddColumnStmt = regexp.MustCompile(`(?is)^\s*alter\s+table\s+\w+\s+add\s+column`)

// sqliteIsAddColumn reports whether a statement adds a column. Postgres uses
// "add column if not exists" idempotently; SQLite lacks the guard, so a
// duplicate-column error on such a statement is the equivalent no-op.
func sqliteIsAddColumn(statement string) bool {
	return sqliteAddColumnStmt.MatchString(statement)
}

// expandSQLiteAddColumns splits a multi-column ALTER TABLE ADD COLUMN into one
// single-column ALTER statement each; other statements pass through unchanged.
func expandSQLiteAddColumns(statement string) []string {
	match := sqliteAlterAddColumns.FindStringSubmatch(statement)
	if match == nil {
		return []string{statement}
	}
	prefix := match[1]
	segments := sqliteAddColumnSep.Split(match[2], -1)
	expanded := make([]string, 0, len(segments))
	for index, segment := range segments {
		if index == 0 {
			expanded = append(expanded, prefix+segment)
			continue
		}
		expanded = append(expanded, prefix+"add column "+segment)
	}
	return expanded
}

// translateSQLiteStatement rewrites a Postgres query so it runs on SQLite.
func translateSQLiteStatement(query string) string {
	query = sqliteForUpdatePattern.ReplaceAllString(query, "")
	query = sqliteCastPattern.ReplaceAllString(query, "")
	query = strings.ReplaceAll(query, "jsonb_build_object", "json_object")
	query = strings.ReplaceAll(query, "jsonb_agg", "json_group_array")
	query = sqliteNowPattern.ReplaceAllString(query, sqliteNowExpr)
	query = sqlitePlaceholder.ReplaceAllString(query, "?$1")
	return query
}

// translateSQLiteDDL rewrites a Postgres migration so it runs on SQLite. Column
// defaults need the expression parenthesized, and Postgres-only type names
// become text (except timestamptz, kept for time decoding).
func translateSQLiteDDL(ddl string) string {
	ddl = sqliteLineComment.ReplaceAllString(ddl, "")
	ddl = sqliteDefaultNowPattern.ReplaceAllString(ddl, "default ("+sqliteNowExpr+")")
	ddl = sqliteAddColumnGuard.ReplaceAllString(ddl, "add column")
	ddl = sqliteCastPattern.ReplaceAllString(ddl, "")
	ddl = sqliteUUIDPattern.ReplaceAllString(ddl, "text")
	ddl = sqliteJSONBTypePattern.ReplaceAllString(ddl, "text")
	ddl = sqliteJSONTypePattern.ReplaceAllString(ddl, "text")
	return ddl
}
