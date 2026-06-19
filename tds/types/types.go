// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package types is the backend-neutral type system, mapped to concrete T-SQL types at the wire.
package types

// Kind is a backend-neutral column type, mapped to a T-SQL type by the gateway.
type Kind int

const (
	Unknown Kind = iota
	Bool
	Int32
	Int64
	Float64
	Decimal
	String
	Bytes
	Time
	UUID
	Int8  // TINYINT; SMO reads sys.databases tinyint columns as Byte
	Int16 // SMALLINT; SMO reads sys.columns.max_length as Int16
)

// Type is a column type: its Kind plus optional nullability, length, and decimal precision/scale.
type Type struct {
	Kind      Kind
	Nullable  bool
	MaxLen    int
	Precision int
	Scale     int
	Name      string // declared T-SQL type ("varchar", "money"); catalog reporters prefer it over the Kind default
}
