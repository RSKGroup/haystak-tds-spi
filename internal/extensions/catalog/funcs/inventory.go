// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

import (
	"fmt"
	"strings"
)

// Family is one cohesive scalar-function file and the full set of SQL Server functions it targets.
type Family struct {
	Name  string
	File  string
	Funcs []string
}

// Manifest is the target scalar-function surface — the superset of what SQL Server has, by family file.
// A function renders ✓ when implemented (registry, or the exceptions below) and ◻ otherwise, so "missing"
// is tracked here explicitly and never by the absence of code. Add a row to declare a new target.
var Manifest = []Family{
	{"Catalog & metadata", "catalog.go", []string{
		"DB_ID", "DB_NAME", "OBJECT_ID", "OBJECT_NAME", "OBJECT_SCHEMA_NAME", "SCHEMA_ID", "SCHEMA_NAME",
		"TYPE_NAME", "HAS_DBACCESS", "QUOTENAME",
		"COL_NAME", "COL_LENGTH", "TYPE_ID", "OBJECT_DEFINITION", "OBJECTPROPERTY", "OBJECTPROPERTYEX",
		"COLUMNPROPERTY", "INDEXPROPERTY", "STATS_DATE",
	}},
	{"String", "string.go", []string{
		"LEN", "DATALEN", "UPPER", "LOWER", "LTRIM", "RTRIM", "TRIM", "SUBSTRING", "REPLACE", "CONCAT",
		"CHARINDEX", "PATINDEX", "STUFF", "LEFT", "RIGHT", "REPLICATE", "SPACE", "REVERSE", "CONCAT_WS",
		"STRING_AGG", "STRING_SPLIT", "STRING_ESCAPE", "TRANSLATE", "FORMATMESSAGE", "UNICODE", "NCHAR",
		"CHAR", "ASCII", "SOUNDEX", "DIFFERENCE", "STR",
	}},
	{"Numeric & math", "math.go", []string{
		"ABS", "CEILING", "FLOOR", "ROUND", "POWER", "SQRT", "SQUARE", "EXP", "LOG", "LOG10", "SIN", "COS",
		"TAN", "COT", "ASIN", "ACOS", "ATAN", "ATN2", "PI", "RAND", "SIGN", "DEGREES", "RADIANS",
	}},
	{"Date & time", "datetime.go", []string{
		"YEAR", "MONTH", "DAY", "GETDATE", "GETUTCDATE", "SYSDATETIME", "SYSUTCDATETIME",
		"SYSDATETIMEOFFSET", "CURRENT_TIMESTAMP", "DATEADD", "DATEDIFF", "DATEDIFF_BIG", "DATEPART",
		"DATENAME", "EOMONTH", "DATEFROMPARTS", "DATETIMEFROMPARTS", "SWITCHOFFSET", "TODATETIMEOFFSET",
		"ISDATE", "DATETRUNC",
	}},
	{"Conversion", "conversion.go", []string{
		"CAST", "CONVERT", "TRY_CAST", "TRY_CONVERT", "PARSE", "TRY_PARSE", "FORMAT", "STR",
	}},
	{"Logical / NULL", "logical.go", []string{
		"ISNULL", "COALESCE", "NULLIF", "IIF", "CHOOSE",
	}},
	{"Security & session identity", "security.go", []string{
		"SYSTEM_USER", "CURRENT_USER", "SESSION_USER", "USER", "USER_NAME", "SUSER_NAME", "SUSER_SNAME",
		"HOST_NAME", "APP_NAME", "ORIGINAL_DB_NAME",
		"USER_ID", "SUSER_ID", "IS_MEMBER", "IS_SRVROLEMEMBER", "IS_ROLEMEMBER", "PERMISSIONS",
		"HAS_PERMS_BY_NAME", "ORIGINAL_LOGIN", "CONTEXT_INFO", "SESSION_CONTEXT",
	}},
	{"Server & config", "server.go", []string{
		"@@VERSION", "@@SERVERNAME", "@@SPID", "@@LANGUAGE", "@@ROWCOUNT", "@@ERROR", "@@TRANCOUNT",
		"@@FETCH_STATUS", "SERVERPROPERTY", "DATABASEPROPERTYEX",
		"@@SERVICENAME", "@@IDENTITY", "@@NESTLEVEL", "@@MAX_PRECISION", "@@OPTIONS", "@@DATEFIRST",
		"@@LOCK_TIMEOUT", "@@CURSOR_ROWS", "@@PROCID", "CONNECTIONPROPERTY",
	}},
	{"JSON", "json.go", []string{
		"ISJSON", "JSON_VALUE", "JSON_QUERY", "JSON_MODIFY", "JSON_PATH_EXISTS", "JSON_OBJECT", "JSON_ARRAY",
	}},
	{"Crypto & hashing", "crypto.go", []string{
		"HASHBYTES", "CHECKSUM", "BINARY_CHECKSUM", "COMPRESS", "DECOMPRESS", "PWDENCRYPT", "PWDCOMPARE",
	}},
}

// implementedElsewhere lists functions resolved outside the registry — env-aware in exec, the engine
// probe, or the tsql parser — so the inventory test still counts them implemented.
var implementedElsewhere = map[string]string{
	"OBJECT_NAME":      "exec.Env resolver",
	"DB_NAME":          "exec.Env resolver",
	"ORIGINAL_DB_NAME": "engine probe (current database)",
	"CAST":             "tsql parser (ValCast)",
	"CONVERT":          "tsql parser (ValCast)",
}

func implemented(name string) bool {
	up := strings.ToUpper(name)
	if _, ok := registry[up]; ok {
		return true
	}
	_, ok := implementedElsewhere[up]
	return ok
}

// split partitions a family's functions into implemented (with a location note for the env-aware /
// parser exceptions) and not-yet, each as a backticked label.
func split(fns []string) (done, todo []string) {
	for _, fn := range fns {
		if !implemented(fn) {
			todo = append(todo, "`"+fn+"`")
			continue
		}
		label := "`" + fn + "`"
		if loc, ok := implementedElsewhere[strings.ToUpper(fn)]; ok {
			label += " *(" + loc + ")*"
		}
		done = append(done, label)
	}
	return done, todo
}

// Render produces the README's scalar-function inventory: one block per family file, its done/total
// count, and the ✓ / ◻ function lists — derived from the code, so it cannot drift from reality.
func Render() string {
	var b strings.Builder
	for _, fam := range Manifest {
		done, todo := split(fam.Funcs)
		fmt.Fprintf(&b, "**%s** — `%s` · %d/%d\n\n", fam.Name, fam.File, len(done), len(fam.Funcs))
		if len(done) > 0 {
			fmt.Fprintf(&b, "- ✓ %s\n", strings.Join(done, ", "))
		}
		if len(todo) > 0 {
			fmt.Fprintf(&b, "- ◻ %s\n", strings.Join(todo, ", "))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}
