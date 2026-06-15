// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// With no principal in context the security scalars return their unauthenticated defaults; the populated
// path (a real tds.Principal reflected through sys.*_principals) is asserted in the inmem example.
var securityCases = []Case{
	{Element: "func:SYSTEM_USER", SQL: "SELECT SYSTEM_USER", Want: []any{"haystak"}},
	{Element: "func:CURRENT_USER", SQL: "SELECT CURRENT_USER", Want: []any{"haystak"}},
	{Element: "func:SESSION_USER", SQL: "SELECT SESSION_USER", Want: []any{"haystak"}},
	{Element: "func:USER", SQL: "SELECT USER", Want: []any{"haystak"}},
	{Element: "func:USER_NAME", SQL: "SELECT USER_NAME()", Want: []any{"haystak"}},
	{Element: "func:SUSER_NAME", SQL: "SELECT SUSER_NAME()", Want: []any{"haystak"}},
	{Element: "func:SUSER_SNAME", SQL: "SELECT SUSER_SNAME()", Want: []any{"haystak"}},
	{Element: "func:ORIGINAL_LOGIN", SQL: "SELECT ORIGINAL_LOGIN()", Want: []any{"haystak"}},
	{Element: "func:HOST_NAME", SQL: "SELECT HOST_NAME()", Want: []any{"haystak-tds-spi"}},
	{Element: "func:APP_NAME", SQL: "SELECT APP_NAME()", Want: []any{"haystak-tds-spi"}},
	{Element: "func:USER_ID", SQL: "SELECT USER_ID()", Want: []any{1}},
	{Element: "func:SUSER_ID", SQL: "SELECT SUSER_ID()", Want: []any{1}},
	{Element: "func:IS_MEMBER", SQL: "SELECT IS_MEMBER('public')", Want: []any{1}},
	{Element: "func:IS_SRVROLEMEMBER", SQL: "SELECT IS_SRVROLEMEMBER('sysadmin')", Want: []any{P(isNull)}},
	{Element: "func:IS_ROLEMEMBER", SQL: "SELECT IS_ROLEMEMBER('public')", Want: []any{1}},
	{Element: "func:CONTEXT_INFO", SQL: "SELECT CONTEXT_INFO()", Want: []any{P(isNull)}},
	{Element: "func:SESSION_CONTEXT", SQL: "SELECT SESSION_CONTEXT('k')", Want: []any{P(isNull)}},
	{Element: "func:HAS_PERMS_BY_NAME", SQL: "SELECT HAS_PERMS_BY_NAME('x','OBJECT','SELECT')", Want: []any{1}},
}
