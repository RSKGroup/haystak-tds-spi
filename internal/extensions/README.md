# `internal/extensions` — what it takes to "be SQL Server"

## What "being SQL Server" means

A SQL Server *client* — SSMS, Azure Data Studio, Power BI, Excel, `sqlcmd`, the .NET `SqlClient`, the
ODBC Driver 18 and Microsoft JDBC drivers, every BI/ETL tool that "connects to SQL Server" — speaks four
things and assumes all of them are present:

1. **TDS** — the wire protocol (PRELOGIN, LOGIN7, `SQL_BATCH`, RPC, result-set streaming).
2. **T-SQL** — the dialect (SELECT/JOIN/CTE/window/`MERGE`, plus the procedural language).
3. **The system catalog** — `sys.*`, `INFORMATION_SCHEMA.*`, and the `sp_*` catalog procedures a GUI
   calls to draw its object tree and a driver calls to describe a result.
4. **Behaviors** — `@@`-functions, `SERVERPROPERTY`, default schema = `dbo`, identifier quoting, and so on.

The **core** (`internal/wire`, `internal/tsql`, `internal/exec`, `internal/engine`) supplies the wire and a
generic SQL engine. **This directory supplies the SQL-Server-specific surface** — a long tail.

## How to read this list

- **✓ shipped · ◻ not yet.** Status is verified against the code, not assumed: the `funcs` registry,
  `exec.evalFunc`, the engine `probe`/`probeValue`/`serverProperty`, the `sysviews`/`infoschema`
  dispatches, the `execProc` dispatch, and the `tsql` parser. (Inventory reconciled 2026-06-14.)
- Organized **per `extensions/<folder>`** (plus a final *Core language* section for things that live in
  `tsql`/`exec`/`engine`), each broken out **by feature group**. A group with a mix shows a ✓ row and a ◻ row.
- `→` items in prior versions are folded into ✓/◻ — there is no separate "next" column.

---

## `catalog/` — describe & compute like SQL Server

### `funcs/` — scalar functions *(registry + engine `probe`)*

