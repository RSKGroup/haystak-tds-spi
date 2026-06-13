# `internal/extensions` — what it takes to "be SQL Server"

## What "being SQL Server" means

A SQL Server *client* — SSMS, Azure Data Studio, Power BI, Excel, `sqlcmd`, the .NET `SqlClient`, the
ODBC Driver 18 and Microsoft JDBC drivers, every BI/ETL tool that "connects to SQL Server" — speaks four
things and assumes all of them are present:

1. **TDS** — the wire protocol (PRELOGIN, LOGIN7, `SQL_BATCH`, RPC, result-set streaming).
2. **T-SQL** — the dialect (SELECT/JOIN/CTE/window/`MERGE`, plus the procedural language: `DECLARE`,
   `IF`, `WHILE`, `TRY…CATCH`, cursors, transactions).
3. **The system catalog** — `sys.*`, `INFORMATION_SCHEMA.*`, and the `sp_*` catalog procedures a GUI
   calls to draw its object tree and a driver calls to describe a result.
4. **Behaviors** — `@@`-functions, error/`@@ROWCOUNT` conventions, `DB_ID()`/`HAS_DBACCESS()` gating,
   identifier quoting, default schema = `dbo`, and so on.

"Being SQL Server" means enough of that surface exists that those clients connect and work **unmodified** —
they enumerate databases/schemas/tables/columns, browse object trees, run queries and stored procedures,
and never notice they're not talking to Microsoft's server.

The **core** (`internal/wire`, `internal/tsql`, `internal/exec`, `internal/engine`) supplies the wire and a
generic SQL engine. **This directory supplies the SQL-Server-specific surface** — and it is a long tail.
What follows is the gap, package by package: **✓ shipped · → next (the script-out slice) · ◻ not yet**.
The list is deliberately large; the point of "be SQL Server" is that there's always one more thing a
client expects.

---

## Shipped today (✓) — the beachhead

The surface that already works (each row is ✓ in the detail below). Everything else in this document is
still owed.

- **`catalog/funcs/`** — `DB_ID`, `HAS_DBACCESS`, `SCHEMA_NAME`, `SCHEMA_ID`, `OBJECT_ID`, `QUOTENAME`
- **`sys.*`** — `sys.databases`, `sys.tables`, `sys.columns`, `sys.schemas`, `sys.types`, `sys.foreign_keys`
- **`INFORMATION_SCHEMA`** — `TABLES`, `COLUMNS`
- **catalog procs** — `sp_databases`, `sp_tables`, `sp_columns`
- **`procedures/`** — `CREATE / ALTER / DROP PROCEDURE`, `EXEC proc @a = …` + parameter substitution
- **`views/`** — `CREATE / ALTER / DROP VIEW` + read-time expansion (body resolves via `QualifyDB`)
- **`routines/`** — the `Runner` seam + DDL helpers + `tds.RoutineStore` persistence
- **`batch/`** — `DECLARE @v = …` / `SET @v = …` bind + string-literal-aware substitution
- **core query** (engine, not this dir) — SELECT, all JOIN kinds + `ON`, chained CTEs, CASE, GROUP BY/
  HAVING/ORDER BY/DISTINCT/TOP, UNION/INTERSECT/EXCEPT, subqueries/EXISTS, LIKE/BETWEEN/IS NULL, the
  generic scalar functions, and the latest Q1 bits — **`NOT IN`** and **aggregate-over-expression**
  (`MAX(CASE …)`)

---

## Next up (→) — make stored routines visible & scriptable

We already *persist* `CREATE VIEW`/`PROCEDURE` via `RoutineStore`, but a client can't see them yet. The
highest-value next slice is the **script-out surface** — what a client's *Script as → CREATE* and object
tree read — built in **dependency order**:

1. **`OBJECT_NAME()`** (+ `DB_NAME`, `OBJECT_SCHEMA_NAME`, `TYPE_NAME`) — `catalog/funcs/`. The **hard
   dependency**: without it the routine views below are unjoinable, so it comes first.
