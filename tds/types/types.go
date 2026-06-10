// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package types

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

type Type struct {
	Kind      Kind
	Nullable  bool
	MaxLen    int
	Precision int
	Scale     int
}
