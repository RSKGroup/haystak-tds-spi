// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import (
	"context"
	"errors"

	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

// ErrUnsupported is returned when a statement needs a capability the backend does not implement.
var ErrUnsupported = errors.New("tds: operation not supported by backend")

// TxModel describes how much transaction support a backend offers, declared via Caps.Tx.
type TxModel int

const (
	TxNone       TxModel = iota // no transactions
	TxBestEffort                // attempted, not guaranteed atomic
	TxFull                      // full ACID transactions
)

// Caps is the backend's honest capability declaration; the gateway degrades against it.
type Caps struct {
	FullQuery bool    // implements QueryExecutor (thick path)
	Pushdown  bool    // implements Scanner (thin path; engine evaluates the query)
	Writable  bool    // implements Writer (INSERT/UPDATE/DELETE)
	DDL       bool    // implements DDL and/or DatabaseDDL
	Aggregate bool    // implements Aggregator (pushes some GROUP BY / aggregate queries)
	Routines  bool    // implements RoutineStore (CREATE VIEW / PROCEDURE)
	Tx        TxModel // transaction support level
}

// Backend is the minimum a store implements: describe its catalog and declare its capabilities.
// It also implements one query path (Scanner or QueryExecutor) plus any optional write/catalog interfaces.
type Backend interface {
	Describe(ctx context.Context) (catalog.Schema, error)
	Capabilities() Caps
}

// QueryExecutor is the thick path: the backend executes a whole logical query in its own engine.
type QueryExecutor interface {
	ExecuteQuery(ctx context.Context, q *Query) (Rows, error)
}

// Scanner is the thin path: the backend returns whole tables and the gateway engine evaluates the query.
type Scanner interface {
	Scan(ctx context.Context, q *Query) (Rows, error)
}

// Writer is the optional row-write contract (INSERT/UPDATE/DELETE), gated by Caps.Writable.
type Writer interface {
	Insert(ctx context.Context, in *Insert) (Result, error)
	Update(ctx context.Context, up *Update) (Result, error)
	Delete(ctx context.Context, del *Delete) (Result, error)
}

// Aggregator optionally pushes a pure aggregation (no joins); return ErrAggregateUnsupported to fall back to scan.
type Aggregator interface {
	Aggregate(ctx context.Context, q *Query) (Rows, error)
}

// ErrAggregateUnsupported tells the engine an Aggregator can't push this query — use the scan path.
var ErrAggregateUnsupported = errors.New("tds: aggregate not pushable")

// DDL is the optional table-write contract (CREATE/ALTER/DROP TABLE), gated by Caps.DDL.
type DDL interface {
	CreateTable(ctx context.Context, t *catalog.Table) error
	AlterTable(ctx context.Context, a *AlterTable) error
	DropTable(ctx context.Context, table string) error
}

// DatabaseDDL is the database-level write contract (CREATE/DROP DATABASE), gated by Caps.DDL.
type DatabaseDDL interface {
	CreateDatabase(ctx context.Context, name string) error
	DropDatabase(ctx context.Context, name string) error
}

// Databaser is implemented by backends exposing more than one database (powering sys.databases and
// Query.Database routing). Single-database backends omit it and the gateway presents one default database.
type Databaser interface {
	Databases(ctx context.Context) ([]string, error)
	DescribeDatabase(ctx context.Context, db string) (catalog.Schema, error)
}

// RoutineKind distinguishes a stored view from a stored procedure.
type RoutineKind int

const (
	RoutineView RoutineKind = iota + 1
	RoutineProc
)

// RoutineParam is one procedure parameter: its @name and declared T-SQL type text.
type RoutineParam struct {
	Name string // includes the leading @
	Type string // declared type as written, e.g. "int", "nvarchar(50)"
}

// Routine is a stored view or procedure definition. The gateway parses and runs Body (the text after
// AS) at use time, so the backend only persists and returns it — no SQL knowledge needed in the store.
type Routine struct {
	Database string
	Schema   string // "dbo" when unspecified
	Name     string
	Kind     RoutineKind
	Body     string
	Params   []RoutineParam // procedures only, in declared order
}

// RoutineStore is the optional contract for persisting view/procedure definitions, gated by
// Caps.Routines. Names are matched case-insensitively within a database.
type RoutineStore interface {
	PutRoutine(ctx context.Context, r *Routine) error
	GetRoutine(ctx context.Context, database, name string) (*Routine, bool, error)
	DropRoutine(ctx context.Context, database, name string) error
	ListRoutines(ctx context.Context, database string) ([]*Routine, error)
}

// TxBeginner is the optional transaction contract, gated by Caps.Tx; Begin opens a Tx.
type TxBeginner interface {
	Begin(ctx context.Context, opts TxOptions) (Tx, error)
}

// Tx is an in-flight transaction returned by TxBeginner.Begin.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TxOptions configures a transaction opened via TxBeginner.Begin.
type TxOptions struct {
	ReadOnly bool
}

// Rows is a forward-only result cursor. Drive it with Next/Values, check Err, and always Close.
type Rows interface {
	Columns() []catalog.Column
	Next() bool
	Values() ([]any, error)
	Err() error
	Close() error
}

// Result reports the outcome of a write: the number of rows affected.
type Result struct {
	RowsAffected int64
}
