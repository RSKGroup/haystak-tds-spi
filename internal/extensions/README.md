# SQL Server surface -- command-level checklist

What it takes to "be SQL Server," one command per line, grouped by SQL Server's own categories.
`Y` = implemented (verified against the code), `-` = not yet. This is a checklist, not a design doc --
the design lives in [ARCHITECTURE.md](../../ARCHITECTURE.md) and [CONTRIBUTING.md](../../CONTRIBUTING.md).

---

## Functions

### Aggregate

| Element | File | Status |
| --- | --- | --- |
| `COUNT` | `internal/exec/aggregate.go` | Y |
| `SUM` | `internal/exec/aggregate.go` | Y |
| `MIN` | `internal/exec/aggregate.go` | Y |
| `MAX` | `internal/exec/aggregate.go` | Y |
| `AVG` | `internal/exec/aggregate.go` | Y |
| `COUNT_BIG` | `internal/exec/aggregate.go` | Y |
| `STDEV` | `internal/exec/aggregate.go` | Y |
| `STDEVP` | `internal/exec/aggregate.go` | Y |
| `VAR` | `internal/exec/aggregate.go` | Y |
| `VARP` | `internal/exec/aggregate.go` | Y |
| `GROUPING` |  | - |
| `GROUPING_ID` |  | - |
| `CHECKSUM_AGG` |  | - |
| `STRING_AGG` | `internal/exec/aggregate.go` | Y |
| `APPROX_COUNT_DISTINCT` |  | - |

### String

| Element | File | Status |
| --- | --- | --- |
| `LEN` | `internal/extensions/functions/string.go` | Y |
| `DATALEN` | `internal/extensions/functions/string.go` | Y |
| `UPPER` | `internal/extensions/functions/string.go` | Y |
| `LOWER` | `internal/extensions/functions/string.go` | Y |
| `LTRIM` | `internal/extensions/functions/string.go` | Y |
| `RTRIM` | `internal/extensions/functions/string.go` | Y |
| `TRIM` | `internal/extensions/functions/string.go` | Y |
| `SUBSTRING` | `internal/extensions/functions/string.go` | Y |
| `REPLACE` | `internal/extensions/functions/string.go` | Y |
| `CONCAT` | `internal/extensions/functions/string.go` | Y |
| `QUOTENAME` | `internal/extensions/functions/string.go` | Y |
| `CHARINDEX` | `internal/extensions/functions/string.go` | Y |
| `PATINDEX` | `internal/extensions/functions/string.go` | Y |
| `STUFF` | `internal/extensions/functions/string.go` | Y |
| `LEFT` | `internal/extensions/functions/string.go` | Y |
| `RIGHT` | `internal/extensions/functions/string.go` | Y |
| `REPLICATE` | `internal/extensions/functions/string.go` | Y |
| `SPACE` | `internal/extensions/functions/string.go` | Y |
| `REVERSE` | `internal/extensions/functions/string.go` | Y |
| `CONCAT_WS` | `internal/extensions/functions/string.go` | Y |
| `STRING_ESCAPE` | `internal/extensions/functions/string.go` | Y |
| `STRING_SPLIT` |  | - |
| `TRANSLATE` | `internal/extensions/functions/string.go` | Y |
| `FORMATMESSAGE` |  | - |
| `UNICODE` | `internal/extensions/functions/string.go` | Y |
| `NCHAR` | `internal/extensions/functions/string.go` | Y |
| `CHAR` | `internal/extensions/functions/string.go` | Y |
| `ASCII` | `internal/extensions/functions/string.go` | Y |
| `SOUNDEX` |  | - |
| `DIFFERENCE` |  | - |
| `STR` | `internal/extensions/functions/string.go` | Y |

### Date & time

