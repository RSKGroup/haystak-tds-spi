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
| `COUNT_BIG` | `internal/exec/aggregate.go` | - |
| `STDEV` | `internal/exec/aggregate.go` | - |
| `STDEVP` | `internal/exec/aggregate.go` | - |
| `VAR` | `internal/exec/aggregate.go` | - |
| `VARP` | `internal/exec/aggregate.go` | - |
| `GROUPING` | `internal/exec/aggregate.go` | - |
| `GROUPING_ID` | `internal/exec/aggregate.go` | - |
| `CHECKSUM_AGG` | `internal/exec/aggregate.go` | - |
| `STRING_AGG` | `internal/exec/aggregate.go` | - |
| `APPROX_COUNT_DISTINCT` | `internal/exec/aggregate.go` | - |

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
| `CHARINDEX` | `internal/extensions/functions/string.go` | - |
| `PATINDEX` | `internal/extensions/functions/string.go` | - |
| `STUFF` | `internal/extensions/functions/string.go` | - |
| `LEFT` | `internal/extensions/functions/string.go` | - |
| `RIGHT` | `internal/extensions/functions/string.go` | - |
| `REPLICATE` | `internal/extensions/functions/string.go` | - |
| `SPACE` | `internal/extensions/functions/string.go` | - |
| `REVERSE` | `internal/extensions/functions/string.go` | - |
| `CONCAT_WS` | `internal/extensions/functions/string.go` | - |
| `STRING_ESCAPE` | `internal/extensions/functions/string.go` | - |
| `STRING_SPLIT` | `internal/extensions/functions/string.go` | - |
| `TRANSLATE` | `internal/extensions/functions/string.go` | - |
| `FORMATMESSAGE` | `internal/extensions/functions/string.go` | - |
| `UNICODE` | `internal/extensions/functions/string.go` | - |
| `NCHAR` | `internal/extensions/functions/string.go` | - |
| `CHAR` | `internal/extensions/functions/string.go` | - |
| `ASCII` | `internal/extensions/functions/string.go` | - |
| `SOUNDEX` | `internal/extensions/functions/string.go` | - |
| `DIFFERENCE` | `internal/extensions/functions/string.go` | - |
| `STR` | `internal/extensions/functions/string.go` | - |

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
| `SYSDATETIMEOFFSET` | `internal/extensions/functions/datetime.go` | - |
| `CURRENT_TIMESTAMP` | `internal/extensions/functions/datetime.go` | - |
| `DATEADD` | `internal/extensions/functions/datetime.go` | - |
| `DATEDIFF` | `internal/extensions/functions/datetime.go` | - |
| `DATEDIFF_BIG` | `internal/extensions/functions/datetime.go` | - |
| `DATEPART` | `internal/extensions/functions/datetime.go` | - |
| `DATENAME` | `internal/extensions/functions/datetime.go` | - |
| `DATEFROMPARTS` | `internal/extensions/functions/datetime.go` | - |
| `DATETIMEFROMPARTS` | `internal/extensions/functions/datetime.go` | - |
| `EOMONTH` | `internal/extensions/functions/datetime.go` | - |
| `SWITCHOFFSET` | `internal/extensions/functions/datetime.go` | - |
| `TODATETIMEOFFSET` | `internal/extensions/functions/datetime.go` | - |
| `ISDATE` | `internal/extensions/functions/datetime.go` | - |
| `DATETRUNC` | `internal/extensions/functions/datetime.go` | - |

### Mathematical

| Element | File | Status |
| --- | --- | --- |
| `ABS` | `internal/extensions/functions/math.go` | Y |
| `CEILING` | `internal/extensions/functions/math.go` | - |
| `FLOOR` | `internal/extensions/functions/math.go` | - |
| `ROUND` | `internal/extensions/functions/math.go` | - |
| `POWER` | `internal/extensions/functions/math.go` | - |
| `SQRT` | `internal/extensions/functions/math.go` | - |
| `SQUARE` | `internal/extensions/functions/math.go` | - |
| `EXP` | `internal/extensions/functions/math.go` | - |
| `LOG` | `internal/extensions/functions/math.go` | - |
| `LOG10` | `internal/extensions/functions/math.go` | - |
| `SIN` | `internal/extensions/functions/math.go` | - |
| `COS` | `internal/extensions/functions/math.go` | - |
| `TAN` | `internal/extensions/functions/math.go` | - |
| `COT` | `internal/extensions/functions/math.go` | - |
| `ASIN` | `internal/extensions/functions/math.go` | - |
| `ACOS` | `internal/extensions/functions/math.go` | - |
| `ATAN` | `internal/extensions/functions/math.go` | - |
| `ATN2` | `internal/extensions/functions/math.go` | - |
| `PI` | `internal/extensions/functions/math.go` | - |
| `RAND` | `internal/extensions/functions/math.go` | - |
| `SIGN` | `internal/extensions/functions/math.go` | - |
| `DEGREES` | `internal/extensions/functions/math.go` | - |
| `RADIANS` | `internal/extensions/functions/math.go` | - |

