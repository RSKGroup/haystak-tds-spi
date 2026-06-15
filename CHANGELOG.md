# Changelog

## Unreleased (v1.5.0)

A broad SQL Server surface expansion — additive, no breaking changes. The one public API addition is
`catalog.Schema.TableTypes` (with `catalog.TableType`), so a backend can declare user-defined table types.

Functions:

- Aggregate: `COUNT_BIG`, `STDEV`, `STDEVP`, `VAR`, `VARP`, `STRING_AGG`, `CHECKSUM_AGG`, `APPROX_COUNT_DISTINCT` (with separator; `WITHIN GROUP` ordering not yet supported).
- String: `CHARINDEX`, `PATINDEX`, `LEFT`, `RIGHT`, `REPLICATE`, `STUFF`, `REVERSE`, `SPACE`, `ASCII`, `CHAR`, `UNICODE`, `NCHAR`, `CONCAT_WS`, `TRANSLATE`, `STR`, `STRING_ESCAPE`.
- Date & time: `DATEADD`, `DATEDIFF`, `DATEDIFF_BIG`, `DATEPART`, `DATENAME`, `DATETRUNC`, `EOMONTH`, `DATEFROMPARTS`, `DATETIMEFROMPARTS`, `ISDATE`, `CURRENT_TIMESTAMP`, `SYSDATETIMEOFFSET` (month-end clamping on date math).
- Mathematical: `CEILING`, `FLOOR`, `ROUND`, `POWER`, `SQRT`, `SQUARE`, `EXP`, `LOG`, `LOG10`, `SIN`, `COS`, `TAN`, `COT`, `ASIN`, `ACOS`, `ATAN`, `ATN2`, `PI`, `RAND`, `SIGN`, `DEGREES`, `RADIANS`.
- Logical: `IIF`, `CHOOSE`.
- Conversion: `TRY_CAST`, `TRY_CONVERT`, `PARSE`, `TRY_PARSE`.
- JSON: `ISJSON`, `JSON_VALUE`, `JSON_QUERY`.
- Cryptographic: `HASHBYTES`, `CHECKSUM`, `BINARY_CHECKSUM`.
- Metadata: `TYPE_ID`, `COL_NAME`, `COL_LENGTH`, `OBJECTPROPERTY`, `OBJECTPROPERTYEX`, `COLUMNPROPERTY`, `OBJECT_DEFINITION`, `STATS_DATE`.
- Security: `USER_ID`, `SUSER_ID`, `IS_MEMBER`, `IS_SRVROLEMEMBER`, `IS_ROLEMEMBER`, `ORIGINAL_LOGIN`.
- System: `NEWID`, `SCOPE_IDENTITY`, `IDENT_CURRENT`, `ERROR_MESSAGE`/`ERROR_NUMBER`/`ERROR_SEVERITY`/`ERROR_STATE`/`ERROR_LINE`/`ERROR_PROCEDURE`.
- Configuration (`@@`): `@@IDENTITY`, `@@PROCID`, `@@SERVICENAME`, `@@NESTLEVEL`, `@@CURSOR_ROWS`, `@@MAX_PRECISION`, `@@DATEFIRST`, `@@LOCK_TIMEOUT`, `@@OPTIONS`.

Catalog:

- Catalog views (`sys.*`): `indexes`, `index_columns`, `key_constraints`, `foreign_key_columns`, `check_constraints`, `default_constraints`, `identity_columns`, `computed_columns`, `sql_expression_dependencies`, `triggers`, `extended_properties`, `all_objects`, `all_columns`, `sequences`, `synonyms`, `table_types`, `database_principals`, `server_principals`, `database_role_members`, `database_permissions`, `server_permissions`; compatibility views `sysobjects`, `syscolumns`, `systypes`, `sysindexes`, `sysusers`.
- INFORMATION_SCHEMA: `SCHEMATA`, `CHECK_CONSTRAINTS`, `CONSTRAINT_COLUMN_USAGE`, `CONSTRAINT_TABLE_USAGE`, `VIEW_TABLE_USAGE`, `ROUTINE_COLUMNS`, `DOMAINS`.
- System stored procedures: `sp_pkeys`, `sp_fkeys`, `sp_statistics`, `sp_special_columns`, `sp_stored_procedures`, `sp_sproc_columns`, `sp_server_info`, `sp_datatype_info`, `sp_helpindex`, `sp_helpconstraint`, `sp_helpdb`, `sp_configure`, `sp_lock`.

Other:

