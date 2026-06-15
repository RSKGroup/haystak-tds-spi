// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

var cryptoCases = []Case{
	{Element: "func:HASHBYTES", Name: "HASHBYTES/SHA2_256", SQL: "SELECT HASHBYTES('SHA2_256','abc')", Want: []any{bytesLen(32)}},
	{Element: "func:HASHBYTES", Name: "HASHBYTES/MD5", SQL: "SELECT HASHBYTES('MD5','abc')", Want: []any{bytesLen(16)}},
	{Element: "func:CHECKSUM", SQL: "SELECT CHECKSUM('hello')", Want: []any{1335831723}},
	{Element: "func:BINARY_CHECKSUM", SQL: "SELECT BINARY_CHECKSUM('hello')", Want: []any{1335831723}},
}
