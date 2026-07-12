package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoRows is the engine-neutral "no rows in result set" sentinel. Both
// adapters translate their driver's own sentinel (pgx.ErrNoRows /
// sql.ErrNoRows) to this, so stores compare against one value regardless of
// engine.
var ErrNoRows = errors.New("db: no rows in result set")

// NamedArgs binds query parameters by name (used with @name placeholders,
// which both pgx and SQLite accept). Each adapter converts it to its driver's
// native named-argument form.
type NamedArgs map[string]any

// Querier is the read/write surface every store depends on, satisfied by both
// the production Postgres adapter (pgx, below) and the browser SQLite adapter
// (database/sql, added later for the demo). Stores never touch pgx or
// database/sql directly, so the same store code runs against either engine.
//
// The variadic argument and scan-destination signatures mirror the underlying
// driver APIs; that is why this file is listed as a policy-check boundary: it
// is the one seam between the typed domain layer and the deliberately untyped
// driver interfaces.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
}

// Beginner is a Querier that can also open a transaction.
type Beginner interface {
	Querier
	Begin(ctx context.Context) (Tx, error)
}

// Tx is an open database transaction.
type Tx interface {
	Querier
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Rows iterates a query result. Close returns nothing, matching pgx.Rows; the
// SQLite adapter discards sql.Rows.Close's error to satisfy the same shape.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

// Row is a single-row query result.
type Row interface {
	Scan(dest ...any) error
}

// NewPGX adapts a pgx pool to Beginner for production use.
func NewPGX(pool *pgxpool.Pool) Beginner {
	return pgxHandle{pool: pool}
}

type pgxHandle struct {
	pool *pgxpool.Pool
}

func (h pgxHandle) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := h.pool.Exec(ctx, sql, pgxArgs(args)...)
	return tag.RowsAffected(), err
}

func (h pgxHandle) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := h.pool.Query(ctx, sql, pgxArgs(args)...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (h pgxHandle) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return pgxRow{row: h.pool.QueryRow(ctx, sql, pgxArgs(args)...)}
}

func (h pgxHandle) Begin(ctx context.Context) (Tx, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return pgxTx{tx: tx}, nil
}

type pgxTx struct {
	tx pgx.Tx
}

func (t pgxTx) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := t.tx.Exec(ctx, sql, pgxArgs(args)...)
	return tag.RowsAffected(), err
}

func (t pgxTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := t.tx.Query(ctx, sql, pgxArgs(args)...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (t pgxTx) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return pgxRow{row: t.tx.QueryRow(ctx, sql, pgxArgs(args)...)}
}

func (t pgxTx) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t pgxTx) Rollback(ctx context.Context) error { return t.tx.Rollback(ctx) }

// pgxRow translates pgx.ErrNoRows to the engine-neutral ErrNoRows.
type pgxRow struct {
	row pgx.Row
}

func (r pgxRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoRows
	}
	return err
}

// pgxArgs converts any NamedArgs in the argument list to pgx.NamedArgs.
func pgxArgs(args []any) []any {
	for index, arg := range args {
		if named, matched := arg.(NamedArgs); matched {
			args[index] = pgx.NamedArgs(named)
		}
	}
	return args
}