### Conversion

| Element | File | Status |
| --- | --- | --- |
| `CAST` | `internal/extensions/functions/conversion.go` | Y |
| `CONVERT` | `internal/extensions/functions/conversion.go` | Y |
| `TRY_CAST` | `internal/extensions/functions/conversion.go` | - |
| `TRY_CONVERT` | `internal/extensions/functions/conversion.go` | - |
| `PARSE` | `internal/extensions/functions/conversion.go` | - |
| `TRY_PARSE` | `internal/extensions/functions/conversion.go` | - |
| `FORMAT` | `internal/extensions/functions/conversion.go` | - |

### Logical

| Element | File | Status |
| --- | --- | --- |
| `ISNULL` | `internal/extensions/functions/logical.go` | Y |
| `COALESCE` | `internal/extensions/functions/logical.go` | Y |
| `NULLIF` | `internal/extensions/functions/logical.go` | Y |
| `CASE` | `internal/extensions/functions/logical.go` | Y |
| `IIF` | `internal/extensions/functions/logical.go` | - |
| `CHOOSE` | `internal/extensions/functions/logical.go` | - |

### Metadata

| Element | File | Status |
| --- | --- | --- |
| `DB_ID` | `internal/extensions/functions/metadata.go` | Y |
| `DB_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `OBJECT_ID` | `internal/extensions/functions/metadata.go` | Y |
| `OBJECT_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `OBJECT_SCHEMA_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `SCHEMA_ID` | `internal/extensions/functions/metadata.go` | Y |
| `SCHEMA_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `TYPE_NAME` | `internal/extensions/functions/metadata.go` | Y |
| `HAS_DBACCESS` | `internal/extensions/functions/metadata.go` | Y |
| `COL_NAME` | `internal/extensions/functions/metadata.go` | - |
| `COL_LENGTH` | `internal/extensions/functions/metadata.go` | - |
| `TYPE_ID` | `internal/extensions/functions/metadata.go` | - |
| `OBJECT_DEFINITION` | `internal/extensions/functions/metadata.go` | - |
| `OBJECTPROPERTY` | `internal/extensions/functions/metadata.go` | - |
| `OBJECTPROPERTYEX` | `internal/extensions/functions/metadata.go` | - |
| `COLUMNPROPERTY` | `internal/extensions/functions/metadata.go` | - |
| `INDEXPROPERTY` | `internal/extensions/functions/metadata.go` | - |
| `INDEXKEY_PROPERTY` | `internal/extensions/functions/metadata.go` | - |
| `STATS_DATE` | `internal/extensions/functions/metadata.go` | - |

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
| `@@IDENTITY` | `internal/extensions/functions/configuration.go` | - |
| `@@SERVICENAME` | `internal/extensions/functions/configuration.go` | - |
| `@@NESTLEVEL` | `internal/extensions/functions/configuration.go` | - |
| `@@MAX_PRECISION` | `internal/extensions/functions/configuration.go` | - |
| `@@OPTIONS` | `internal/extensions/functions/configuration.go` | - |
| `@@DATEFIRST` | `internal/extensions/functions/configuration.go` | - |
| `@@LOCK_TIMEOUT` | `internal/extensions/functions/configuration.go` | - |
| `@@CURSOR_ROWS` | `internal/extensions/functions/configuration.go` | - |
| `@@PROCID` | `internal/extensions/functions/configuration.go` | - |

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
| `ORIGINAL_DB_NAME` | `internal/extensions/functions/security.go` | Y |
| `USER_ID` | `internal/extensions/functions/security.go` | - |
| `SUSER_ID` | `internal/extensions/functions/security.go` | - |
| `IS_MEMBER` | `internal/extensions/functions/security.go` | - |
| `IS_SRVROLEMEMBER` | `internal/extensions/functions/security.go` | - |
| `IS_ROLEMEMBER` | `internal/extensions/functions/security.go` | - |
| `PERMISSIONS` | `internal/extensions/functions/security.go` | - |
| `HAS_PERMS_BY_NAME` | `internal/extensions/functions/security.go` | - |
| `ORIGINAL_LOGIN` | `internal/extensions/functions/security.go` | - |
| `CONTEXT_INFO` | `internal/extensions/functions/security.go` | - |
| `SESSION_CONTEXT` | `internal/extensions/functions/security.go` | - |

