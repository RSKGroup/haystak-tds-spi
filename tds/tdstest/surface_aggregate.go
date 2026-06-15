// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// Aggregates run over sys.types, a static 20-row catalog with schema_id=4 on every backend, so the
// results are exact and backend-independent (variance/stdev of a constant column is 0).
var aggregateCases = []Case{
	{Element: "agg:COUNT", SQL: "SELECT COUNT(*) FROM sys.types", Want: []any{20}},
	{Element: "agg:COUNT_BIG", SQL: "SELECT COUNT_BIG(*) FROM sys.types", Want: []any{20}},
	{Element: "agg:SUM", SQL: "SELECT SUM(schema_id) FROM sys.types", Want: []any{approx(80)}},
	{Element: "agg:AVG", SQL: "SELECT AVG(schema_id) FROM sys.types", Want: []any{approx(4)}},
	{Element: "agg:MIN", SQL: "SELECT MIN(schema_id) FROM sys.types", Want: []any{4}},
	{Element: "agg:MAX", SQL: "SELECT MAX(schema_id) FROM sys.types", Want: []any{4}},
	{Element: "agg:STDEV", SQL: "SELECT STDEV(schema_id) FROM sys.types", Want: []any{approx(0)}},
	{Element: "agg:STDEVP", SQL: "SELECT STDEVP(schema_id) FROM sys.types", Want: []any{approx(0)}},
	{Element: "agg:VAR", SQL: "SELECT VAR(schema_id) FROM sys.types", Want: []any{approx(0)}},
	{Element: "agg:VARP", SQL: "SELECT VARP(schema_id) FROM sys.types", Want: []any{approx(0)}},
	{Element: "agg:STRING_AGG", SQL: "SELECT STRING_AGG(name, ',') FROM sys.types", Want: []any{"bit,tinyint,smallint,int,bigint,decimal,numeric,float,real,date,time,datetime,datetime2,char,varchar,nchar,nvarchar,binary,varbinary,uniqueidentifier"}},
	{Element: "agg:CHECKSUM_AGG", SQL: "SELECT CHECKSUM_AGG(schema_id) FROM sys.types", Want: []any{0}}, // 20 XORs of 4 = 0
	{Element: "agg:APPROX_COUNT_DISTINCT", SQL: "SELECT APPROX_COUNT_DISTINCT(schema_id) FROM sys.types", Want: []any{1}},
}
