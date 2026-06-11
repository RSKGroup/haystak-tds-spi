# Changelog

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
