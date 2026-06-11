// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import "github.com/RSKGroup/haystak-tds-spi/tds/catalog"

// Insert is a parsed INSERT: one entry in Rows per VALUES tuple, aligned to Columns (or table order).
type Insert struct {
	Database string
	Schema   string
	Table    string
	Columns  []string
	Rows     [][]any
}

// Assignment is one SET column = Value pair in an Update.
type Assignment struct {
	Column string
	Value  any
}

// Update is a parsed UPDATE: apply Assignments to the rows matching Where.
type Update struct {
	Database    string
	Schema      string
	Table       string
	Assignments []Assignment
	Where       []Predicate
}

// Delete is a parsed DELETE: remove the rows matching Where (all rows when Where is empty).
type Delete struct {
	Database string
	Schema   string
	Table    string
	Where    []Predicate
}

// AlterTable is a parsed ALTER TABLE: add and/or drop columns.
type AlterTable struct {
	Table       string
	AddColumns  []catalog.Column
	DropColumns []string
}

// WriteStmt is a parsed write/DDL statement; exactly one field is set.
type WriteStmt struct {
	Insert      *Insert
	Update      *Update
	Delete      *Delete
	CreateTable *catalog.Table
	Alter       *AlterTable
	DropTable   string
	CreateDB    string
	DropDB      string
}
