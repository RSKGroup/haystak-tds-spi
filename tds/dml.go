// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import "github.com/RSKGroup/haystak-tds-spi/tds/catalog"

type Insert struct {
	Table   string
	Columns []string
	Rows    [][]any
}

type Assignment struct {
	Column string
	Value  any
}

type Update struct {
	Table       string
	Assignments []Assignment
	Where       []Predicate
}

type Delete struct {
	Table string
	Where []Predicate
}

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