| Element | File | Status |
| --- | --- | --- |
| `GETDATE` | `internal/extensions/functions/datetime.go` | Y |
| `GETUTCDATE` | `internal/extensions/functions/datetime.go` | Y |
| `SYSDATETIME` | `internal/extensions/functions/datetime.go` | Y |
| `SYSUTCDATETIME` | `internal/extensions/functions/datetime.go` | Y |
| `YEAR` | `internal/extensions/functions/datetime.go` | Y |
| `MONTH` | `internal/extensions/functions/datetime.go` | Y |
| `DAY` | `internal/extensions/functions/datetime.go` | Y |
| `SYSDATETIMEOFFSET` | `internal/extensions/functions/datetime.go` | Y |
| `CURRENT_TIMESTAMP` | `internal/extensions/functions/datetime.go` | Y |
| `DATEADD` | `internal/extensions/functions/datetime.go` | Y |
| `DATEDIFF` | `internal/extensions/functions/datetime.go` | Y |
| `DATEDIFF_BIG` | `internal/extensions/functions/datetime.go` | Y |
| `DATEPART` | `internal/extensions/functions/datetime.go` | Y |
| `DATENAME` | `internal/extensions/functions/datetime.go` | Y |
| `DATEFROMPARTS` | `internal/extensions/functions/datetime.go` | Y |
| `DATETIMEFROMPARTS` | `internal/extensions/functions/datetime.go` | Y |
| `EOMONTH` | `internal/extensions/functions/datetime.go` | Y |
| `SWITCHOFFSET` |  | - |
| `TODATETIMEOFFSET` |  | - |
| `ISDATE` | `internal/extensions/functions/datetime.go` | Y |
| `DATETRUNC` | `internal/extensions/functions/datetime.go` | Y |

### Mathematical

| Element | File | Status |
| --- | --- | --- |
| `ABS` | `internal/extensions/functions/math.go` | Y |
| `CEILING` | `internal/extensions/functions/math.go` | Y |
| `FLOOR` | `internal/extensions/functions/math.go` | Y |
| `ROUND` | `internal/extensions/functions/math.go` | Y |
| `POWER` | `internal/extensions/functions/math.go` | Y |
| `SQRT` | `internal/extensions/functions/math.go` | Y |
| `SQUARE` | `internal/extensions/functions/math.go` | Y |
| `EXP` | `internal/extensions/functions/math.go` | Y |
| `LOG` | `internal/extensions/functions/math.go` | Y |
| `LOG10` | `internal/extensions/functions/math.go` | Y |
| `SIN` | `internal/extensions/functions/math.go` | Y |
| `COS` | `internal/extensions/functions/math.go` | Y |
| `TAN` | `internal/extensions/functions/math.go` | Y |
| `COT` | `internal/extensions/functions/math.go` | Y |
| `ASIN` | `internal/extensions/functions/math.go` | Y |
| `ACOS` | `internal/extensions/functions/math.go` | Y |
| `ATAN` | `internal/extensions/functions/math.go` | Y |
| `ATN2` | `internal/extensions/functions/math.go` | Y |
| `PI` | `internal/extensions/functions/math.go` | Y |
| `RAND` | `internal/extensions/functions/math.go` | Y |
| `SIGN` | `internal/extensions/functions/math.go` | Y |
| `DEGREES` | `internal/extensions/functions/math.go` | Y |
| `RADIANS` | `internal/extensions/functions/math.go` | Y |

### Conversion

| Element | File | Status |
| --- | --- | --- |
| `CAST` | `internal/tsql/parser.go` | Y |
| `CONVERT` | `internal/tsql/parser.go` | Y |
| `TRY_CAST` | `internal/tsql/parser.go` | Y |
| `TRY_CONVERT` | `internal/tsql/parser.go` | Y |
| `PARSE` | `internal/tsql/parser.go` | Y |
| `TRY_PARSE` | `internal/tsql/parser.go` | Y |
| `FORMAT` |  | - |

### Logical

| Element | File | Status |
| --- | --- | --- |
| `ISNULL` | `internal/extensions/functions/logical.go` | Y |
| `COALESCE` | `internal/extensions/functions/logical.go` | Y |
| `NULLIF` | `internal/extensions/functions/logical.go` | Y |
| `CASE` | `internal/exec/value.go` | Y |
| `IIF` | `internal/tsql/parser.go` | Y |
| `CHOOSE` | `internal/extensions/functions/logical.go` | Y |

### Metadata

