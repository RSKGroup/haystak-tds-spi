// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package tds defines the backend SPI for the haystak-tds-spi SQL-wire (TDS) gateway.
//
// A backend implements [Backend] (Describe + Capabilities) plus one query path: [Scanner]
// (thin: return whole tables, the core engine evaluates) or [QueryExecutor] (thick: handle a
// whole [Query]). It advertises what it supports via [Caps]. Write and catalog extensions are
// optional, detected by interface assertion and gated by Caps: [Writer], [DDL], [DatabaseDDL],
// [Databaser], [TxBeginner]/[Tx]. Run a backend with the server package's ListenAndServe, and
// validate it with tds/tdstest.RunConformance.
package tds
