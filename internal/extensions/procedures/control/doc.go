// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package control holds T-SQL procedural constructs — one file per statement: if.go, while.go,
// declare.go, set.go, return.go, break.go. Each file parses its construct and registers an executor
// the procedure body runner dispatches by leading keyword, so adding a construct is one new file and
// a registration — nothing else changes.
//
// To build the IF statement: create if.go here, parse `IF <cond> <stmt> [ELSE <stmt>]`, register it
// under "IF", and implement Exec(scope, runner).
//
// Currently empty: procedures execute as parameterized single/multi-statement batches. Control flow
// is built out here demand-driven.
package control
