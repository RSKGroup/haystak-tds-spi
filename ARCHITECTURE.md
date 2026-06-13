# Architecture

Two layers, connected by one seam.

## Core — the foundation (stable; change with care)

These implement the SQL Server wire and the query engine. Almost all feature work happens *above*
them, not in them.

| Package | Responsibility |
|---|---|
| `internal/wire` | TDS protocol: PRELOGIN, LOGIN7, TLS-in-TDS, SQL_BATCH, RPC, token/result encoding |
| `internal/tsql` | T-SQL → AST (lexer + parser) |
| `internal/exec` | expression and row evaluation (filters, joins, projection, generic functions) |
| `internal/engine` | the read query engine; the hub that wires the feature packages together |
| `tds`, `server` | the **public SPI** a backend implements, and the wire server that drives it |

## Feature surface — the "look like SQL Server" extensions (where growth happens)

Each feature is its own package and reaches the core only through the `routines.Runner` seam (run a
SQL batch, run a parsed query, read the current database), so a feature never imports the engine and
there are no import cycles.

| Package | Responsibility | Add a … |
|---|---|---|
| `internal/extensions/catalog/funcs` | scalar system/catalog functions (`DB_ID`, `HAS_DBACCESS`, `QUOTENAME`, …) | function → register it in a group file |
| `internal/extensions/views` | `CREATE/ALTER/DROP VIEW` + read-time expansion | — |
| `internal/extensions/procedures` | `CREATE/DROP PROCEDURE` + `EXEC` + parameter substitution | — |
| `internal/extensions/procedures/control` | T-SQL procedural constructs, **one file per statement** | construct (`IF`/`WHILE`/…) → new file here |
| `internal/extensions/routines` | shared base: the `Runner` seam + DDL-text helpers | — |

`internal/sysviews` and `internal/infoschema` (the `sys.*` and `INFORMATION_SCHEMA.*` catalogs) are
stable today; they are slated to move under `internal/extensions/catalog/` as the catalog surface consolidates.

## The seam

The engine implements `routines.Runner` and hands it to the feature packages:

```go
type Runner interface {
    Exec(ctx, sql string) (tds.Rows, error)        // run a SQL batch (procedure bodies)
    RunQuery(ctx, q *tds.Query) (tds.Rows, error)  // run a parsed query (view expansion)
    CurrentDB(ctx) string
}
```

Because the dependency only points *down* (feature → routines → tds), each feature is independently
buildable and testable, and contributors add capability by adding a file, not by editing the engine.

## Backend-owned storage

Views and procedures are persisted by the backend via `tds.RoutineStore` (gated by `Caps.Routines`).
The gateway stores each definition's raw body and parses/runs it at use time, so a backend needs no
SQL knowledge to support stored objects — it just keeps and returns the text.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the step-by-step recipes.