| Element | File | Status |
| --- | --- | --- |
| `DB_ID` | `internal/extensions/functions/metadata.go` | Y |
| `DB_NAME` | `internal/exec/value.go` | Y |
| `OBJECT_ID` | `internal/extensions/functions/metadata.go` | Y |
| `OBJECT_NAME` | `internal/exec/value.go` | Y |
| `OBJECT_SCHEMA_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `SCHEMA_ID` | `internal/extensions/functions/metadata.go` | Y |
| `SCHEMA_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `TYPE_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `HAS_DBACCESS` | `internal/extensions/functions/metadata.go` | Y |
| `COL_NAME` | `internal/exec/value.go` | Y |
| `COL_LENGTH` | `internal/exec/value.go` | Y |
| `TYPE_ID` | `internal/extensions/functions/metadata.go` | Y |
| `OBJECT_DEFINITION` | `internal/exec/value.go` | Y |
| `OBJECTPROPERTY` | `internal/exec/value.go` | Y |
| `OBJECTPROPERTYEX` | `internal/exec/value.go` | Y |
| `COLUMNPROPERTY` | `internal/exec/value.go` | Y |
| `INDEXPROPERTY` |  | - |
| `INDEXKEY_PROPERTY` |  | - |
| `STATS_DATE` | `internal/extensions/functions/metadata.go` | Y |

### Configuration (`@@`)

| Element | File | Status |
| --- | --- | --- |
| `@@VERSION` | `internal/extensions/functions/configuration.go` | Y |
| `@@SERVERNAME` | `internal/extensions/functions/configuration.go` | Y |
| `@@SPID` | `internal/extensions/functions/configuration.go` | Y |
| `@@LANGUAGE` | `internal/extensions/functions/configuration.go` | Y |
| `@@ROWCOUNT` | `internal/extensions/functions/configuration.go` | Y |
| `@@ERROR` | `internal/extensions/functions/configuration.go` | Y |
| `@@TRANCOUNT` | `internal/extensions/functions/configuration.go` | Y |
| `@@FETCH_STATUS` | `internal/extensions/functions/configuration.go` | Y |
| `@@IDENTITY` | `internal/extensions/functions/configuration.go` | Y |
| `@@SERVICENAME` | `internal/extensions/functions/configuration.go` | Y |
| `@@NESTLEVEL` | `internal/extensions/functions/configuration.go` | Y |
| `@@MAX_PRECISION` | `internal/extensions/functions/configuration.go` | Y |
| `@@OPTIONS` | `internal/extensions/functions/configuration.go` | Y |
| `@@DATEFIRST` | `internal/extensions/functions/configuration.go` | Y |
| `@@LOCK_TIMEOUT` | `internal/extensions/functions/configuration.go` | Y |
| `@@CURSOR_ROWS` | `internal/extensions/functions/configuration.go` | Y |
| `@@PROCID` | `internal/extensions/functions/configuration.go` | Y |

### Security

| Element | File | Status |
| --- | --- | --- |
| `SYSTEM_USER` | `internal/extensions/functions/security.go` | Y |
| `CURRENT_USER` | `internal/extensions/functions/security.go` | Y |
| `SESSION_USER` | `internal/extensions/functions/security.go` | Y |
| `USER` | `internal/extensions/functions/security.go` | Y |
| `USER_NAME` | `internal/extensions/functions/security.go` | Y |
| `SUSER_NAME` | `internal/extensions/functions/security.go` | Y |
| `SUSER_SNAME` | `internal/extensions/functions/security.go` | Y |
| `HOST_NAME` | `internal/extensions/functions/security.go` | Y |
| `APP_NAME` | `internal/extensions/functions/security.go` | Y |
| `ORIGINAL_DB_NAME` | `internal/engine/engine.go` | Y |
| `USER_ID` | `internal/extensions/functions/security.go` | Y |
| `SUSER_ID` | `internal/extensions/functions/security.go` | Y |
| `IS_MEMBER` | `internal/extensions/functions/security.go` | Y |
| `IS_SRVROLEMEMBER` | `internal/extensions/functions/security.go` | Y |
| `IS_ROLEMEMBER` | `internal/extensions/functions/security.go` | Y |
| `PERMISSIONS` |  | - |
| `HAS_PERMS_BY_NAME` |  | - |
| `ORIGINAL_LOGIN` | `internal/extensions/functions/security.go` | Y |
| `CONTEXT_INFO` |  | - |
| `SESSION_CONTEXT` |  | - |

### System / server

| Element | File | Status |
| --- | --- | --- |
| `SERVERPROPERTY` | `internal/extensions/functions/system.go` | Y |
| `DATABASEPROPERTYEX` | `internal/extensions/functions/system.go` | Y |
| `CONNECTIONPROPERTY` |  | - |
| `NEWID` | `internal/extensions/functions/system.go` | Y |
| `NEWSEQUENTIALID` |  | - |
| `SCOPE_IDENTITY` | `internal/extensions/functions/system.go` | Y |
| `IDENT_CURRENT` | `internal/extensions/functions/system.go` | Y |
| `XACT_STATE` |  | - |
| `ERROR_MESSAGE` | `internal/extensions/functions/system.go` | Y |
| `ERROR_NUMBER` | `internal/extensions/functions/system.go` | Y |
| `ERROR_SEVERITY` | `internal/extensions/functions/system.go` | Y |
| `ERROR_STATE` | `internal/extensions/functions/system.go` | Y |
| `ERROR_LINE` | `internal/extensions/functions/system.go` | Y |
| `ERROR_PROCEDURE` | `internal/extensions/functions/system.go` | Y |

### Ranking & window

| Element | File | Status |
| --- | --- | --- |
| `ROW_NUMBER` |  | - |
| `RANK` |  | - |
| `DENSE_RANK` |  | - |
| `NTILE` |  | - |
| `LAG` |  | - |
| `LEAD` |  | - |
| `FIRST_VALUE` |  | - |
| `LAST_VALUE` |  | - |
| `PERCENT_RANK` |  | - |
| `CUME_DIST` |  | - |
| `PERCENTILE_CONT` |  | - |
| `PERCENTILE_DISC` |  | - |

### JSON

| Element | File | Status |
| --- | --- | --- |
| `ISJSON` | `internal/extensions/functions/json.go` | Y |
| `JSON_VALUE` | `internal/extensions/functions/json.go` | Y |
| `JSON_QUERY` | `internal/extensions/functions/json.go` | Y |
| `JSON_MODIFY` |  | - |
| `JSON_PATH_EXISTS` |  | - |
| `JSON_OBJECT` |  | - |
| `JSON_ARRAY` |  | - |
| `OPENJSON` |  | - |

### Cryptographic

| Element | File | Status |
| --- | --- | --- |
| `HASHBYTES` | `internal/extensions/functions/crypto.go` | Y |
| `CHECKSUM` | `internal/extensions/functions/crypto.go` | Y |
| `BINARY_CHECKSUM` | `internal/extensions/functions/crypto.go` | Y |
| `COMPRESS` |  | - |
| `DECOMPRESS` |  | - |
| `PWDENCRYPT` |  | - |
| `PWDCOMPARE` |  | - |

### Cursor

| Element | File | Status |
| --- | --- | --- |
| `CURSOR_STATUS` |  | - |

---

## Catalog views (`sys.*`)

### Objects & modules

| Element | File | Status |
| --- | --- | --- |
| `sys.objects` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.tables` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.views` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.procedures` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.sql_modules` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.parameters` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.triggers` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.all_objects` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.system_objects` |  | - |
| `sys.sequences` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.synonyms` | `internal/extensions/sysviews/sysviews.go` | Y |

