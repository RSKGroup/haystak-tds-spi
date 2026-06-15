# SQL Server surface — command-level checklist

What it takes to "be SQL Server," one command per line, grouped by SQL Server's own categories.
`[x]` = implemented (verified against the code), `[ ]` = not yet. This is a checklist, not a design doc —
the design lives in [ARCHITECTURE.md](../../ARCHITECTURE.md) and [CONTRIBUTING.md](../../CONTRIBUTING.md).

---

## Functions

### Aggregate

| Element | Status |
| --- | --- |
| `COUNT` | ✅ |
| `SUM` | ✅ |
| `MIN` | ✅ |
| `MAX` | ✅ |
| `AVG` | ✅ |
| `COUNT_BIG` | ◻ |
| `STDEV` | ◻ |
| `STDEVP` | ◻ |
| `VAR` | ◻ |
| `VARP` | ◻ |
| `GROUPING` | ◻ |
| `GROUPING_ID` | ◻ |
| `CHECKSUM_AGG` | ◻ |
| `STRING_AGG` | ◻ |
| `APPROX_COUNT_DISTINCT` | ◻ |

### String

| Element | Status |
| --- | --- |
| `LEN` | ✅ |
| `DATALEN` | ✅ |
| `UPPER` | ✅ |
| `LOWER` | ✅ |
| `LTRIM` | ✅ |
| `RTRIM` | ✅ |
| `TRIM` | ✅ |
| `SUBSTRING` | ✅ |
| `REPLACE` | ✅ |
| `CONCAT` | ✅ |
| `QUOTENAME` | ✅ |
| `CHARINDEX` | ◻ |
| `PATINDEX` | ◻ |
| `STUFF` | ◻ |
| `LEFT` | ◻ |
| `RIGHT` | ◻ |
| `REPLICATE` | ◻ |
| `SPACE` | ◻ |
| `REVERSE` | ◻ |
| `CONCAT_WS` | ◻ |
| `STRING_ESCAPE` | ◻ |
| `STRING_SPLIT` | ◻ |
| `TRANSLATE` | ◻ |
| `FORMATMESSAGE` | ◻ |
| `UNICODE` | ◻ |
| `NCHAR` | ◻ |
| `CHAR` | ◻ |
| `ASCII` | ◻ |
| `SOUNDEX` | ◻ |
| `DIFFERENCE` | ◻ |
| `STR` | ◻ |

### Date & time

| Element | Status |
| --- | --- |
| `GETDATE` | ✅ |
| `GETUTCDATE` | ✅ |
| `SYSDATETIME` | ✅ |
| `SYSUTCDATETIME` | ✅ |
| `YEAR` | ✅ |
| `MONTH` | ✅ |
| `DAY` | ✅ |
| `SYSDATETIMEOFFSET` | ◻ |
| `CURRENT_TIMESTAMP` | ◻ |
| `DATEADD` | ◻ |
| `DATEDIFF` | ◻ |
| `DATEDIFF_BIG` | ◻ |
| `DATEPART` | ◻ |
| `DATENAME` | ◻ |
| `DATEFROMPARTS` | ◻ |
| `DATETIMEFROMPARTS` | ◻ |
| `EOMONTH` | ◻ |
| `SWITCHOFFSET` | ◻ |
| `TODATETIMEOFFSET` | ◻ |
| `ISDATE` | ◻ |
| `DATETRUNC` | ◻ |

### Mathematical

| Element | Status |
| --- | --- |
| `ABS` | ✅ |
| `CEILING` | ◻ |
| `FLOOR` | ◻ |
| `ROUND` | ◻ |
| `POWER` | ◻ |
| `SQRT` | ◻ |
| `SQUARE` | ◻ |
| `EXP` | ◻ |
| `LOG` | ◻ |
| `LOG10` | ◻ |
| `SIN` | ◻ |
| `COS` | ◻ |
| `TAN` | ◻ |
| `COT` | ◻ |
| `ASIN` | ◻ |
| `ACOS` | ◻ |
| `ATAN` | ◻ |
| `ATN2` | ◻ |
| `PI` | ◻ |
| `RAND` | ◻ |
| `SIGN` | ◻ |
| `DEGREES` | ◻ |
| `RADIANS` | ◻ |

