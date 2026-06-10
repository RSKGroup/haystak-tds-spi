// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package catalog

import "github.com/RSKGroup/haystak-tds-spi/tds/types"

type Schema struct {
	Tables []Table
}

type Table struct {
	Name        string
	Columns     []Column
	PrimaryKey  []string
	ForeignKeys []ForeignKey
}

type Column struct {
	Name    string
	Type    types.Type
	Default string
}

type ForeignKey struct {
	Columns    []string
	RefTable   string
	RefColumns []string
}
