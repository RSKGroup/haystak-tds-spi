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
)

// Type is a column type: its Kind plus optional nullability, length, and decimal precision/scale.
type Type struct {
	Kind      Kind
	Nullable  bool
	MaxLen    int
	Precision int
	Scale     int
}