| Group | Functions | Status |
| --- | --- | --- |
| Catalog & metadata | `DB_ID`, `DB_NAME`, `OBJECT_ID`, `OBJECT_NAME`, `OBJECT_SCHEMA_NAME`, `SCHEMA_ID`, `SCHEMA_NAME`, `TYPE_NAME`, `HAS_DBACCESS`, `QUOTENAME`, `ORIGINAL_DB_NAME` | ✓ |
| Catalog & metadata | `COL_NAME`, `COL_LENGTH`, `TYPE_ID`, `OBJECT_DEFINITION`, `OBJECTPROPERTY(EX)`, `COLUMNPROPERTY`, `INDEXPROPERTY`, `INDEXKEY_PROPERTY`, `STATS_DATE`, `FILEGROUP_NAME`, `FILE_NAME` | ◻ |
| Server & database props | `SERVERPROPERTY`, `DATABASEPROPERTYEX` | ✓ |
| Server & database props | `CONNECTIONPROPERTY`, `ASSEMBLYPROPERTY`, `COLLATIONPROPERTY`, `FILEPROPERTY`, `FULLTEXTSERVICEPROPERTY` | ◻ |
| Security & session identity | `SYSTEM_USER`, `CURRENT_USER`, `SESSION_USER`, `USER`, `USER_NAME`, `SUSER_NAME`, `SUSER_SNAME`, `HOST_NAME`, `APP_NAME` | ✓ |
| Security & session identity | `USER_ID`, `SUSER_ID`, `SUSER_SID`, `IS_MEMBER`, `IS_SRVROLEMEMBER`, `IS_ROLEMEMBER`, `PERMISSIONS`, `HAS_PERMS_BY_NAME`, `ORIGINAL_LOGIN`, `CONTEXT_INFO`, `SESSION_CONTEXT` | ◻ |
| `@@` config & session | `@@VERSION`, `@@SERVERNAME`, `@@SPID`, `@@LANGUAGE`, `@@ROWCOUNT`, `@@ERROR`, `@@TRANCOUNT`, `@@FETCH_STATUS` | ✓ |
| `@@` config & session | `@@SERVICENAME`, `@@IDENTITY`, `@@NESTLEVEL`, `@@MAX_PRECISION`, `@@OPTIONS`, `@@DATEFIRST`, `@@LOCK_TIMEOUT`, `@@CURSOR_ROWS`, `@@PROCID`, `@@CONNECTIONS`, `@@CPU_BUSY`, `@@PACK_RECEIVED`, `@@TOTAL_ERRORS` | ◻ |
| String | `LEN`, `DATALEN`, `UPPER`, `LOWER`, `LTRIM`, `RTRIM`, `TRIM`, `SUBSTRING`, `REPLACE`, `CONCAT` | ✓ |
| String | `CHARINDEX`, `PATINDEX`, `STUFF`, `LEFT`, `RIGHT`, `REPLICATE`, `SPACE`, `REVERSE`, `CONCAT_WS`, `STRING_AGG`, `STRING_SPLIT`, `STRING_ESCAPE`, `TRANSLATE`, `FORMATMESSAGE`, `UNICODE`, `NCHAR`, `CHAR`, `ASCII`, `SOUNDEX`, `DIFFERENCE`, `STR` | ◻ |
| Numeric & math | `ABS` | ✓ |
| Numeric & math | `CEILING`, `FLOOR`, `ROUND`, `POWER`, `SQRT`, `SQUARE`, `EXP`, `LOG`, `LOG10`, `SIN`, `COS`, `TAN`, `COT`, `ASIN`, `ACOS`, `ATAN`, `ATN2`, `PI`, `RAND`, `SIGN`, `DEGREES`, `RADIANS` | ◻ |
| Date & time | `GETDATE`, `GETUTCDATE`, `SYSDATETIME`, `SYSUTCDATETIME`, `YEAR`, `MONTH`, `DAY` | ✓ |
| Date & time | `SYSDATETIMEOFFSET`, `CURRENT_TIMESTAMP`, `DATEADD`, `DATEDIFF`, `DATEDIFF_BIG`, `DATEPART`, `DATENAME`, `*FROMPARTS`, `EOMONTH`, `SWITCHOFFSET`, `TODATETIMEOFFSET`, `ISDATE`, `DATETRUNC` | ◻ |
| Conversion | `CAST`, `CONVERT` (basic — style ignored) | ✓ |
| Conversion | `TRY_CAST`, `TRY_CONVERT`, `PARSE`, `TRY_PARSE`, `FORMAT`, `CONVERT` styles | ◻ |
| Logical / NULL | `ISNULL`, `COALESCE`, `NULLIF`, `CASE` (searched + simple) | ✓ |
| Logical / NULL | `IIF`, `CHOOSE` | ◻ |
| Aggregate | `COUNT`, `SUM`, `MIN`, `MAX`, `AVG` (+ GROUP BY / HAVING, aggregate-over-expression) | ✓ |
| Aggregate | `COUNT_BIG`, `STDEV`, `STDEVP`, `VAR`, `VARP`, `GROUPING`, `GROUPING_ID`, `CHECKSUM_AGG`, `STRING_AGG`, `APPROX_COUNT_DISTINCT` | ◻ |
| Window & ranking | `OVER`/`PARTITION BY`, `ROW_NUMBER`, `RANK`, `DENSE_RANK`, `NTILE`, `LAG`, `LEAD`, `FIRST_VALUE`, `LAST_VALUE`, `PERCENT_RANK`, `CUME_DIST`, `PERCENTILE_CONT/DISC`, framed/running aggregates (`ROWS`/`RANGE`) | ◻ |
| JSON | `ISJSON`, `JSON_VALUE`, `JSON_QUERY`, `JSON_MODIFY`, `JSON_PATH_EXISTS`, `JSON_OBJECT`, `JSON_ARRAY`, `OPENJSON`, `FOR JSON` | ◻ |
| XML | `xml` type, `.value()`/`.query()`/`.nodes()`/`.exist()`/`.modify()`, `FOR XML`, `OPENXML` | ◻ |
| Crypto & hashing | `HASHBYTES`, `CHECKSUM`, `BINARY_CHECKSUM`, `COMPRESS`, `DECOMPRESS`, `ENCRYPTBYKEY`/`DECRYPTBYKEY`, `PWDENCRYPT`, `PWDCOMPARE` | ◻ |