### Columns & types

| Element | File | Status |
| --- | --- | --- |
| `sys.columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.types` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.identity_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.computed_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.all_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.table_types` | `internal/extensions/sysviews/sysviews.go` | Y |

### Indexes & keys

| Element | File | Status |
| --- | --- | --- |
| `sys.indexes` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.index_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.key_constraints` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.foreign_keys` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.foreign_key_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.stats` |  | - |
| `sys.stats_columns` |  | - |
| `sys.partitions` |  | - |

### Constraints

| Element | File | Status |
| --- | --- | --- |
| `sys.check_constraints` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.default_constraints` | `internal/extensions/sysviews/sysviews.go` | Y |

### Databases & schemas

| Element | File | Status |
| --- | --- | --- |
| `sys.databases` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.schemas` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.database_files` |  | - |
| `sys.filegroups` |  | - |
| `sys.extended_properties` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.sql_expression_dependencies` | `internal/extensions/sysviews/sysviews.go` | Y |

### Security

| Element | File | Status |
| --- | --- | --- |
| `sys.database_principals` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.server_principals` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.database_permissions` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.server_permissions` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.database_role_members` | `internal/extensions/sysviews/sysviews.go` | Y |

### Dynamic management views

| Element | File | Status |
| --- | --- | --- |
| `sys.dm_exec_sessions` |  | - |
| `sys.dm_exec_connections` |  | - |
| `sys.dm_exec_requests` |  | - |
| `sys.dm_exec_query_stats` |  | - |
| `sys.dm_os_waiting_tasks` |  | - |

### Compatibility views