2. **`sys.sql_modules`** (the verbatim CREATE body — the script-out payload), **`sys.objects`** widened
   to `V`/`P`/`FN`/`TR`, **`sys.views`**, **`sys.procedures`**, **`sys.parameters`** — `internal/sysviews`.
3. **`INFORMATION_SCHEMA.VIEWS` / `ROUTINES` / `PARAMETERS`** — `internal/infoschema` (the portable path).
4. **`sp_helptext`**, **`sp_help`** — `procedures/` catalog procs (the classic definition dump).

That turns "we stored the DDL" into "the client shows it and scripts it back out." After it comes the full
browsable tree + driver metadata (`sys.indexes`/`index_columns`/`key_constraints`/`foreign_key_columns`,
`sp_pkeys`/`sp_fkeys`/`sp_stored_procedures`, `@@VERSION`/`SERVERPROPERTY`). *Executing* a captured routine
(control flow, variables-from-query, cursors, transactions, the scalar/date function library) is a later,
separate effort — none of it is needed to capture, catalog, or script-out.

---

## `catalog/` — describe & compute like SQL Server

The "what's in this server, and may I see it" surface. GUIs gate their trees on these; drivers describe
results with them.

### Scalar system / metadata functions (`catalog/funcs/`)

| Function(s) | Status |
| --- | --- |
| `DB_ID`, `HAS_DBACCESS`, `SCHEMA_NAME`, `SCHEMA_ID`, `OBJECT_ID`, `QUOTENAME` | ✓ |
| `OBJECT_NAME`, `DB_NAME`, `OBJECT_SCHEMA_NAME`, `TYPE_NAME` | → next |
| `COL_NAME`, `COL_LENGTH`, `TYPE_ID` | ◻ |
| `OBJECTPROPERTY`, `OBJECTPROPERTYEX`, `COLUMNPROPERTY`, `INDEXPROPERTY`, `DATABASEPROPERTYEX`, `SERVERPROPERTY` | ◻ |
| `USER_NAME`, `USER_ID`, `SUSER_NAME`, `SUSER_SNAME`, `SUSER_ID`, `IS_MEMBER`, `IS_SRVROLEMEMBER`, `PERMISSIONS` | ◻ |
| `@@VERSION`, `@@SERVERNAME`, `@@SERVICENAME`, `@@SPID`, `@@ROWCOUNT`, `@@ERROR`, `@@IDENTITY`, `@@TRANCOUNT`, `@@NESTLEVEL`, `@@LANGUAGE`, `@@MAX_PRECISION`, `@@OPTIONS` | ◻ |
| `NEWID`, `NEWSEQUENTIALID`, `HOST_NAME`, `APP_NAME`, `CONNECTIONPROPERTY`, `CONTEXT_INFO`, `ORIGINAL_LOGIN` | ◻ |

### Catalog views — `sys.*` *(today in `internal/sysviews`; slated under `catalog/`)*

| View(s) | Status |
| --- | --- |
| `sys.databases`, `sys.tables`, `sys.columns`, `sys.schemas`, `sys.types`, `sys.foreign_keys` | ✓ |
| `sys.objects` — exists but **tables only**; widen to `V`/`P`/`FN`/`TR` | ✓ partial |
| `sys.sql_modules` (verbatim CREATE body), `sys.views`, `sys.procedures`, `sys.parameters` | → next |
| `sys.sql_expression_dependencies` | ◻ |
| `sys.indexes`, `sys.index_columns`, `sys.key_constraints`, `sys.foreign_key_columns`, `sys.check_constraints`, `sys.default_constraints` | ◻ |
| `sys.identity_columns`, `sys.computed_columns`, `sys.triggers`, `sys.partitions`, `sys.tables` extended cols, `sys.extended_properties` | ◻ |
| `sys.database_principals`, `sys.server_principals`, `sys.database_permissions`, `sys.schemas` perms | ◻ |
| `sys.dm_exec_sessions`, `sys.dm_exec_connections`, `sys.dm_exec_requests`, `sys.dm_os_*` (DMVs) | ◻ |
| compat views: `sys.sysobjects`, `sys.syscolumns`, `sys.systypes`, `sys.sysindexes`, `sys.sysusers` | ◻ |

