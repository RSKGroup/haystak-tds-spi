// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package catalog is the schema model a tds.Backend declares: tables, columns, and keys.
package catalog

import "github.com/RSKGroup/haystak-tds-spi/tds/types"

// Schema is one database's worth of tables, returned by Backend.Describe.
type Schema struct {
	Tables []Table
}

// Table is one table's definition: columns, keys, and (multi-database backends) its Catalog.
type Table struct {
	Name        string
	Catalog     string
	Columns     []Column
	PrimaryKey  []string
	ForeignKeys []ForeignKey
}

// Column is one column: its name, type, and optional default expression.
type Column struct {
	Name    string
	Type    types.Type
	Default string
}

// ForeignKey declares that Columns reference RefColumns in RefTable; it drives sys.foreign_keys and joins.
type ForeignKey struct {
	Columns    []string
	RefTable   string
	RefColumns []string
}