### Conversion

| Element | Status |
| --- | --- |
| `CAST` | ✅ |
| `CONVERT` | ✅ |
| `TRY_CAST` | ◻ |
| `TRY_CONVERT` | ◻ |
| `PARSE` | ◻ |
| `TRY_PARSE` | ◻ |
| `FORMAT` | ◻ |

### Logical

| Element | Status |
| --- | --- |
| `ISNULL` | ✅ |
| `COALESCE` | ✅ |
| `NULLIF` | ✅ |
| `CASE` | ✅ |
| `IIF` | ◻ |
| `CHOOSE` | ◻ |

### Metadata

| Element | Status |
| --- | --- |
| `DB_ID` | ✅ |
| `DB_NAME` | ✅ |
| `OBJECT_ID` | ✅ |
| `OBJECT_NAME` | ✅ |
| `OBJECT_SCHEMA_NAME` | ✅ |
| `SCHEMA_ID` | ✅ |
| `SCHEMA_NAME` | ✅ |
| `TYPE_NAME` | ✅ |
| `HAS_DBACCESS` | ✅ |
| `COL_NAME` | ◻ |
| `COL_LENGTH` | ◻ |
| `TYPE_ID` | ◻ |
| `OBJECT_DEFINITION` | ◻ |
| `OBJECTPROPERTY` | ◻ |
| `OBJECTPROPERTYEX` | ◻ |
| `COLUMNPROPERTY` | ◻ |
| `INDEXPROPERTY` | ◻ |
| `INDEXKEY_PROPERTY` | ◻ |
| `STATS_DATE` | ◻ |

### Configuration (`@@`)

| Element | Status |
| --- | --- |
| `@@VERSION` | ✅ |
| `@@SERVERNAME` | ✅ |
| `@@SPID` | ✅ |
| `@@LANGUAGE` | ✅ |
| `@@ROWCOUNT` | ✅ |
| `@@ERROR` | ✅ |
| `@@TRANCOUNT` | ✅ |
| `@@FETCH_STATUS` | ✅ |
| `@@IDENTITY` | ◻ |
| `@@SERVICENAME` | ◻ |
| `@@NESTLEVEL` | ◻ |
| `@@MAX_PRECISION` | ◻ |
| `@@OPTIONS` | ◻ |
| `@@DATEFIRST` | ◻ |
| `@@LOCK_TIMEOUT` | ◻ |
| `@@CURSOR_ROWS` | ◻ |
| `@@PROCID` | ◻ |

### Security

| Element | Status |
| --- | --- |
| `SYSTEM_USER` | ✅ |
| `CURRENT_USER` | ✅ |
| `SESSION_USER` | ✅ |
| `USER` | ✅ |
| `USER_NAME` | ✅ |
| `SUSER_NAME` | ✅ |
| `SUSER_SNAME` | ✅ |
| `HOST_NAME` | ✅ |
| `APP_NAME` | ✅ |
| `ORIGINAL_DB_NAME` | ✅ |
| `USER_ID` | ◻ |
| `SUSER_ID` | ◻ |
| `IS_MEMBER` | ◻ |
| `IS_SRVROLEMEMBER` | ◻ |
| `IS_ROLEMEMBER` | ◻ |
| `PERMISSIONS` | ◻ |
| `HAS_PERMS_BY_NAME` | ◻ |
| `ORIGINAL_LOGIN` | ◻ |
| `CONTEXT_INFO` | ◻ |
| `SESSION_CONTEXT` | ◻ |

### System / server

| Element | Status |
| --- | --- |
| `SERVERPROPERTY` | ✅ |
| `DATABASEPROPERTYEX` | ✅ |
| `CONNECTIONPROPERTY` | ◻ |
| `NEWID` | ◻ |
| `NEWSEQUENTIALID` | ◻ |
| `SCOPE_IDENTITY` | ◻ |
| `IDENT_CURRENT` | ◻ |
| `XACT_STATE` | ◻ |
| `ERROR_MESSAGE` | ◻ |
| `ERROR_NUMBER` | ◻ |
| `ERROR_SEVERITY` | ◻ |
| `ERROR_STATE` | ◻ |
| `ERROR_LINE` | ◻ |
| `ERROR_PROCEDURE` | ◻ |

