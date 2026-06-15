# SQL Server surface -- command-level checklist

What it takes to "be SQL Server," one command per line, grouped by SQL Server's own categories.
`[x]` = implemented (verified against the code), `[ ]` = not yet. This is a checklist, not a design doc --
the design lives in [ARCHITECTURE.md](../../ARCHITECTURE.md) and [CONTRIBUTING.md](../../CONTRIBUTING.md).

---

## Functions

### Aggregate

| Element | Status |
| --- | --- |
| `COUNT` | x |
| `SUM` | x |
| `MIN` | x |
| `MAX` | x |
| `AVG` | x |
| `COUNT_BIG` | - |
| `STDEV` | - |
| `STDEVP` | - |
| `VAR` | - |
| `VARP` | - |
| `GROUPING` | - |
| `GROUPING_ID` | - |
| `CHECKSUM_AGG` | - |
| `STRING_AGG` | - |
| `APPROX_COUNT_DISTINCT` | - |

### String

| Element | Status |
| --- | --- |
| `LEN` | x |
| `DATALEN` | x |
| `UPPER` | x |
| `LOWER` | x |
| `LTRIM` | x |
| `RTRIM` | x |
| `TRIM` | x |
| `SUBSTRING` | x |
| `REPLACE` | x |
| `CONCAT` | x |
| `QUOTENAME` | x |
| `CHARINDEX` | - |
| `PATINDEX` | - |
| `STUFF` | - |
| `LEFT` | - |
| `RIGHT` | - |
| `REPLICATE` | - |
| `SPACE` | - |
| `REVERSE` | - |
| `CONCAT_WS` | - |
| `STRING_ESCAPE` | - |
| `STRING_SPLIT` | - |
| `TRANSLATE` | - |
| `FORMATMESSAGE` | - |
| `UNICODE` | - |
| `NCHAR` | - |
| `CHAR` | - |
| `ASCII` | - |
| `SOUNDEX` | - |
| `DIFFERENCE` | - |
| `STR` | - |

### Date & time

| Element | Status |
| --- | --- |
| `GETDATE` | x |
| `GETUTCDATE` | x |
| `SYSDATETIME` | x |
| `SYSUTCDATETIME` | x |
| `YEAR` | x |
| `MONTH` | x |
| `DAY` | x |
| `SYSDATETIMEOFFSET` | - |
| `CURRENT_TIMESTAMP` | - |
| `DATEADD` | - |
| `DATEDIFF` | - |
| `DATEDIFF_BIG` | - |
| `DATEPART` | - |
| `DATENAME` | - |
| `DATEFROMPARTS` | - |
| `DATETIMEFROMPARTS` | - |
| `EOMONTH` | - |
| `SWITCHOFFSET` | - |
| `TODATETIMEOFFSET` | - |
| `ISDATE` | - |
| `DATETRUNC` | - |

### Mathematical

| Element | Status |
| --- | --- |
| `ABS` | x |
| `CEILING` | - |
| `FLOOR` | - |
| `ROUND` | - |
| `POWER` | - |
| `SQRT` | - |
| `SQUARE` | - |
| `EXP` | - |
| `LOG` | - |
| `LOG10` | - |
| `SIN` | - |
| `COS` | - |
| `TAN` | - |
| `COT` | - |
| `ASIN` | - |
| `ACOS` | - |
| `ATAN` | - |
| `ATN2` | - |
| `PI` | - |
| `RAND` | - |
| `SIGN` | - |
| `DEGREES` | - |
| `RADIANS` | - |

### Conversion

| Element | Status |
| --- | --- |
| `CAST` | x |
| `CONVERT` | x |
| `TRY_CAST` | - |
| `TRY_CONVERT` | - |
| `PARSE` | - |
| `TRY_PARSE` | - |
| `FORMAT` | - |

### Logical

| Element | Status |
| --- | --- |
| `ISNULL` | x |
| `COALESCE` | x |
| `NULLIF` | x |
| `CASE` | x |
| `IIF` | - |
| `CHOOSE` | - |

### Metadata

