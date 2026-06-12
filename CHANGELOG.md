# Changelog

## v1.2.4

GUI/driver compatibility — additive, no API changes.

- **Catalog scalar functions** — `HAS_DBACCESS`, `DB_ID`, `SCHEMA_NAME`, `SCHEMA_ID`, `OBJECT_ID`,
  and `QUOTENAME` now return real values. After v1.2.3 advertised a SQL Server build, native GUIs
  (SQLPro Studio, SSMS, Power BI) gate the database/object tree on `HAS_DBACCESS(...) = 1`; the
  evaluator previously had no answer and the tree came up empty. Clients now browse databases,
  schemas, and objects.

## v1.2.3

GUI/driver compatibility — additive, no API changes.

- **Handshake reports a real server version** — PRELOGIN and LOGINACK now advertise a SQL Server
  build (`16.0.1000.6`) instead of `0.0.0.0`. Microsoft's native stack (ODBC Driver 18, OLE DB,
  .NET SqlClient — and therefore SSMS, Power BI, Excel) reads the PRELOGIN version and rejects
  anything it reads as "SQL Server 2000 or earlier" *before* login; go-mssqldb and FreeTDS were
  lenient and connected regardless. Native-driver clients now connect.

## v1.2.2

Docs only — no code or API changes.

- **"Where it fits"** — a README positioning section describing, by category, how this SPI differs
  from single-store SQL features, single-database wire shims, proprietary multi-connector gateways,
  data-virtualization platforms, and distributed query engines.

## v1.2.1

GUI/driver compatibility — additive, no breaking API changes.

- **LOGINACK TDS version** — the server reports TDS 7.4 with the on-the-wire bytes real SQL Server
  sends (`74 00 00 04`), so strict clients (FreeTDS — SQLPro Studio, DBeaver/jTDS, pyodbc/pymssql)
  accept the login instead of dropping it, while go-mssqldb / .NET read the full 7.4 feature level.
- **Catalog stored procedures** — `sp_databases`, `sp_tables`, and `sp_columns` are answered both as a
  batch (`EXEC sp_tables`) and as a by-name RPC, and an RPC for any other procedure is no longer
  silently dropped, so ODBC/GUI clients can enumerate databases, tables, and columns.
- **TDS-in-TLS handshake pinned to TLS 1.2** — the PRELOGIN-wrapped handshake completes with FreeTDS
  and other clients that do not expect a TLS 1.3 flight.
- **Session current database** — `USE [db]` updates the connection's current database (with an
  `ENVCHANGE`), an unqualified query resolves against it, and `DB_NAME()` reflects it.
- **Per-database catalog scoping** — `sys.tables` / `sys.columns` / `INFORMATION_SCHEMA` report the
  current database's objects so a GUI's per-node table list is correct; `sys.databases` stays
  server-wide, and an unknown or system database resolves to an empty catalog rather than an error.

## v1.2.0

Additive — no breaking API changes.

- **Multi-database catalog views** — `sys.databases` lists a `Databaser` backend's databases, and
  `INFORMATION_SCHEMA` / `sys.*` aggregate every database's tables (each tagged with its catalog) so
  GUI/BI tools can browse the whole server; a database-qualified query (`[db].INFORMATION_SCHEMA.…`)
  narrows to that database. `catalog.Table` gains a `Catalog` field.