| Element | File | Status |
| --- | --- | --- |
| `sys.sysobjects` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.syscolumns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.systypes` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.sysindexes` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.sysusers` | `internal/extensions/sysviews/sysviews.go` | Y |

---

## Information schema views

| Element | File | Status |
| --- | --- | --- |
| `INFORMATION_SCHEMA.TABLES` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.COLUMNS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.VIEWS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.ROUTINES` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.PARAMETERS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.TABLE_CONSTRAINTS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.KEY_COLUMN_USAGE` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.CHECK_CONSTRAINTS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.SCHEMATA` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.DOMAINS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.ROUTINE_COLUMNS` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.VIEW_TABLE_USAGE` | `internal/extensions/infoschema/infoschema.go` | Y |
| `INFORMATION_SCHEMA.VIEW_COLUMN_USAGE` |  | - |

---

## System stored procedures (`sp_*`)

### Catalog / ODBC driver metadata

| Element | File | Status |
| --- | --- | --- |
| `sp_databases` | `internal/engine/proc.go` | Y |
| `sp_tables` | `internal/engine/proc.go` | Y |
| `sp_columns` | `internal/engine/proc.go` | Y |
| `sp_pkeys` | `internal/engine/proc.go` | Y |
| `sp_fkeys` | `internal/engine/proc.go` | Y |
| `sp_statistics` | `internal/engine/proc.go` | Y |
| `sp_special_columns` | `internal/engine/proc.go` | Y |
| `sp_stored_procedures` | `internal/engine/proc.go` | Y |
| `sp_sproc_columns` | `internal/engine/proc.go` | Y |
| `sp_server_info` | `internal/engine/proc.go` | Y |
| `sp_datatype_info` | `internal/engine/proc.go` | Y |
| `sp_table_privileges` |  | - |
| `sp_column_privileges` |  | - |

### Help & scripting

| Element | File | Status |
| --- | --- | --- |
| `sp_help` | `internal/engine/proc.go` | Y |
| `sp_helptext` | `internal/engine/proc.go` | Y |
| `sp_helpindex` | `internal/engine/proc.go` | Y |
| `sp_helpconstraint` | `internal/engine/proc.go` | Y |
| `sp_helpdb` | `internal/engine/proc.go` | Y |
| `sp_helptrigger` |  | - |
| `sp_depends` |  | - |

### Administration

| Element | File | Status |
| --- | --- | --- |
| `sp_executesql` |  | - |
| `sp_rename` |  | - |
| `sp_addextendedproperty` |  | - |
| `sp_who` |  | - |
| `sp_lock` | `internal/engine/proc.go` | Y |
| `sp_configure` | `internal/engine/proc.go` | Y |

---

## Language & statements

### Queries

| Element | File | Status |
| --- | --- | --- |
| `SELECT` | `internal/tsql/parser.go + internal/exec` | Y |
| `WHERE` | `internal/tsql/parser.go + internal/exec` | Y |
| `INNER JOIN` | `internal/tsql/parser.go + internal/exec` | Y |
| `LEFT JOIN` | `internal/tsql/parser.go + internal/exec` | Y |
| `RIGHT JOIN` | `internal/tsql/parser.go + internal/exec` | Y |
| `FULL JOIN` | `internal/tsql/parser.go + internal/exec` | Y |
| `CROSS JOIN` | `internal/tsql/parser.go + internal/exec` | Y |
| `GROUP BY` | `internal/tsql/parser.go + internal/exec` | Y |
| `HAVING` | `internal/tsql/parser.go + internal/exec` | Y |
| `ORDER BY` | `internal/tsql/parser.go + internal/exec` | Y |
| `DISTINCT` | `internal/tsql/parser.go + internal/exec` | Y |
| `TOP` | `internal/tsql/parser.go + internal/exec` | Y |
| `OFFSET / FETCH` | `internal/tsql/parser.go + internal/exec` | Y |
| `UNION` | `internal/tsql/parser.go + internal/exec` | Y |
| `UNION ALL` | `internal/tsql/parser.go + internal/exec` | Y |
| `INTERSECT` | `internal/tsql/parser.go + internal/exec` | Y |
| `EXCEPT` | `internal/tsql/parser.go + internal/exec` | Y |
| `IN` | `internal/tsql/parser.go + internal/exec` | Y |
| `NOT IN` | `internal/tsql/parser.go + internal/exec` | Y |
| `EXISTS` | `internal/tsql/parser.go + internal/exec` | Y |
| `LIKE` | `internal/tsql/parser.go + internal/exec` | Y |
| `BETWEEN` | `internal/tsql/parser.go + internal/exec` | Y |
| `IS NULL` | `internal/tsql/parser.go + internal/exec` | Y |
| common table expression (`WITH`) | `internal/tsql/parser.go + internal/exec` | Y |
| subquery (scalar / `IN` / `EXISTS`) | `internal/tsql/parser.go + internal/exec` | Y |
| recursive CTE (at depth) |  | - |
| `PIVOT` / `UNPIVOT` |  | - |
| `CROSS APPLY` / `OUTER APPLY` |  | - |
| `GROUPING SETS` / `ROLLUP` / `CUBE` |  | - |
| `OVER` (window clause) |  | - |
| `TABLESAMPLE` |  | - |
| query hints (`OPTION`, `WITH (NOLOCK)`) |  | - |