### System / server

| Element | File | Status |
| --- | --- | --- |
| `SERVERPROPERTY` | `internal/extensions/functions/system.go` | Y |
| `DATABASEPROPERTYEX` | `internal/extensions/functions/system.go` | Y |
| `CONNECTIONPROPERTY` | `internal/extensions/functions/system.go` | - |
| `NEWID` | `internal/extensions/functions/system.go` | - |
| `NEWSEQUENTIALID` | `internal/extensions/functions/system.go` | - |
| `SCOPE_IDENTITY` | `internal/extensions/functions/system.go` | - |
| `IDENT_CURRENT` | `internal/extensions/functions/system.go` | - |
| `XACT_STATE` | `internal/extensions/functions/system.go` | - |
| `ERROR_MESSAGE` | `internal/extensions/functions/system.go` | - |
| `ERROR_NUMBER` | `internal/extensions/functions/system.go` | - |
| `ERROR_SEVERITY` | `internal/extensions/functions/system.go` | - |
| `ERROR_STATE` | `internal/extensions/functions/system.go` | - |
| `ERROR_LINE` | `internal/extensions/functions/system.go` | - |
| `ERROR_PROCEDURE` | `internal/extensions/functions/system.go` | - |

### Ranking & window

| Element | File | Status |
| --- | --- | --- |
| `ROW_NUMBER` | `internal/exec/window.go` | - |
| `RANK` | `internal/exec/window.go` | - |
| `DENSE_RANK` | `internal/exec/window.go` | - |
| `NTILE` | `internal/exec/window.go` | - |
| `LAG` | `internal/exec/window.go` | - |
| `LEAD` | `internal/exec/window.go` | - |
| `FIRST_VALUE` | `internal/exec/window.go` | - |
| `LAST_VALUE` | `internal/exec/window.go` | - |
| `PERCENT_RANK` | `internal/exec/window.go` | - |
| `CUME_DIST` | `internal/exec/window.go` | - |
| `PERCENTILE_CONT` | `internal/exec/window.go` | - |
| `PERCENTILE_DISC` | `internal/exec/window.go` | - |

### JSON

| Element | File | Status |
| --- | --- | --- |
| `ISJSON` | `internal/extensions/functions/json.go` | - |
| `JSON_VALUE` | `internal/extensions/functions/json.go` | - |
| `JSON_QUERY` | `internal/extensions/functions/json.go` | - |
| `JSON_MODIFY` | `internal/extensions/functions/json.go` | - |
| `JSON_PATH_EXISTS` | `internal/extensions/functions/json.go` | - |
| `JSON_OBJECT` | `internal/extensions/functions/json.go` | - |
| `JSON_ARRAY` | `internal/extensions/functions/json.go` | - |
| `OPENJSON` | `internal/extensions/functions/json.go` | - |

### Cryptographic

| Element | File | Status |
| --- | --- | --- |
| `HASHBYTES` | `internal/extensions/functions/crypto.go` | - |
| `CHECKSUM` | `internal/extensions/functions/crypto.go` | - |
| `BINARY_CHECKSUM` | `internal/extensions/functions/crypto.go` | - |
| `COMPRESS` | `internal/extensions/functions/crypto.go` | - |
| `DECOMPRESS` | `internal/extensions/functions/crypto.go` | - |
| `PWDENCRYPT` | `internal/extensions/functions/crypto.go` | - |
| `PWDCOMPARE` | `internal/extensions/functions/crypto.go` | - |

### Cursor

| Element | File | Status |
| --- | --- | --- |
| `CURSOR_STATUS` | `internal/extensions/functions/cursor.go` | - |

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
| `sys.triggers` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.all_objects` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.system_objects` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.sequences` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.synonyms` | `internal/extensions/sysviews/sysviews.go` | - |

### Columns & types

