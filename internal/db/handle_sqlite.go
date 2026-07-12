package db

import (
	"context"
	"database/sql"
	"errors"
)

// NewSQLite adapts a database/sql handle (backed by ncruces/go-sqlite3) to
// Beginner for the browser demo. Every statement is passed through the SQLite
// dialect translation before execution, so stores run their Postgres SQL
// unchanged. Like handle.go this is a policy-check boundary: it wraps the
// weakly-typed database/sql surface.
func NewSQLite(handle *sql.DB) Beginner {
	return sqliteHandle{handle: handle}
}

type sqliteHandle struct {
	handle *sql.DB
}

func (h sqliteHandle) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := h.handle.ExecContext(ctx, translateSQLiteStatement(query), sqliteArgs(args)...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (h sqliteHandle) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := h.handle.QueryContext(ctx, translateSQLiteStatement(query), sqliteArgs(args)...)
	if err != nil {
		return nil, err
	}
	return sqliteRows{rows: rows}, nil
}

func (h sqliteHandle) QueryRow(ctx context.Context, query string, args ...any) Row {
	return sqliteRow{row: h.handle.QueryRowContext(ctx, translateSQLiteStatement(query), sqliteArgs(args)...)}
}

func (h sqliteHandle) Begin(ctx context.Context) (Tx, error) {
	tx, err := h.handle.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return sqliteTx{tx: tx}, nil
}

type sqliteTx struct {
	tx *sql.Tx
}

func (t sqliteTx) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := t.tx.ExecContext(ctx, translateSQLiteStatement(query), sqliteArgs(args)...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (t sqliteTx) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, translateSQLiteStatement(query), sqliteArgs(args)...)
	if err != nil {
		return nil, err
	}
	return sqliteRows{rows: rows}, nil
}

func (t sqliteTx) QueryRow(ctx context.Context, query string, args ...any) Row {
	return sqliteRow{row: t.tx.QueryRowContext(ctx, translateSQLiteStatement(query), sqliteArgs(args)...)}
}

func (t sqliteTx) Commit(ctx context.Context) error   { return t.tx.Commit() }
func (t sqliteTx) Rollback(ctx context.Context) error { return t.tx.Rollback() }

// sqliteRows adapts *sql.Rows; Close discards its error to match the Rows shape.
type sqliteRows struct {
	rows *sql.Rows
}

func (r sqliteRows) Next() bool             { return r.rows.Next() }
func (r sqliteRows) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r sqliteRows) Close()                 { _ = r.rows.Close() }
func (r sqliteRows) Err() error             { return r.rows.Err() }

// sqliteRow translates sql.ErrNoRows to the engine-neutral ErrNoRows.
type sqliteRow struct {
	row *sql.Row
}

func (r sqliteRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoRows
	}
	return err
}

// sqliteArgs expands any NamedArgs into database/sql named arguments and passes
// other arguments through unchanged.
func sqliteArgs(args []any) []any {
	expanded := make([]any, 0, len(args))
	for _, arg := range args {
		if named, matched := arg.(NamedArgs); matched {
			for name, value := range named {
				expanded = append(expanded, sql.Named(name, value))
			}
			continue
		}
		expanded = append(expanded, arg)
	}
	return expanded
}