### `sys.*` catalog views *(today in `internal/sysviews`)*

| Group | Views | Status |
| --- | --- | --- |
| Databases, schemas, objects, modules | `sys.databases`, `sys.schemas`, `sys.tables`, `sys.objects` (`U`/`V`/`P`/`FN`/`TR`), `sys.views`, `sys.procedures`, `sys.sql_modules`, `sys.parameters` | ✓ |
| Databases, schemas, objects, modules | `sys.all_objects`, `sys.all_views`, `sys.system_objects`, `sys.numbered_procedures` | ◻ |
| Columns & types | `sys.columns`, `sys.types` | ✓ |
| Columns & types | `sys.all_columns`, `sys.table_types`, `sys.assembly_types` | ◻ |
| Keys, indexes, constraints | `sys.foreign_keys`, `sys.foreign_key_columns`, `sys.key_constraints`, `sys.indexes`, `sys.index_columns`, `sys.check_constraints`, `sys.default_constraints` | ✓ |
| Keys, indexes, constraints | `sys.stats`, `sys.stats_columns`, `sys.partitions`, `sys.partition_schemes`, `sys.allocation_units` | ◻ |
| Identity & computed | `sys.identity_columns`, `sys.computed_columns` | ✓ |
| Identity & computed | `sys.masked_columns`, `sys.column_encryption_keys` | ◻ |
| Dependencies | `sys.sql_expression_dependencies`, `sys.sql_dependencies`, `sys.dm_sql_referenced_entities`, `sys.dm_sql_referencing_entities` | ◻ |
| Security: principals & permissions | `sys.database_principals`, `sys.server_principals`, `sys.database_permissions`, `sys.server_permissions`, `sys.database_role_members`, `sys.server_role_members` | ◻ |
| Dynamic management views (DMVs) | `sys.dm_exec_sessions`, `sys.dm_exec_connections`, `sys.dm_exec_requests`, `sys.dm_exec_query_stats`, `sys.dm_os_*`, `sys.dm_db_*`, `sys.dm_tran_*` | ◻ |
| Compatibility (`sys.sys*`) | `sys.sysobjects`, `sys.syscolumns`, `sys.systypes`, `sys.sysindexes`, `sys.sysusers`, `sys.sysdatabases`, `sys.sysprocesses` | ◻ |
| Other objects | `sys.triggers`, `sys.sequences`, `sys.synonyms`, `sys.extended_properties`, `sys.service_queues`, `sys.filegroups`, `sys.data_spaces` | ◻ |

### `INFORMATION_SCHEMA.*` *(today in `internal/infoschema`)*

| Group | Views | Status |
| --- | --- | --- |
| Tables, columns, views, routines | `TABLES`, `COLUMNS`, `VIEWS`, `ROUTINES`, `PARAMETERS` | ✓ |
| Tables, columns, views, routines | `ROUTINE_COLUMNS`, `VIEW_COLUMN_USAGE`, `VIEW_TABLE_USAGE`, `COLUMN_DOMAIN_USAGE` | ◻ |
| Constraints | `TABLE_CONSTRAINTS`, `KEY_COLUMN_USAGE`, `REFERENTIAL_CONSTRAINTS` | ✓ |
| Constraints | `CHECK_CONSTRAINTS`, `CONSTRAINT_COLUMN_USAGE`, `CONSTRAINT_TABLE_USAGE` | ◻ |
| Other | `SCHEMATA`, `DOMAINS`, `DOMAIN_CONSTRAINTS`, `COLUMN_PRIVILEGES`, `TABLE_PRIVILEGES` | ◻ |

### Catalog stored procedures *(today in engine `proc.go`)*

| Group | Procedures | Status |
| --- | --- | --- |
| ODBC / driver metadata | `sp_databases`, `sp_tables`, `sp_columns`, `sp_pkeys`, `sp_fkeys`, `sp_statistics`, `sp_special_columns`, `sp_stored_procedures`, `sp_sproc_columns` | ✓ |
| ODBC / driver metadata | `sp_columns_ex`, `sp_tables_ex`, `sp_column_privileges`, `sp_table_privileges` | ◻ |
| Help & scripting | `sp_help`, `sp_helptext`, `sp_helpindex`, `sp_helpconstraint` | ✓ |
| Help & scripting | `sp_helpdb`, `sp_helpuser`, `sp_helprotect`, `sp_helptrigger`, `sp_helpstats`, `sp_depends` | ◻ |
| Server & type info | `sp_server_info`, `sp_datatype_info`, `sp_server_diagnostics` | ◻ |
| Admin & object management | `sp_executesql`, `sp_rename`, `sp_addextendedproperty`, `sp_who`, `sp_lock`, `sp_configure`, `sp_addmessage` | ◻ |