| Element | File | Status |
| --- | --- | --- |
| `sys.columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.types` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.identity_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.computed_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.all_columns` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.table_types` | `internal/extensions/sysviews/sysviews.go` | - |

### Indexes & keys

| Element | File | Status |
| --- | --- | --- |
| `sys.indexes` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.index_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.key_constraints` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.foreign_keys` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.foreign_key_columns` | `internal/extensions/sysviews/sysviews.go` | Y |
| `sys.stats` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.stats_columns` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.partitions` | `internal/extensions/sysviews/sysviews.go` | - |

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
| `sys.database_files` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.filegroups` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.extended_properties` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.sql_expression_dependencies` | `internal/extensions/sysviews/sysviews.go` | - |

### Security

| Element | File | Status |
| --- | --- | --- |
| `sys.database_principals` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.server_principals` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.database_permissions` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.server_permissions` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.database_role_members` | `internal/extensions/sysviews/sysviews.go` | - |

### Dynamic management views

| Element | File | Status |
| --- | --- | --- |
| `sys.dm_exec_sessions` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.dm_exec_connections` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.dm_exec_requests` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.dm_exec_query_stats` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.dm_os_waiting_tasks` | `internal/extensions/sysviews/sysviews.go` | - |

### Compatibility views

| Element | File | Status |
| --- | --- | --- |
| `sys.sysobjects` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.syscolumns` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.systypes` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.sysindexes` | `internal/extensions/sysviews/sysviews.go` | - |
| `sys.sysusers` | `internal/extensions/sysviews/sysviews.go` | - |

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
| `INFORMATION_SCHEMA.CHECK_CONSTRAINTS` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.SCHEMATA` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.DOMAINS` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.ROUTINE_COLUMNS` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.VIEW_TABLE_USAGE` | `internal/extensions/infoschema/infoschema.go` | - |
| `INFORMATION_SCHEMA.VIEW_COLUMN_USAGE` | `internal/extensions/infoschema/infoschema.go` | - |

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
| `sp_server_info` | `internal/engine/proc.go` | - |
| `sp_datatype_info` | `internal/engine/proc.go` | - |
| `sp_table_privileges` | `internal/engine/proc.go` | - |
| `sp_column_privileges` | `internal/engine/proc.go` | - |

### Help & scripting

| Element | File | Status |
| --- | --- | --- |
| `sp_help` | `internal/engine/proc.go` | Y |
| `sp_helptext` | `internal/engine/proc.go` | Y |
| `sp_helpindex` | `internal/engine/proc.go` | Y |
| `sp_helpconstraint` | `internal/engine/proc.go` | Y |
| `sp_helpdb` | `internal/engine/proc.go` | - |
| `sp_helptrigger` | `internal/engine/proc.go` | - |
| `sp_depends` | `internal/engine/proc.go` | - |

### Administration

| Element | File | Status |
| --- | --- | --- |
| `sp_executesql` | `internal/engine/proc.go` | - |
| `sp_rename` | `internal/engine/proc.go` | - |
| `sp_addextendedproperty` | `internal/engine/proc.go` | - |
| `sp_who` | `internal/engine/proc.go` | - |
| `sp_lock` | `internal/engine/proc.go` | - |
| `sp_configure` | `internal/engine/proc.go` | - |

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
| recursive CTE (at depth) | `internal/tsql/parser.go + internal/exec` | - |
| `PIVOT` / `UNPIVOT` | `internal/tsql/parser.go + internal/exec` | - |
| `CROSS APPLY` / `OUTER APPLY` | `internal/tsql/parser.go + internal/exec` | - |
| `GROUPING SETS` / `ROLLUP` / `CUBE` | `internal/tsql/parser.go + internal/exec` | - |
| `OVER` (window clause) | `internal/tsql/parser.go + internal/exec` | - |
| `TABLESAMPLE` | `internal/tsql/parser.go + internal/exec` | - |
| query hints (`OPTION`, `WITH (NOLOCK)`) | `internal/tsql/parser.go + internal/exec` | - |

### DML

| Element | File | Status |
| --- | --- | --- |
| `INSERT` | `internal/engine/engine.go` | Y |
| `UPDATE` | `internal/engine/engine.go` | Y |
| `DELETE` | `internal/engine/engine.go` | Y |
| `MERGE` | `internal/engine/engine.go` | - |
| `SELECT ... INTO` | `internal/engine/engine.go` | - |
| `INSERT ... OUTPUT` | `internal/engine/engine.go` | - |
| `UPDATE ... FROM` | `internal/engine/engine.go` | - |
| `DELETE ... FROM` | `internal/engine/engine.go` | - |
| `TRUNCATE TABLE` | `internal/engine/engine.go` | - |
| `BULK INSERT` | `internal/engine/engine.go` | - |

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
| `CREATE / ALTER / DROP FUNCTION` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE / ALTER / DROP TRIGGER` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE / DROP INDEX` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE / ALTER / DROP SEQUENCE` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE / DROP SYNONYM` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE / ALTER / DROP SCHEMA` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE / DROP TYPE` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `CREATE OR ALTER` | `internal/tsql + internal/extensions/{views,procedures}` | - |
| table constraints (PK / FK / UNIQUE / CHECK / DEFAULT) in DDL | `internal/tsql + internal/extensions/{views,procedures}` | - |
| `IDENTITY` / computed columns in DDL | `internal/tsql + internal/extensions/{views,procedures}` | - |

### Control-of-flow

| Element | File | Status |
| --- | --- | --- |
| `IF ... ELSE` | `internal/extensions/procedures/control/` | - |
| `WHILE` | `internal/extensions/procedures/control/` | - |
| `BEGIN ... END` | `internal/extensions/procedures/control/` | - |
| `BREAK` | `internal/extensions/procedures/control/` | - |
| `CONTINUE` | `internal/extensions/procedures/control/` | - |
| `RETURN` | `internal/extensions/procedures/control/` | - |
| `GOTO` | `internal/extensions/procedures/control/` | - |
| `WAITFOR` | `internal/extensions/procedures/control/` | - |
| `TRY ... CATCH` | `internal/extensions/procedures/control/` | - |
| `THROW` | `internal/extensions/procedures/control/` | - |
| `RAISERROR` | `internal/extensions/procedures/control/` | - |
| `PRINT` | `internal/extensions/procedures/control/` | - |

### Variables & batch

| Element | File | Status |
| --- | --- | --- |
| `DECLARE @v = ...` | `internal/extensions/batch/batch.go` | Y |
| `SET @v = ...` | `internal/extensions/batch/batch.go` | Y |
| `USE` | `internal/extensions/batch/batch.go` | Y |
| `EXEC` / `EXECUTE` (stored procedure) | `internal/extensions/batch/batch.go` | Y |
| `SELECT @v = col` | `internal/extensions/batch/batch.go` | - |
| `DECLARE @t TABLE (...)` | `internal/extensions/batch/batch.go` | - |
| `sp_executesql` / dynamic SQL | `internal/extensions/batch/batch.go` | - |
| `SET` options with effect (`ANSI_NULLS`, `QUOTED_IDENTIFIER`, `ROWCOUNT`, `IDENTITY_INSERT`) | `internal/extensions/batch/batch.go` | - |

### Transactions

| Element | File | Status |
| --- | --- | --- |
| `BEGIN TRANSACTION` | `internal/engine (Tx)` | - |
| `COMMIT` | `internal/engine (Tx)` | - |
| `ROLLBACK` | `internal/engine (Tx)` | - |
| `SAVE TRANSACTION` | `internal/engine (Tx)` | - |
| `SET XACT_ABORT` | `internal/engine (Tx)` | - |
| `SET TRANSACTION ISOLATION LEVEL` | `internal/engine (Tx)` | - |

### Cursors

| Element | File | Status |
| --- | --- | --- |
| `DECLARE ... CURSOR` | `internal/extensions/procedures/control/` | - |
| `OPEN` | `internal/extensions/procedures/control/` | - |
| `FETCH` | `internal/extensions/procedures/control/` | - |
| `CLOSE` | `internal/extensions/procedures/control/` | - |
| `DEALLOCATE` | `internal/extensions/procedures/control/` | - |

### Security

| Element | File | Status |
| --- | --- | --- |
| `GRANT` | `internal/engine (security DDL)` | - |
| `REVOKE` | `internal/engine (security DDL)` | - |
| `DENY` | `internal/engine (security DDL)` | - |
| `CREATE / ALTER / DROP USER` | `internal/engine (security DDL)` | - |
| `CREATE / ALTER / DROP LOGIN` | `internal/engine (security DDL)` | - |
| `CREATE / ALTER / DROP ROLE` | `internal/engine (security DDL)` | - |
| `EXECUTE AS` / `REVERT` | `internal/engine (security DDL)` | - |
