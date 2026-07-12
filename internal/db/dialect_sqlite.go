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
	// now() + make_interval(hours => $N) — Postgres interval arithmetic becomes
	// a strftime "now" modifier ('+N hours'). Must run before the now() rewrite.
	sqliteMakeIntervalHours = regexp.MustCompile(`(?i)now\(\)\s*\+\s*make_interval\(\s*hours\s*=>\s*(\$\d+)\s*\)`)
	// Postgres $N placeholders bind by number. SQLite treats $1 as a named
	// parameter numbered by appearance, so "$3 ... $1" would bind wrong;
	// rewriting to ?N restores explicit by-number binding.
	sqlitePlaceholder = regexp.MustCompile(`\$(\d+)`)

	// array_remove(array_agg(X), null) drops nulls from an aggregated array;
	// json_group_array keeps them, so the null filter moves into a FILTER clause.
	sqliteArrayRemoveAgg = regexp.MustCompile(`(?is)array_remove\(\s*array_agg\(([^)]+)\)\s*,\s*null\s*\)`)
	// ILIKE has no SQLite form, but SQLite's LIKE is already case-insensitive
	// for ASCII, which is what the scope/title searches rely on.
	sqliteILike = regexp.MustCompile(`(?i)\bilike\b`)
	// "X at time zone 'UTC'" — ncruces stores UTC, so the conversion is dropped.
	sqliteAtTimeZone = regexp.MustCompile(`(?i)\s+at time zone '[^']*'`)
	// to_char(X, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') renders an RFC3339 timestamp; the
	// SQLite equivalent is strftime with the matching format.
	sqliteToCharRFC3339 = regexp.MustCompile(`(?is)to_char\(\s*([^,]+?)\s*,\s*'YYYY-MM-DD"T"HH24:MI:SS"Z"'\s*\)`)
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
	// SQLite's ADD/DROP COLUMN have no IF (NOT) EXISTS; a fresh sequential apply
	// always has the expected column state, so the guards can be dropped.
	sqliteAddColumnGuard  = regexp.MustCompile(`(?i)add\s+column\s+if\s+not\s+exists`)
	sqliteDropColumnGuard = regexp.MustCompile(`(?i)drop\s+column\s+if\s+exists`)

	// ALTER TABLE forms SQLite cannot execute. Constraint and column-type
	// changes are dropped; the demo builds the final schema fresh and validates
	// data in the domain layer. (DROP COLUMN, which SQLite does support, is
	// executed instead so removed columns don't linger with stale NOT NULLs.)
	sqliteUnsupportedAlter = regexp.MustCompile(`(?is)^\s*alter\s+table\s+\w+\s+(drop\s+constraint|add\s+constraint|alter\s+column)`)
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
	query = sqliteAtTimeZone.ReplaceAllString(query, "")
	query = sqliteToCharRFC3339.ReplaceAllString(query, "strftime('%Y-%m-%dT%H:%M:%SZ', $1)")
	query = sqliteArrayRemoveAgg.ReplaceAllString(query, "json_group_array($1) filter (where $1 is not null)")
	query = strings.ReplaceAll(query, "array_agg", "json_group_array")
	query = strings.ReplaceAll(query, "jsonb_build_object", "json_object")
	query = strings.ReplaceAll(query, "jsonb_agg", "json_group_array")
	query = sqliteILike.ReplaceAllString(query, "like")
	query = sqliteCastPattern.ReplaceAllString(query, "")
	query = sqliteMakeIntervalHours.ReplaceAllString(query, "strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+' || ${1} || ' hours')")
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
	ddl = sqliteDropColumnGuard.ReplaceAllString(ddl, "drop column")
	ddl = sqliteCastPattern.ReplaceAllString(ddl, "")
	ddl = sqliteUUIDPattern.ReplaceAllString(ddl, "text")
	ddl = sqliteJSONBTypePattern.ReplaceAllString(ddl, "text")
	ddl = sqliteJSONTypePattern.ReplaceAllString(ddl, "text")
	return ddl
}

var (
	sqliteCheckKeyword     = regexp.MustCompile(`(?i)\bcheck\s*\(`)
	sqliteConstraintPrefix = regexp.MustCompile(`(?i)constraint\s+\w+\s*$`)
	sqliteDoubleComma      = regexp.MustCompile(`,(\s*,)+`)
	sqliteTrailingComma    = regexp.MustCompile(`,\s*\)`)
	sqliteLeadingComma     = regexp.MustCompile(`\(\s*,`)
)

// stripSQLiteChecks removes CHECK constraints from DDL. Later migrations relax
// several of them via ALTER statements SQLite cannot replay, so the original
// inline checks would reject rows Postgres accepts; the domain layer validates
// instead. A named "constraint <name> check (...)" prefix is removed with it,
// and comma artifacts are cleaned up afterward.
func stripSQLiteChecks(ddl string) string {
	for {
		keyword := sqliteCheckKeyword.FindStringIndex(ddl)
		if keyword == nil {
			break
		}
		closeParen := matchParen(ddl, keyword[1]-1)
		if closeParen < 0 {
			break
		}
		start := keyword[0]
		if prefix := sqliteConstraintPrefix.FindStringIndex(ddl[:start]); prefix != nil {
			start = prefix[0]
		}
		ddl = ddl[:start] + ddl[closeParen+1:]
	}
	ddl = sqliteDoubleComma.ReplaceAllString(ddl, ",")
	ddl = sqliteTrailingComma.ReplaceAllString(ddl, ")")
	ddl = sqliteLeadingComma.ReplaceAllString(ddl, "(")
	return ddl
}

// matchParen returns the index of the parenthesis closing the one at open, or
// -1 if unbalanced.
func matchParen(text string, open int) int {
	depth := 0
	for index := open; index < len(text); index++ {
		switch text[index] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}