- Parser: reserved-keyword function names (`LEFT`/`RIGHT`), datepart keywords (`DATEADD`/`DATEPART`/…), `IIF` (desugars to `CASE`), and `TRY_CAST`/`TRY_CONVERT`/`PARSE`.
- Seam-first catalog: views are projections over SPI seams — `sys.table_types` over `catalog.Schema.TableTypes`, the security catalog over the authenticated `tds.Principal` — populating when the backend/identity supplies data and degrading to the correct empty shape otherwise.
- Catalog scalars (`OBJECT_NAME`/`COL_*`/`*PROPERTY`/`OBJECT_DEFINITION`) resolve in every clause: SELECT, ORDER BY, WHERE, HAVING, JOIN-ON, and searched-CASE WHEN conditions (the evaluators thread the catalog env, not just SubFn).
- Recursive CTEs (`WITH cte AS (anchor UNION ALL recursive)`) support the recursive arm joining real tables against the CTE, so hierarchy walks (org charts, category trees, BOM) traverse fully, not just self-contained recursion.
- Window functions `ROW_NUMBER`, `RANK`, `DENSE_RANK`, `NTILE`, `PERCENT_RANK`, `CUME_DIST`, `LAG`, `LEAD`, `FIRST_VALUE`, `LAST_VALUE` with `OVER (PARTITION BY … ORDER BY …)`. (`FIRST_VALUE`/`LAST_VALUE` use the whole partition as the frame; aggregate windows like `SUM(x) OVER (…)` and explicit frames are not yet supported.)
- Table-valued functions in `FROM`: `STRING_SPLIT(string, separator)` (one `value` row per part) and `OPENJSON(json [, path])` (`key`/`value`/`type` rows over an array or object).
- `FORMAT(value, format)`: standard numeric specifiers (`N`/`F`/`D`/`C`/`P`/`X`) and simple custom patterns, plus datetime token mapping (`yyyy`/`MM`/`dd`/`HH`/`mm`/`ss`/`MMMM`/…).
- `JSON_MODIFY(json, path, value)`: set, insert, delete (NULL value), and `append` at a JSON path.
- `JSON_PATH_EXISTS`, `JSON_ARRAY`; string `SOUNDEX`, `DIFFERENCE`, `FORMATMESSAGE`.
- System/security scalars: `NEWSEQUENTIALID`, `XACT_STATE`, `CURSOR_STATUS`, `CONNECTIONPROPERTY`, `CONTEXT_INFO`, `SESSION_CONTEXT`, `HAS_PERMS_BY_NAME`; crypto `COMPRESS`/`DECOMPRESS` (gzip) and `PWDENCRYPT`/`PWDCOMPARE` (SHA-256).
- `INDEXPROPERTY`, `INDEXKEY_PROPERTY` (catalog index metadata); `SWITCHOFFSET`, `TODATETIMEOFFSET` (datetimeoffset).
- More catalog: `sys.partitions` (one per table), `sys.database_files`/`sys.filegroups` (single-DB topology), `sys.system_objects`/`sys.stats`/`sys.stats_columns` (empty-shaped); `sp_helptrigger`/`sp_depends` (project routine data), `sp_table_privileges`/`sp_column_privileges` (empty).
- Control-of-flow: a procedural interpreter runs `IF…ELSE`, `WHILE`, `BEGIN…END`, `BREAK`, `CONTINUE`, `RETURN`, `PRINT`, `TRY…CATCH`, `THROW`, `RAISERROR`, with runtime `DECLARE`/`SET` variables. A caught error populates the `ERROR_*` family (`ERROR_MESSAGE`/`ERROR_NUMBER`/`ERROR_SEVERITY`/`ERROR_STATE`/`ERROR_LINE`/`ERROR_PROCEDURE`) inside the `CATCH` block. Opt-in: the engine routes a batch here only when it leads with a control statement; plain batches keep the flat path unchanged. (`GOTO`, `WAITFOR` not yet.)

## v1.4.0

Stored routines are now visible and scriptable through the standard catalog — additive, no breaking changes.

- **Routine catalog surface** — `sys.sql_modules` (reconstructed CREATE text), `sys.views`,
  `sys.procedures`, `sys.parameters`, and `sys.objects` widened to span tables, views, procedures,
  functions, and triggers (`U`/`V`/`P`/`FN`/`TR`); `INFORMATION_SCHEMA.VIEWS` / `ROUTINES` /
  `PARAMETERS`; and `sp_helptext` / `sp_help`. A client's *Script as -> CREATE* and object tree now read
  stored views, procedures, functions, and triggers.
- **Catalog scalar functions** — `OBJECT_NAME`, `DB_NAME`, `OBJECT_SCHEMA_NAME`, `TYPE_NAME`, backed by a
  single unified `object_id` / `database_id` scheme so `OBJECT_ID('x')` joins the catalog views and
  `OBJECT_NAME` reverses it.
- **API additions** — `tds.RoutineFunc` / `tds.RoutineTrigger` routine kinds; `catalog.Index`,
  `catalog.Check`, and `catalog.Column.Identity` / `.Computed`, so a backend can declare indexes, check
  constraints, identity, and computed columns.
- **Conformance** — `tdstest.RunConformance` now checks `Caps.Routines` <-> `RoutineStore` and that a
  written routine of each kind surfaces through the catalog views.

## v1.3.0

Extension surface foundation — additive, no breaking changes.

- **`internal/extensions` package layout** — `catalog/` (scalar functions), `views/`, `procedures/`,
  `routines/` (the shared base plus the `Runner` engine seam), and `batch/`.
- **`RoutineStore`** — persist `CREATE VIEW` / `PROCEDURE`, with read-time view expansion (FROM-a-view ->
  derived table, body qualified to the view's database).
- **Batch variables** — `DECLARE @v = …` / `SET @v = …` bound and substituted before parse, so the core
  lexer never sees a `@`.
- **Query** — `NOT IN` and aggregate-over-expression (`MAX(CASE …)`).

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