### DML

| Element | File | Status |
| --- | --- | --- |
| `INSERT` | `internal/engine/engine.go` | Y |
| `UPDATE` | `internal/engine/engine.go` | Y |
| `DELETE` | `internal/engine/engine.go` | Y |
| `MERGE` |  | - |
| `SELECT ... INTO` |  | - |
| `INSERT ... OUTPUT` |  | - |
| `UPDATE ... FROM` |  | - |
| `DELETE ... FROM` |  | - |
| `TRUNCATE TABLE` |  | - |
| `BULK INSERT` |  | - |

### DDL

| Element | File | Status |
| --- | --- | --- |
| `CREATE TABLE` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `ALTER TABLE` (add / drop column) | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `DROP TABLE` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `CREATE VIEW` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `ALTER VIEW` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `DROP VIEW` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `CREATE PROCEDURE` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `ALTER PROCEDURE` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `DROP PROCEDURE` | `internal/tsql + internal/extensions/{views,procedures}` | Y |
| `CREATE / ALTER / DROP FUNCTION` |  | - |
| `CREATE / ALTER / DROP TRIGGER` |  | - |
| `CREATE / DROP INDEX` |  | - |
| `CREATE / ALTER / DROP SEQUENCE` |  | - |
| `CREATE / DROP SYNONYM` |  | - |
| `CREATE / ALTER / DROP SCHEMA` |  | - |
| `CREATE / DROP TYPE` |  | - |
| `CREATE OR ALTER` |  | - |
| table constraints (PK / FK / UNIQUE / CHECK / DEFAULT) in DDL |  | - |
| `IDENTITY` / computed columns in DDL |  | - |

### Control-of-flow

| Element | File | Status |
| --- | --- | --- |
| `IF ... ELSE` |  | - |
| `WHILE` |  | - |
| `BEGIN ... END` |  | - |
| `BREAK` |  | - |
| `CONTINUE` |  | - |
| `RETURN` |  | - |
| `GOTO` |  | - |
| `WAITFOR` |  | - |
| `TRY ... CATCH` |  | - |
| `THROW` |  | - |
| `RAISERROR` |  | - |
| `PRINT` |  | - |

### Variables & batch

| Element | File | Status |
| --- | --- | --- |
| `DECLARE @v = ...` | `internal/extensions/batch/batch.go` | Y |
| `SET @v = ...` | `internal/extensions/batch/batch.go` | Y |
| `USE` | `internal/extensions/batch/batch.go` | Y |
| `EXEC` / `EXECUTE` (stored procedure) | `internal/extensions/batch/batch.go` | Y |
| `SELECT @v = col` |  | - |
| `DECLARE @t TABLE (...)` |  | - |
| `sp_executesql` / dynamic SQL |  | - |
| `SET` options with effect (`ANSI_NULLS`, `QUOTED_IDENTIFIER`, `ROWCOUNT`, `IDENTITY_INSERT`) |  | - |

### Transactions

| Element | File | Status |
| --- | --- | --- |
| `BEGIN TRANSACTION` |  | - |
| `COMMIT` |  | - |
| `ROLLBACK` |  | - |
| `SAVE TRANSACTION` |  | - |
| `SET XACT_ABORT` |  | - |
| `SET TRANSACTION ISOLATION LEVEL` |  | - |

### Cursors

| Element | File | Status |
| --- | --- | --- |
| `DECLARE ... CURSOR` |  | - |
| `OPEN` |  | - |
| `FETCH` |  | - |
| `CLOSE` |  | - |
| `DEALLOCATE` |  | - |

### Security

| Element | File | Status |
| --- | --- | --- |
| `GRANT` |  | - |
| `REVOKE` |  | - |
| `DENY` |  | - |
| `CREATE / ALTER / DROP USER` |  | - |
| `CREATE / ALTER / DROP LOGIN` |  | - |
| `CREATE / ALTER / DROP ROLE` |  | - |
| `EXECUTE AS` / `REVERT` |  | - |
