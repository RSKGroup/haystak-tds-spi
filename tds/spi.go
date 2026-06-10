// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import (
	"context"
	"errors"

	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

var ErrUnsupported = errors.New("tds: operation not supported by backend")

type TxModel int

const (
	TxNone TxModel = iota
	TxBestEffort
	TxFull
)

// Caps is the backend's honest capability declaration; the gateway degrades against it.
type Caps struct {
	FullQuery bool
	Pushdown  bool
	Writable  bool
	DDL       bool
	Tx        TxModel
}

type Backend interface {
	Describe(ctx context.Context) (catalog.Schema, error)
	Capabilities() Caps
}

// QueryExecutor is the thick path: the backend executes a whole logical query.
type QueryExecutor interface {
	ExecuteQuery(ctx context.Context, q *Query) (Rows, error)
}

// Scanner is the thin path: the gateway's engine drives primitive scans.
type Scanner interface {
	Scan(ctx context.Context, q *Query) (Rows, error)
}

type Writer interface {
	Insert(ctx context.Context, in *Insert) (Result, error)
	Update(ctx context.Context, up *Update) (Result, error)
	Delete(ctx context.Context, del *Delete) (Result, error)
}

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

// Databaser is implemented by backends exposing >1 database (→ sys.databases + Query.Database
// routing). Single-database backends omit it; the gateway then presents one default database.
type Databaser interface {
	Databases(ctx context.Context) ([]string, error)
	DescribeDatabase(ctx context.Context, db string) (catalog.Schema, error)
}

type TxBeginner interface {
	Begin(ctx context.Context, opts TxOptions) (Tx, error)
}

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type TxOptions struct {
	ReadOnly bool
}

type Rows interface {
	Columns() []catalog.Column
	Next() bool
	Values() ([]any, error)
	Err() error
	Close() error
}

type Result struct {
	RowsAffected int64
}