| Element | Status |
| --- | --- |
| `DB_ID` | x |
| `DB_NAME` | x |
| `OBJECT_ID` | x |
| `OBJECT_NAME` | x |
| `OBJECT_SCHEMA_NAME` | x |
| `SCHEMA_ID` | x |
| `SCHEMA_NAME` | x |
| `TYPE_NAME` | x |
| `HAS_DBACCESS` | x |
| `COL_NAME` | - |
| `COL_LENGTH` | - |
| `TYPE_ID` | - |
| `OBJECT_DEFINITION` | - |
| `OBJECTPROPERTY` | - |
| `OBJECTPROPERTYEX` | - |
| `COLUMNPROPERTY` | - |
| `INDEXPROPERTY` | - |
| `INDEXKEY_PROPERTY` | - |
| `STATS_DATE` | - |

### Configuration (`@@`)

| Element | Status |
| --- | --- |
| `@@VERSION` | x |
| `@@SERVERNAME` | x |
| `@@SPID` | x |
| `@@LANGUAGE` | x |
| `@@ROWCOUNT` | x |
| `@@ERROR` | x |
| `@@TRANCOUNT` | x |
| `@@FETCH_STATUS` | x |
| `@@IDENTITY` | - |
| `@@SERVICENAME` | - |
| `@@NESTLEVEL` | - |
| `@@MAX_PRECISION` | - |
| `@@OPTIONS` | - |
| `@@DATEFIRST` | - |
| `@@LOCK_TIMEOUT` | - |
| `@@CURSOR_ROWS` | - |
| `@@PROCID` | - |

### Security

| Element | Status |
| --- | --- |
| `SYSTEM_USER` | x |
| `CURRENT_USER` | x |
| `SESSION_USER` | x |
| `USER` | x |
| `USER_NAME` | x |
| `SUSER_NAME` | x |
| `SUSER_SNAME` | x |
| `HOST_NAME` | x |
| `APP_NAME` | x |
| `ORIGINAL_DB_NAME` | x |
| `USER_ID` | - |
| `SUSER_ID` | - |
| `IS_MEMBER` | - |
| `IS_SRVROLEMEMBER` | - |
| `IS_ROLEMEMBER` | - |
| `PERMISSIONS` | - |
| `HAS_PERMS_BY_NAME` | - |
| `ORIGINAL_LOGIN` | - |
| `CONTEXT_INFO` | - |
| `SESSION_CONTEXT` | - |

### System / server

| Element | Status |
| --- | --- |
| `SERVERPROPERTY` | x |
| `DATABASEPROPERTYEX` | x |
| `CONNECTIONPROPERTY` | - |
| `NEWID` | - |
| `NEWSEQUENTIALID` | - |
| `SCOPE_IDENTITY` | - |
| `IDENT_CURRENT` | - |
| `XACT_STATE` | - |
| `ERROR_MESSAGE` | - |
| `ERROR_NUMBER` | - |
| `ERROR_SEVERITY` | - |
| `ERROR_STATE` | - |
| `ERROR_LINE` | - |
| `ERROR_PROCEDURE` | - |

### Ranking & window

| Element | Status |
| --- | --- |
| `ROW_NUMBER` | - |
| `RANK` | - |
| `DENSE_RANK` | - |
| `NTILE` | - |
| `LAG` | - |
| `LEAD` | - |
| `FIRST_VALUE` | - |
| `LAST_VALUE` | - |
| `PERCENT_RANK` | - |
| `CUME_DIST` | - |
| `PERCENTILE_CONT` | - |
| `PERCENTILE_DISC` | - |

### JSON

| Element | Status |
| --- | --- |
| `ISJSON` | - |
| `JSON_VALUE` | - |
| `JSON_QUERY` | - |
| `JSON_MODIFY` | - |
| `JSON_PATH_EXISTS` | - |
| `JSON_OBJECT` | - |
| `JSON_ARRAY` | - |
| `OPENJSON` | - |

### Cryptographic

| Element | Status |
| --- | --- |
| `HASHBYTES` | - |
| `CHECKSUM` | - |
| `BINARY_CHECKSUM` | - |
| `COMPRESS` | - |
| `DECOMPRESS` | - |
| `PWDENCRYPT` | - |
| `PWDCOMPARE` | - |

### Cursor

| Element | Status |
| --- | --- |
| `CURSOR_STATUS` | - |

---

## Catalog views (`sys.*`)

### Objects & modules

| Element | Status |
| --- | --- |
| `sys.objects` | x |
| `sys.tables` | x |
| `sys.views` | x |
| `sys.procedures` | x |
| `sys.sql_modules` | x |
| `sys.parameters` | x |
| `sys.triggers` | - |
| `sys.all_objects` | - |
| `sys.system_objects` | - |
| `sys.sequences` | - |
| `sys.synonyms` | - |