### Ranking & window

| Element | Status |
| --- | --- |
| `ROW_NUMBER` | ◻ |
| `RANK` | ◻ |
| `DENSE_RANK` | ◻ |
| `NTILE` | ◻ |
| `LAG` | ◻ |
| `LEAD` | ◻ |
| `FIRST_VALUE` | ◻ |
| `LAST_VALUE` | ◻ |
| `PERCENT_RANK` | ◻ |
| `CUME_DIST` | ◻ |
| `PERCENTILE_CONT` | ◻ |
| `PERCENTILE_DISC` | ◻ |

### JSON

| Element | Status |
| --- | --- |
| `ISJSON` | ◻ |
| `JSON_VALUE` | ◻ |
| `JSON_QUERY` | ◻ |
| `JSON_MODIFY` | ◻ |
| `JSON_PATH_EXISTS` | ◻ |
| `JSON_OBJECT` | ◻ |
| `JSON_ARRAY` | ◻ |
| `OPENJSON` | ◻ |

### Cryptographic

| Element | Status |
| --- | --- |
| `HASHBYTES` | ◻ |
| `CHECKSUM` | ◻ |
| `BINARY_CHECKSUM` | ◻ |
| `COMPRESS` | ◻ |
| `DECOMPRESS` | ◻ |
| `PWDENCRYPT` | ◻ |
| `PWDCOMPARE` | ◻ |

### Cursor

| Element | Status |
| --- | --- |
| `CURSOR_STATUS` | ◻ |

---

## Catalog views (`sys.*`)

### Objects & modules

| Element | Status |
| --- | --- |
| `sys.objects` | ✅ |
| `sys.tables` | ✅ |
| `sys.views` | ✅ |
| `sys.procedures` | ✅ |
| `sys.sql_modules` | ✅ |
| `sys.parameters` | ✅ |
| `sys.triggers` | ◻ |
| `sys.all_objects` | ◻ |
| `sys.system_objects` | ◻ |
| `sys.sequences` | ◻ |
| `sys.synonyms` | ◻ |

### Columns & types

| Element | Status |
| --- | --- |
| `sys.columns` | ✅ |
| `sys.types` | ✅ |
| `sys.identity_columns` | ✅ |
| `sys.computed_columns` | ✅ |
| `sys.all_columns` | ◻ |
| `sys.table_types` | ◻ |

### Indexes & keys

| Element | Status |
| --- | --- |
| `sys.indexes` | ✅ |
| `sys.index_columns` | ✅ |
| `sys.key_constraints` | ✅ |
| `sys.foreign_keys` | ✅ |
| `sys.foreign_key_columns` | ✅ |
| `sys.stats` | ◻ |
| `sys.stats_columns` | ◻ |
| `sys.partitions` | ◻ |

### Constraints

| Element | Status |
| --- | --- |
| `sys.check_constraints` | ✅ |
| `sys.default_constraints` | ✅ |

### Databases & schemas

| Element | Status |
| --- | --- |
| `sys.databases` | ✅ |
| `sys.schemas` | ✅ |
| `sys.database_files` | ◻ |
| `sys.filegroups` | ◻ |
| `sys.extended_properties` | ◻ |
| `sys.sql_expression_dependencies` | ◻ |

### Security

| Element | Status |
| --- | --- |
| `sys.database_principals` | ◻ |
| `sys.server_principals` | ◻ |
| `sys.database_permissions` | ◻ |
| `sys.server_permissions` | ◻ |
| `sys.database_role_members` | ◻ |

### Dynamic management views

| Element | Status |
| --- | --- |
| `sys.dm_exec_sessions` | ◻ |
| `sys.dm_exec_connections` | ◻ |
| `sys.dm_exec_requests` | ◻ |
| `sys.dm_exec_query_stats` | ◻ |
| `sys.dm_os_waiting_tasks` | ◻ |

### Compatibility views

