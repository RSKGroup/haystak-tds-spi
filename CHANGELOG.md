# Changelog

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