### Columns & types

| Element | Status |
| --- | --- |
| `sys.columns` | x |
| `sys.types` | x |
| `sys.identity_columns` | x |
| `sys.computed_columns` | x |
| `sys.all_columns` | - |
| `sys.table_types` | - |

### Indexes & keys

| Element | Status |
| --- | --- |
| `sys.indexes` | x |
| `sys.index_columns` | x |
| `sys.key_constraints` | x |
| `sys.foreign_keys` | x |
| `sys.foreign_key_columns` | x |
| `sys.stats` | - |
| `sys.stats_columns` | - |
| `sys.partitions` | - |

### Constraints

| Element | Status |
| --- | --- |
| `sys.check_constraints` | x |
| `sys.default_constraints` | x |

### Databases & schemas

| Element | Status |
| --- | --- |
| `sys.databases` | x |
| `sys.schemas` | x |
| `sys.database_files` | - |
| `sys.filegroups` | - |
| `sys.extended_properties` | - |
| `sys.sql_expression_dependencies` | - |

### Security

| Element | Status |
| --- | --- |
| `sys.database_principals` | - |
| `sys.server_principals` | - |
| `sys.database_permissions` | - |
| `sys.server_permissions` | - |
| `sys.database_role_members` | - |

### Dynamic management views

| Element | Status |
| --- | --- |
| `sys.dm_exec_sessions` | - |
| `sys.dm_exec_connections` | - |
| `sys.dm_exec_requests` | - |
| `sys.dm_exec_query_stats` | - |
| `sys.dm_os_waiting_tasks` | - |

### Compatibility views

| Element | Status |
| --- | --- |
| `sys.sysobjects` | - |
| `sys.syscolumns` | - |
| `sys.systypes` | - |
| `sys.sysindexes` | - |
| `sys.sysusers` | - |

---

## Information schema views

| Element | Status |
| --- | --- |
| `INFORMATION_SCHEMA.TABLES` | x |
| `INFORMATION_SCHEMA.COLUMNS` | x |
| `INFORMATION_SCHEMA.VIEWS` | x |
| `INFORMATION_SCHEMA.ROUTINES` | x |
| `INFORMATION_SCHEMA.PARAMETERS` | x |
| `INFORMATION_SCHEMA.TABLE_CONSTRAINTS` | x |
| `INFORMATION_SCHEMA.KEY_COLUMN_USAGE` | x |
| `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS` | x |
| `INFORMATION_SCHEMA.CHECK_CONSTRAINTS` | - |
| `INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE` | - |
| `INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE` | - |
| `INFORMATION_SCHEMA.SCHEMATA` | - |
| `INFORMATION_SCHEMA.DOMAINS` | - |
| `INFORMATION_SCHEMA.ROUTINE_COLUMNS` | - |
| `INFORMATION_SCHEMA.VIEW_TABLE_USAGE` | - |
| `INFORMATION_SCHEMA.VIEW_COLUMN_USAGE` | - |

---

## System stored procedures (`sp_*`)

### Catalog / ODBC driver metadata

| Element | Status |
| --- | --- |
| `sp_databases` | x |
| `sp_tables` | x |
| `sp_columns` | x |
| `sp_pkeys` | x |
| `sp_fkeys` | x |
| `sp_statistics` | x |
| `sp_special_columns` | x |
| `sp_stored_procedures` | x |
| `sp_sproc_columns` | x |
| `sp_server_info` | - |
| `sp_datatype_info` | - |
| `sp_table_privileges` | - |
| `sp_column_privileges` | - |

### Help & scripting

| Element | Status |
| --- | --- |
| `sp_help` | x |
| `sp_helptext` | x |
| `sp_helpindex` | x |
| `sp_helpconstraint` | x |
| `sp_helpdb` | - |
| `sp_helptrigger` | - |
| `sp_depends` | - |

### Administration

| Element | Status |
| --- | --- |
| `sp_executesql` | - |
| `sp_rename` | - |
| `sp_addextendedproperty` | - |
| `sp_who` | - |
| `sp_lock` | - |
| `sp_configure` | - |

---

## Language & statements

### Queries