---

## `procedures/` — stored procedures + the procedural language

| Group | Feature | Status |
| --- | --- | --- |
| EXEC & parameters | `CREATE / ALTER / DROP PROCEDURE`, `EXEC proc @a = …` / positional + parameter substitution | ✓ |
| EXEC & parameters | `CREATE OR ALTER PROCEDURE`, `WITH RECOMPILE`/`ENCRYPTION`/`EXECUTE AS`, default / `OUTPUT` / table-valued params, `EXEC @rc = proc` + return codes | ◻ |
| control/ — flow | `IF … ELSE`, `WHILE`, `BEGIN … END`, `BREAK`, `CONTINUE`, `RETURN`, `GOTO`, `WAITFOR` | ◻ |
| control/ — errors | `TRY … CATCH`, `THROW`, `RAISERROR`, `PRINT`, `ERROR_MESSAGE()`/`ERROR_NUMBER()`/…, `XACT_STATE()` | ◻ |
| Variables | `DECLARE @v = …` (scalar, via `batch/`), `SET @v = …` | ✓ |
| Variables | `SELECT @v = col`, `DECLARE @t TABLE(...)`, multi-var assignment from query, compound assignment (`+=`) | ◻ |
| Cursors | `DECLARE … CURSOR`, `OPEN`, `FETCH`, `CLOSE`, `DEALLOCATE`, cursor variables | ◻ |
| Transactions | `BEGIN`/`COMMIT`/`ROLLBACK TRANSACTION`, `SAVE TRANSACTION`, `SET XACT_ABORT`, isolation levels, `@@TRANCOUNT` semantics | ◻ |
| Temp tables & table types | `#local` / `##global` temp tables, `CREATE TYPE … AS TABLE`, table-valued parameters | ◻ |
| Triggers & UDFs (execution) | `CREATE TRIGGER` run (`INSERTED`/`DELETED`), `INSTEAD OF`/DDL triggers; `CREATE FUNCTION` run (scalar / inline-TVF / multi-statement-TVF) — *capture & scripting is ✓; running is ◻* | ◻ |
| Dynamic SQL | `sp_executesql`, `EXEC('…')`, parameterized dynamic SQL | ◻ |

---

## `routines/` — the shared seam (infrastructure, not a SQL feature)