### `INFORMATION_SCHEMA.*` *(today in `internal/infoschema`; slated under `catalog/`)*

| View(s) | Status |
| --- | --- |
| `TABLES`, `COLUMNS`, `TABLE_CONSTRAINTS`, `KEY_COLUMN_USAGE`, `REFERENTIAL_CONSTRAINTS` | ✓ |
| `VIEWS`, `ROUTINES`, `PARAMETERS` | → next |
| `ROUTINE_COLUMNS`, `CONSTRAINT_COLUMN_USAGE`, `CHECK_CONSTRAINTS`, `SCHEMATA`, `DOMAINS`, `VIEW_COLUMN_USAGE`, `VIEW_TABLE_USAGE` | ◻ |

### Catalog stored procedures

| Proc(s) | Status |
| --- | --- |
| `sp_databases`, `sp_tables`, `sp_columns` | ✓ |
| `sp_helptext`, `sp_help` | → next |
| `sp_helpdb`, `sp_helpindex`, `sp_helpconstraint` | ◻ |
| `sp_pkeys`, `sp_fkeys`, `sp_special_columns`, `sp_statistics`, `sp_stored_procedures`, `sp_sproc_columns` | ◻ |
| `sp_server_info`, `sp_datatype_info`, `sp_tables_ex`, `sp_columns_ex`, `sp_table_privileges`, `sp_column_privileges` | ◻ |

---

## `procedures/` — stored procedures + the procedural language

| Feature | Status |
| --- | --- |
| `CREATE / ALTER / DROP PROCEDURE`, `EXEC proc @a = …` + named/positional parameter substitution | ✓ |
| `CREATE OR ALTER PROCEDURE`, `WITH RECOMPILE`, `WITH ENCRYPTION`, default/`OUTPUT`/table-valued parameters | ◻ |
| `sp_executesql`, `EXEC('dynamic sql')`, `EXEC @rc = proc`, return codes | ◻ |
| **control flow** (`procedures/control/`): `IF … ELSE`, `WHILE`, `BEGIN … END`, `BREAK`, `CONTINUE`, `RETURN`, `GOTO`, `WAITFOR` | ◻ |
| **error handling**: `TRY … CATCH`, `THROW`, `RAISERROR`, `ERROR_MESSAGE()`/`ERROR_NUMBER()`/… | ◻ |
| **variables**: `DECLARE @v = …` (✓ via `extensions/batch`), `SET @v = …`, `SELECT @v = col`, table variables `DECLARE @t TABLE(...)` | ◻ (scalar DECLARE/SET ✓ in batch) |
| **cursors**: `DECLARE … CURSOR`, `OPEN`, `FETCH`, `CLOSE`, `DEALLOCATE` | ◻ |
| **transactions**: `BEGIN/COMMIT/ROLLBACK TRANSACTION`, `SAVE TRANSACTION`, `SET XACT_ABORT`, `@@TRANCOUNT` | ◻ |
| **temp tables**: `#local`, `##global`, table-valued parameters | ◻ |
| triggers (`CREATE TRIGGER`, `INSERTED`/`DELETED`), user-defined functions (`CREATE FUNCTION`, scalar/inline-TVF/multi-statement-TVF) | ◻ |

---

## `routines/` — the shared base (the seam)

Infrastructure, not a SQL feature — the contract views/procedures build on so they never import the engine.