| Element | Status |
| --- | --- |
| `sys.sysobjects` | ◻ |
| `sys.syscolumns` | ◻ |
| `sys.systypes` | ◻ |
| `sys.sysindexes` | ◻ |
| `sys.sysusers` | ◻ |

---

## Information schema views

| Element | Status |
| --- | --- |
| `INFORMATION_SCHEMA.TABLES` | ✅ |
| `INFORMATION_SCHEMA.COLUMNS` | ✅ |
| `INFORMATION_SCHEMA.VIEWS` | ✅ |
| `INFORMATION_SCHEMA.ROUTINES` | ✅ |
| `INFORMATION_SCHEMA.PARAMETERS` | ✅ |
| `INFORMATION_SCHEMA.TABLE_CONSTRAINTS` | ✅ |
| `INFORMATION_SCHEMA.KEY_COLUMN_USAGE` | ✅ |
| `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS` | ✅ |
| `INFORMATION_SCHEMA.CHECK_CONSTRAINTS` | ◻ |
| `INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE` | ◻ |
| `INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE` | ◻ |
| `INFORMATION_SCHEMA.SCHEMATA` | ◻ |
| `INFORMATION_SCHEMA.DOMAINS` | ◻ |
| `INFORMATION_SCHEMA.ROUTINE_COLUMNS` | ◻ |
| `INFORMATION_SCHEMA.VIEW_TABLE_USAGE` | ◻ |
| `INFORMATION_SCHEMA.VIEW_COLUMN_USAGE` | ◻ |

---

## System stored procedures (`sp_*`)

### Catalog / ODBC driver metadata

| Element | Status |
| --- | --- |
| `sp_databases` | ✅ |
| `sp_tables` | ✅ |
| `sp_columns` | ✅ |
| `sp_pkeys` | ✅ |
| `sp_fkeys` | ✅ |
| `sp_statistics` | ✅ |
| `sp_special_columns` | ✅ |
| `sp_stored_procedures` | ✅ |
| `sp_sproc_columns` | ✅ |
| `sp_server_info` | ◻ |
| `sp_datatype_info` | ◻ |
| `sp_table_privileges` | ◻ |
| `sp_column_privileges` | ◻ |

### Help & scripting

| Element | Status |
| --- | --- |
| `sp_help` | ✅ |
| `sp_helptext` | ✅ |
| `sp_helpindex` | ✅ |
| `sp_helpconstraint` | ✅ |
| `sp_helpdb` | ◻ |
| `sp_helptrigger` | ◻ |
| `sp_depends` | ◻ |

### Administration

| Element | Status |
| --- | --- |
| `sp_executesql` | ◻ |
| `sp_rename` | ◻ |
| `sp_addextendedproperty` | ◻ |
| `sp_who` | ◻ |
| `sp_lock` | ◻ |
| `sp_configure` | ◻ |

---

## Language & statements

### Queries

| Element | Status |
| --- | --- |
| `SELECT` | ✅ |
| `WHERE` | ✅ |
| `INNER JOIN` | ✅ |
| `LEFT JOIN` | ✅ |
| `RIGHT JOIN` | ✅ |
| `FULL JOIN` | ✅ |
| `CROSS JOIN` | ✅ |
| `GROUP BY` | ✅ |
| `HAVING` | ✅ |
| `ORDER BY` | ✅ |
| `DISTINCT` | ✅ |
| `TOP` | ✅ |
| `OFFSET / FETCH` | ✅ |
| `UNION` | ✅ |
| `UNION ALL` | ✅ |
| `INTERSECT` | ✅ |
| `EXCEPT` | ✅ |
| `IN` | ✅ |
| `NOT IN` | ✅ |
| `EXISTS` | ✅ |
| `LIKE` | ✅ |
| `BETWEEN` | ✅ |
| `IS NULL` | ✅ |
| common table expression (`WITH`) | ✅ |
| subquery (scalar / `IN` / `EXISTS`) | ✅ |
| recursive CTE (at depth) | ◻ |
| `PIVOT` / `UNPIVOT` | ◻ |
| `CROSS APPLY` / `OUTER APPLY` | ◻ |
| `GROUPING SETS` / `ROLLUP` / `CUBE` | ◻ |
| `OVER` (window clause) | ◻ |
| `TABLESAMPLE` | ◻ |
| query hints (`OPTION`, `WITH (NOLOCK)`) | ◻ |