| Piece | Status |
| --- | --- |
| `Runner` seam (`Exec`/`RunQuery`/`CurrentDB`), DDL-text helpers, `QualifyDB` (body resolves in the routine's db), `ScriptDefinition` (reconstruct CREATE text) | ✓ |
| Backends persist definitions through the **public** `tds.RoutineStore` (gated by `Caps.Routines`); read-time view expansion | ✓ |

---

## `views/` — stored views

| Feature | Status |
| --- | --- |
| `CREATE / ALTER / DROP VIEW` + read-time expansion (FROM-a-view → derived table, body resolves via `QualifyDB`) | ✓ |
| `CREATE OR ALTER VIEW`, column-rename list `CREATE VIEW v (a, b) AS …` | ◻ |
| `WITH SCHEMABINDING`, `WITH CHECK OPTION`, `WITH ENCRYPTION` | ◻ |
| updatable views / `INSTEAD OF` triggers, indexed (materialized) views, partitioned views | ◻ |

---

## `batch/` — batch variables

| Feature | Status |
| --- | --- |
| `DECLARE @v = …` / `SET @v = …` bind + string-literal-aware substitution (core lexer never sees `@`) | ✓ |
| `GO` batch separator (+ count), `:setvar` / sqlcmd directives, variable scoping across control flow | ◻ |

---

## Core language (`tsql` / `exec` / `engine`) — not in this dir, listed for completeness

A client's idea of "SQL Server" includes the language itself; these land in the **core**, changed with care.

| Group | Feature | Status |
| --- | --- | --- |
| Query — relational | `SELECT`, `WHERE`, INNER/LEFT/RIGHT/FULL/CROSS `JOIN` + `ON`, CTEs (incl. chained), subqueries, `EXISTS`, `IN`/`NOT IN`, `LIKE`, `BETWEEN`, `IS [NOT] NULL`, `GROUP BY`, `HAVING`, `ORDER BY`, `DISTINCT`, `TOP`, `OFFSET`/`FETCH`, `UNION`/`INTERSECT`/`EXCEPT`, `CASE` | ✓ |
| Query — advanced | window functions, `PIVOT`/`UNPIVOT`, `CROSS`/`OUTER APPLY`, `MERGE`, `TOP … WITH TIES`, `GROUPING SETS`/`ROLLUP`/`CUBE`, recursive CTEs at depth, `TABLESAMPLE`, table hints (`WITH (NOLOCK)`), `OPTION (…)` query hints | ◻ |
| DML | `INSERT`, `UPDATE`, `DELETE` (+ `WHERE`) | ✓ |
| DML | `MERGE`, `INSERT … OUTPUT`, `UPDATE … FROM`, `DELETE … FROM`, `SELECT … INTO`, `TRUNCATE TABLE`, `BULK INSERT`, `INSERT … EXEC` | ◻ |
| DDL — tables & columns | `CREATE / ALTER / DROP TABLE` (add/drop column) | ✓ |
| DDL — tables & columns | constraints DDL (PK/FK/UNIQUE/CHECK/DEFAULT), `IDENTITY(seed,incr)`, computed columns, `ALTER COLUMN`, `CREATE/ALTER/DROP INDEX`, `CREATE/UPDATE/DROP STATISTICS` | ◻ |
| DDL — other objects | `CREATE/ALTER/DROP FUNCTION`/`TRIGGER`/`SEQUENCE`/`SYNONYM`/`SCHEMA`/`TYPE`, `NEXT VALUE FOR`, `CREATE OR ALTER` | ◻ |
| Data types | bit, int, bigint, decimal/numeric, float/real, nvarchar/varchar, varbinary, datetime2, date/time, uniqueidentifier (value model) | ✓ |
| Data types | datetimeoffset, smalldatetime, money/smallmoney, sql_variant, xml, hierarchyid, geometry/geography, rowversion, image/text/ntext, CLR UDTs, collation & precision/scale edge cases | ◻ |
| Security DDL | `GRANT`/`REVOKE`/`DENY`, `CREATE/ALTER/DROP USER`/`LOGIN`/`ROLE`, `ALTER ROLE … ADD MEMBER`, `EXECUTE AS`/`REVERT` | ◻ |
| Temporal & full-text | system-versioned (temporal) tables + `FOR SYSTEM_TIME`; `CONTAINS`/`FREETEXT`/`CONTAINSTABLE`, `CREATE FULLTEXT INDEX` | ◻ |
| SET options & session | `SET` (no-op parse, e.g. `SET NOCOUNT ON`), `USE [db]` | ✓ |
| SET options & session | `SET ANSI_NULLS`/`QUOTED_IDENTIFIER`/`ROWCOUNT`/`DATEFIRST`/`IDENTITY_INSERT`/isolation level **with effect** (not just parse) | ◻ |
| Admin | `DBCC` (CHECKDB / SHRINK / FREEPROCCACHE / …), `BACKUP`/`RESTORE`, linked servers (`OPENROWSET`/`OPENQUERY`), Service Broker, bulk ops, SQL Agent | ◻ |

---

### How a row gets checked off

1. Find the owning package above (catalog function → `catalog/funcs/`; procedural construct →
   `procedures/control/`; batch variable → `extensions/batch`; view option → `views/`). Core-language rows
   are filed in `internal/tsql` + `internal/exec`.
2. Add a file there — it depends *down* on `routines`/`tds`, never up into the engine.
3. Register it (function registry, keyword dispatch, DDL head-match), add a test, **and flip the ✓ here in
   the same change** — the list is only worth anything if it matches the code.

See [`../../ARCHITECTURE.md`](../../ARCHITECTURE.md) for the core-vs-extensions map and the `Runner` seam,
and [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) for the step-by-step recipes.