- **`nvarchar(max)` / PLP** — string columns that are unbounded or longer than 4000 characters are
  declared `nvarchar(max)` and their values PLP-encoded, so a value larger than 8000 bytes (e.g. a full
  document's text) no longer overflows the client reader.
- **Aggregation pushdown** — an optional `Aggregator` interface (gated by `Caps.Aggregate`) lets a
  backend answer a pure aggregation in its own engine; returning `ErrAggregateUnsupported` falls back to
  the scan path.
- **Writable multi-database routing** — `tds.Insert` / `Update` / `Delete` carry the `Database` /
  `Schema` qualifier so a written `[db].table` routes to the intended database.
- **`NULL` literal** — `NULL` parses as a literal, so `SET col = NULL` and `INSERT … VALUES (NULL)` work.
- **Non-reserved keywords as identifiers** — a column or table named like a clause keyword (`first`,
  `next`, `rows`, `value`, …) parses unquoted; keyword matching is case-insensitive and identifier case
  is preserved.

## v1.1.0

Engine and examples, additive — no breaking API changes.

- **Join pushdown** — for an equi-`JOIN … ON`, the engine scans the right table only for rows whose
  join key matches a left-side key (a semi-join pushed as a right-side `IN` filter), for INNER/LEFT
  joins. The database qualifier of a joined table is now threaded through the parser and executor, so
  cross-database joins (`… JOIN otherdb.schema.t …`) resolve against the intended database.
- **Aggregates in HAVING and ORDER BY** — `COUNT(*)` parses as a function argument anywhere an
  expression is allowed, so `HAVING COUNT(*) > 1` works; `ORDER BY` accepts expressions and aggregates
  (`ORDER BY COUNT(*) DESC`); and aggregate calls in HAVING/ORDER BY evaluate over the group rather than
  the already-aggregated output row. `tds.OrderItem` gains an `Expr` field.
- **Elasticsearch and OpenSearch example backends** — `examples/elasticsearch-community` and
  `examples/opensearch-community` (inferred catalog, fields sampled from `_source`) plus their `-2`
  variants (declared catalog: columns from the native `_mapping`, primary/foreign keys from a reserved
  `haystak_catalog` index). Each takes `--host <url:port>` and `--db <name>`; the MongoDB examples gain
  the same flags.

## v1.0.1

Documentation only, no API or behavior changes: a doc comment on every exported symbol,
package overviews on all public packages, and runnable `Example` functions (package
example, `ExampleAuthFunc`, `ExampleServer`, `ExampleStaticAuth`) — so the full reference
renders on pkg.go.dev. README gains Install, Documentation, and a Go Reference badge.

## v1.0.0

Initial release — a pure-Go TDS (SQL Server wire) gateway shipped as an importable SPI,
Apache-2.0 licensed. No binary: consumers `go get` the module and implement a backend.

- **Read engine** — full T-SQL read subset: projection / DISTINCT / TOP[/PERCENT] / aliases; JOINs
  (INNER/LEFT/RIGHT/FULL/CROSS); GROUP BY / HAVING / aggregates; ORDER BY (incl. ordinals);
  OFFSET/FETCH; UNION/ALL/INTERSECT/EXCEPT; subqueries (IN/EXISTS/scalar/derived/correlated);
  CTEs (incl. recursive); expressions (arithmetic/CASE/CAST/string+date funcs) in SELECT and WHERE;
  no-FROM scalar SELECT.
- **Write dispatch** — INSERT/UPDATE/DELETE + CREATE/DROP TABLE/DATABASE routed to the backend's
  Writer / DDL / DatabaseDDL (fail-closed when unsupported).
- **Catalog** — INFORMATION_SCHEMA + sys.* including foreign-key views.
- **Wire** — PRELOGIN / LOGIN7 / TLS-in-TDS / SQL_BATCH / RPC sp_executesql / token streams; typed
  encodings (int/bit/float/decimal/datetime2/uniqueidentifier/varbinary/nvarchar).
- **SPI** — `Backend` (+ `Scanner` thin / `QueryExecutor` thick), `Writer` / `DDL` / `DatabaseDDL` /
  `Databaser` / `Tx`, `Authenticator` (backend-owned login go/no-go + per-user `Principal` in
  context); `Caps` capability model; `tds/tdstest` conformance harness.
- **Examples** — `inmem` (reference), `gateway` (runnable), and three `mongodb-community` variants
  (inferred / declared / hardcoded catalog) showing the catalog models against real MongoDB, each
  its own module so the core stays dependency-free.