### DML

| Element | Status |
| --- | --- |
| `INSERT` | ✅ |
| `UPDATE` | ✅ |
| `DELETE` | ✅ |
| `MERGE` | ◻ |
| `SELECT ... INTO` | ◻ |
| `INSERT ... OUTPUT` | ◻ |
| `UPDATE ... FROM` | ◻ |
| `DELETE ... FROM` | ◻ |
| `TRUNCATE TABLE` | ◻ |
| `BULK INSERT` | ◻ |

### DDL

| Element | Status |
| --- | --- |
| `CREATE TABLE` | ✅ |
| `ALTER TABLE` (add / drop column) | ✅ |
| `DROP TABLE` | ✅ |
| `CREATE VIEW` | ✅ |
| `ALTER VIEW` | ✅ |
| `DROP VIEW` | ✅ |
| `CREATE PROCEDURE` | ✅ |
| `ALTER PROCEDURE` | ✅ |
| `DROP PROCEDURE` | ✅ |
| `CREATE / ALTER / DROP FUNCTION` | ◻ |
| `CREATE / ALTER / DROP TRIGGER` | ◻ |
| `CREATE / DROP INDEX` | ◻ |
| `CREATE / ALTER / DROP SEQUENCE` | ◻ |
| `CREATE / DROP SYNONYM` | ◻ |
| `CREATE / ALTER / DROP SCHEMA` | ◻ |
| `CREATE / DROP TYPE` | ◻ |
| `CREATE OR ALTER` | ◻ |
| table constraints (PK / FK / UNIQUE / CHECK / DEFAULT) in DDL | ◻ |
| `IDENTITY` / computed columns in DDL | ◻ |

### Control-of-flow

| Element | Status |
| --- | --- |
| `IF ... ELSE` | ◻ |
| `WHILE` | ◻ |
| `BEGIN ... END` | ◻ |
| `BREAK` | ◻ |
| `CONTINUE` | ◻ |
| `RETURN` | ◻ |
| `GOTO` | ◻ |
| `WAITFOR` | ◻ |
| `TRY ... CATCH` | ◻ |
| `THROW` | ◻ |
| `RAISERROR` | ◻ |
| `PRINT` | ◻ |

### Variables & batch

| Element | Status |
| --- | --- |
| `DECLARE @v = ...` | ✅ |
| `SET @v = ...` | ✅ |
| `USE` | ✅ |
| `EXEC` / `EXECUTE` (stored procedure) | ✅ |
| `SELECT @v = col` | ◻ |
| `DECLARE @t TABLE (...)` | ◻ |
| `sp_executesql` / dynamic SQL | ◻ |
| `SET` options with effect (`ANSI_NULLS`, `QUOTED_IDENTIFIER`, `ROWCOUNT`, `IDENTITY_INSERT`) | ◻ |

### Transactions

| Element | Status |
| --- | --- |
| `BEGIN TRANSACTION` | ◻ |
| `COMMIT` | ◻ |
| `ROLLBACK` | ◻ |
| `SAVE TRANSACTION` | ◻ |
| `SET XACT_ABORT` | ◻ |
| `SET TRANSACTION ISOLATION LEVEL` | ◻ |

### Cursors

| Element | Status |
| --- | --- |
| `DECLARE ... CURSOR` | ◻ |
| `OPEN` | ◻ |
| `FETCH` | ◻ |
| `CLOSE` | ◻ |
| `DEALLOCATE` | ◻ |

### Security

| Element | Status |
| --- | --- |
| `GRANT` | ◻ |
| `REVOKE` | ◻ |
| `DENY` | ◻ |
| `CREATE / ALTER / DROP USER` | ◻ |
| `CREATE / ALTER / DROP LOGIN` | ◻ |
| `CREATE / ALTER / DROP ROLE` | ◻ |
| `EXECUTE AS` / `REVERT` | ◻ |