| Element | Status |
| --- | --- |
| `SELECT` | x |
| `WHERE` | x |
| `INNER JOIN` | x |
| `LEFT JOIN` | x |
| `RIGHT JOIN` | x |
| `FULL JOIN` | x |
| `CROSS JOIN` | x |
| `GROUP BY` | x |
| `HAVING` | x |
| `ORDER BY` | x |
| `DISTINCT` | x |
| `TOP` | x |
| `OFFSET / FETCH` | x |
| `UNION` | x |
| `UNION ALL` | x |
| `INTERSECT` | x |
| `EXCEPT` | x |
| `IN` | x |
| `NOT IN` | x |
| `EXISTS` | x |
| `LIKE` | x |
| `BETWEEN` | x |
| `IS NULL` | x |
| common table expression (`WITH`) | x |
| subquery (scalar / `IN` / `EXISTS`) | x |
| recursive CTE (at depth) | - |
| `PIVOT` / `UNPIVOT` | - |
| `CROSS APPLY` / `OUTER APPLY` | - |
| `GROUPING SETS` / `ROLLUP` / `CUBE` | - |
| `OVER` (window clause) | - |
| `TABLESAMPLE` | - |
| query hints (`OPTION`, `WITH (NOLOCK)`) | - |

### DML

| Element | Status |
| --- | --- |
| `INSERT` | x |
| `UPDATE` | x |
| `DELETE` | x |
| `MERGE` | - |
| `SELECT ... INTO` | - |
| `INSERT ... OUTPUT` | - |
| `UPDATE ... FROM` | - |
| `DELETE ... FROM` | - |
| `TRUNCATE TABLE` | - |
| `BULK INSERT` | - |

### DDL

| Element | Status |
| --- | --- |
| `CREATE TABLE` | x |
| `ALTER TABLE` (add / drop column) | x |
| `DROP TABLE` | x |
| `CREATE VIEW` | x |
| `ALTER VIEW` | x |
| `DROP VIEW` | x |
| `CREATE PROCEDURE` | x |
| `ALTER PROCEDURE` | x |
| `DROP PROCEDURE` | x |
| `CREATE / ALTER / DROP FUNCTION` | - |
| `CREATE / ALTER / DROP TRIGGER` | - |
| `CREATE / DROP INDEX` | - |
| `CREATE / ALTER / DROP SEQUENCE` | - |
| `CREATE / DROP SYNONYM` | - |
| `CREATE / ALTER / DROP SCHEMA` | - |
| `CREATE / DROP TYPE` | - |
| `CREATE OR ALTER` | - |
| table constraints (PK / FK / UNIQUE / CHECK / DEFAULT) in DDL | - |
| `IDENTITY` / computed columns in DDL | - |

### Control-of-flow

| Element | Status |
| --- | --- |
| `IF ... ELSE` | - |
| `WHILE` | - |
| `BEGIN ... END` | - |
| `BREAK` | - |
| `CONTINUE` | - |
| `RETURN` | - |
| `GOTO` | - |
| `WAITFOR` | - |
| `TRY ... CATCH` | - |
| `THROW` | - |
| `RAISERROR` | - |
| `PRINT` | - |

### Variables & batch

| Element | Status |
| --- | --- |
| `DECLARE @v = ...` | x |
| `SET @v = ...` | x |
| `USE` | x |
| `EXEC` / `EXECUTE` (stored procedure) | x |
| `SELECT @v = col` | - |
| `DECLARE @t TABLE (...)` | - |
| `sp_executesql` / dynamic SQL | - |
| `SET` options with effect (`ANSI_NULLS`, `QUOTED_IDENTIFIER`, `ROWCOUNT`, `IDENTITY_INSERT`) | - |

### Transactions

| Element | Status |
| --- | --- |
| `BEGIN TRANSACTION` | - |
| `COMMIT` | - |
| `ROLLBACK` | - |
| `SAVE TRANSACTION` | - |
| `SET XACT_ABORT` | - |
| `SET TRANSACTION ISOLATION LEVEL` | - |

### Cursors

| Element | Status |
| --- | --- |
| `DECLARE ... CURSOR` | - |
| `OPEN` | - |
| `FETCH` | - |
| `CLOSE` | - |
| `DEALLOCATE` | - |

### Security

| Element | Status |
| --- | --- |
| `GRANT` | - |
| `REVOKE` | - |
| `DENY` | - |
| `CREATE / ALTER / DROP USER` | - |
| `CREATE / ALTER / DROP LOGIN` | - |
| `CREATE / ALTER / DROP ROLE` | - |
| `EXECUTE AS` / `REVERT` | - |
