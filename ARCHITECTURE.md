# Architecture

Two layers, connected by one seam.

## Core — the foundation (stable; change with care)

These implement the SQL Server wire and the query engine. Almost all feature work happens *above*
them, not in them.

| Package | Responsibility |
|---|---|
| `internal/wire` | TDS protocol: PRELOGIN, LOGIN7, TLS-in-TDS, SQL_BATCH, RPC, token/result encoding |
| `internal/tsql` | T-SQL -> AST (lexer + parser) |
| `internal/exec` | expression/row evaluation (filters, joins, projection, aggregates); plus catalog scalars that need the live schema (`COL_NAME`/`COL_LENGTH`/`OBJECTPROPERTY`/`COLUMNPROPERTY`) resolved off the query `Env` |
| `internal/engine` | the read query engine; the hub that wires the feature packages together |
| `tds`, `server` | the **public SPI** a backend implements, and the wire server that drives it |

## Feature surface — the "look like SQL Server" extensions (where growth happens)

Each feature is its own package and reaches the core only through the `routines.Runner` seam (run a
SQL batch, run a parsed query, read the current database), so a feature never imports the engine and
there are no import cycles.

| Package | Responsibility | Add a … |
|---|---|---|
| `internal/extensions/functions` | scalar functions by family — `string`/`datetime`/`math`/`logical`/`metadata`/`security`/`configuration`/`system` | function -> register it in its family file |
| `internal/extensions/views` | `CREATE/ALTER/DROP VIEW` + read-time expansion | — |
| `internal/extensions/procedures` | `CREATE/DROP PROCEDURE` + `EXEC` + parameter substitution | — |
| `internal/extensions/procedures/control` | T-SQL procedural constructs, **one file per statement** | construct (`IF`/`WHILE`/…) -> new file here |
| `internal/extensions/routines` | shared base: the `Runner` seam + DDL-text helpers | — |
| `internal/extensions/sysviews` | `sys.*` catalog views | a view -> a case in `Resolve` |
| `internal/extensions/infoschema` | `INFORMATION_SCHEMA.*` views | a view -> a case in `Resolve` |
| `internal/extensions/batch` | `DECLARE`/`SET @v` batch-variable binding | — |

Each `extensions/` package maps to one SQL Server surface — `functions` (the function families, one file per
category), `sysviews` (`sys.*`), `infoschema` (`INFORMATION_SCHEMA.*`), and the stored-object packages. The
`sp_*` catalog procedures still live in `internal/engine` (coupled to its introspection helpers).

**Two homes for functions.** Pure scalars register in `functions` and resolve everywhere via
`functions.Eval` (no context). Catalog scalars that need the live schema — `OBJECT_NAME`, `COL_NAME`,
`COL_LENGTH`, `OBJECTPROPERTY`, `COLUMNPROPERTY` — evaluate in `internal/exec` off the query `Env` and
resolve only in the SELECT projection and ORDER BY (predicate clauses evaluate with a nil env).
Aggregates are an `AggFunc` enum spanning `tds`, the parser, and `internal/exec`. A few functions need
parser support, not just registration: reserved-keyword names (`LEFT`/`RIGHT`), datepart keywords
(`DATEADD`/`DATEPART`), `IIF` (desugars to `CASE`), and `TRY_CAST`/`TRY_CONVERT`.

## The seam

The engine implements `routines.Runner` and hands it to the feature packages:

```go
type Runner interface {
    Exec(ctx, sql string) (tds.Rows, error)        // run a SQL batch (procedure bodies)
    RunQuery(ctx, q *tds.Query) (tds.Rows, error)  // run a parsed query (view expansion)
    CurrentDB(ctx) string
}
```

Because the dependency only points *down* (feature -> routines -> tds), each feature is independently
buildable and testable, and contributors add capability by adding a file, not by editing the engine.

## Backend-owned storage

Views and procedures are persisted by the backend via `tds.RoutineStore` (gated by `Caps.Routines`).
The gateway stores each definition's raw body and parses/runs it at use time, so a backend needs no
SQL knowledge to support stored objects — it just keeps and returns the text.

**Catalog views are seam projections.** A `sys.*` / `INFORMATION_SCHEMA.*` view renders what the backend
supplies through an SPI seam — the `catalog.Schema` (tables, plus optional `Indexes`/`Checks`/`TableTypes`)
and the authenticated `tds.Principal` (identity-aware `sys.database_principals` etc.) — and degrades to the
correct empty shape otherwise. A backend lights up a view by *declaring data*, not by writing view code.
Relationship views (`sys.foreign_keys`, `INFORMATION_SCHEMA.*_CONSTRAINTS`) light up when the backend
declares `PrimaryKey`/`ForeignKeys`; where it sources them (sampled, or a reserved catalog store it
bootstraps) is the adapter's choice.

## Runtime sessions and audit

Live-session state is the server's, not the backend's. The server keeps an in-memory session registry
(spid assigned at LOGIN7; login/host/app/time captured; removed on disconnect). It feeds `@@SPID`, the
runtime DMVs (`sys.dm_exec_sessions`/`dm_exec_connections`/`dm_exec_requests`), and `sp_who` — which
enumerate the registry and degrade to the correct empty shape when there is no server. Nothing to enable,
nothing to store; it is ephemeral connection state.

Audit is a hook, not a store. `Server.Audit func(tds.SessionEvent)` fires on every login and logout with
`{Kind, Session{SessionID, LoginName, Host, Program, LoginTime}, At}`. The core never persists it — a
read-only backend just leaves the hook unset. An adapter "turns on audit" by assigning the hook to append
events to a store it owns and names (its retention/ILM, its index/collection), kept separate from the
catalog store. The event maps directly to SQL Server's audit fields (`At`→event time, `Kind`→action,
`SessionID`→session_id, `LoginName`→server principal, `Host`/`Program`→client host/app).

See [CONTRIBUTING.md](CONTRIBUTING.md) for the step-by-step recipes.