| Piece | Status |
| --- | --- |
| `Runner` seam (`Exec` / `RunQuery` / `CurrentDB`); DDL-text helpers; `QualifyDB` (body resolves in the routine's db) | ✓ |
| Backends persist definitions through the **public** `tds.RoutineStore` (gated by `Caps.Routines`) | ✓ |

---

## `views/` — stored views

| Feature | Status |
| --- | --- |
| `CREATE / ALTER / DROP VIEW` + read-time expansion (FROM-a-view → derived table, body resolves via `QualifyDB`) | ✓ |
| `CREATE OR ALTER VIEW`, column-rename list `CREATE VIEW v (a, b) AS …` | ◻ |
| `WITH SCHEMABINDING`, `WITH CHECK OPTION`, `WITH ENCRYPTION` | ◻ |
| updatable views / `INSTEAD OF` triggers, indexed (materialized) views, partitioned views | ◻ |

---

## Beyond the four packages — core T-SQL the engine still owes

These are *language* features, not catalog/objects, so they land in the **core** (`tsql`/`exec`/`engine`),
not here — but a client's idea of "SQL Server" includes them, so they're listed for completeness.

| Area | Have (✓) / Owe (◻) |
| --- | --- |
| query | ✓ SELECT, WHERE, INNER/LEFT/RIGHT/FULL/CROSS JOIN + ON, CTEs (incl. chained), subqueries, EXISTS, `IN`/`NOT IN`, LIKE, BETWEEN, IS [NOT] NULL, GROUP BY, HAVING, ORDER BY, DISTINCT, TOP, OFFSET/FETCH, UNION/INTERSECT/EXCEPT, CASE, aggregate-over-expression |
| query | ◻ window functions (`OVER`, `PARTITION BY`, `ROW_NUMBER`/`RANK`/`DENSE_RANK`/`NTILE`/`LAG`/`LEAD`, running/framed aggregates), `PIVOT`/`UNPIVOT`, `CROSS`/`OUTER APPLY`, `MERGE`, `TOP … WITH TIES`, `GROUPING SETS`/`ROLLUP`/`CUBE`, recursive CTEs at depth |
| scalar fns | ✓ LEN, DATALEN, UPPER, LOWER, LTRIM/RTRIM/TRIM, SUBSTRING, REPLACE, CONCAT, ISNULL, COALESCE, NULLIF, ABS, YEAR/MONTH/DAY, GETDATE/GETUTCDATE, CAST |
| scalar fns | ◻ `CONVERT(… , style)`, `TRY_CAST`/`TRY_CONVERT`, `FORMAT`, `IIF`, `CHOOSE`, `DATEADD`/`DATEDIFF`/`DATEPART`/`DATENAME`/`EOMONTH`, `CHARINDEX`/`PATINDEX`/`STUFF`/`STRING_AGG`/`STRING_SPLIT`/`REPLICATE`/`SPACE`/`LEFT`/`RIGHT`, `ROUND`/`CEILING`/`FLOOR`/`POWER`/`SQRT`, `ISNUMERIC`/`ISDATE` |
| types | ◻ `uniqueidentifier`, `datetime2`/`datetimeoffset`/`time`, `decimal`/`numeric` scale fidelity, `varbinary`, `sql_variant`, `xml`, `hierarchyid`, `geography`/`geometry`, collations |
| DML/DDL | ◻ `INSERT … OUTPUT`, `UPDATE … FROM`, `DELETE … FROM`, `MERGE`, `SELECT … INTO`, `CREATE/ALTER TABLE` DDL, computed columns, constraints |

---

### How a row gets checked off

1. Find the owning package above (catalog function → `catalog/funcs/`; procedural construct →
   `procedures/control/`; batch variable → `extensions/batch`; view option → `views/`).
2. Add a file there — it depends *down* on `routines`/`tds`, never up into the engine.
3. Register it (function registry, keyword dispatch, DDL head-match) and add a test.

Core-language rows (the last table) are filed in `internal/tsql` + `internal/exec`, changed with care.

See [`../../ARCHITECTURE.md`](../../ARCHITECTURE.md) for the core-vs-extensions map and the `Runner` seam,
and [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) for the step-by-step recipes.
