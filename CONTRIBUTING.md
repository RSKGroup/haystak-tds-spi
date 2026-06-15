# Contributing

Thanks for extending the gateway. Read [ARCHITECTURE.md](ARCHITECTURE.md) first — the one rule is:
**add capability in the feature packages, not in the core.** Each recipe below is "create a file in a
specific place + register it." None requires touching `internal/engine`, `internal/exec`, or
`internal/wire`.

## Add a procedural construct (e.g. the `IF` statement)

1. Create **`internal/extensions/procedures/control/if.go`**.
2. Parse your construct (`IF <cond> <stmt> [ELSE <stmt>]`) into a statement value.
3. Implement its `Exec` against the procedure scope + the `routines.Runner`.
4. Register it under its leading keyword (`IF`) so the body runner dispatches to it.

That's the whole change — `while.go`, `declare.go`, `set.go`, `return.go` follow the same shape, one
file each. Build out control flow demand-driven; you never edit the constructs that already exist.

## Add a scalar / catalog function (e.g. `DB_NAME`)

1. Open the file for that SQL Server function category under **`internal/extensions/functions/`** —
   `metadata.go`, `string.go`, `datetime.go`, `math.go`, `logical.go`, `security.go`, `configuration.go`,
   `system.go` — or add a new category file (e.g. `json.go`) if one is warranted.
2. In that file's `init`, `register("DB_NAME", func(a []any) any { … })`.
3. Add a surface case (see "The regression gate" below) and flip its README cell.

The evaluator calls `functions.Eval` for any function it doesn't handle generically, so registration is
all it takes. Keep groups cohesive; split a file once it gets large.

Three things take more than a registration:

- **Catalog scalars that need the live schema** (`COL_NAME`, `OBJECTPROPERTY`, …): add a case in
  `internal/exec/value.go`'s `evalFunc` keyed off the query `Env` (built by `CatalogObjects`), and add the
  name to `envScalars` there; the registry has no catalog access. These resolve in SELECT/ORDER BY only.
- **Parser-backed functions**: reserved-keyword names (`LEFT`/`RIGHT`), datepart keywords
  (`DATEADD`/`DATEPART`), `IIF`, and `TRY_CAST`/`TRY_CONVERT` need a case in `internal/tsql/parser.go`.
  A parser-only construct (one not also in the registry) goes in `specialFuncs` there.
- **Aggregates**: add an `AggFunc` value in `tds`, an entry in the `aggByName` map in
  `internal/exec/aggregate.go`, and the computation alongside it.

Keep each SQL Server surface's own mapping: `COL_LENGTH`, `sys.columns.max_length`, `sp_columns` LENGTH,
and `CHARACTER_MAXIMUM_LENGTH` each own their length logic. Do not fold superficially-similar functions
into one shared helper just because their bodies match today — distinct surfaces evolve independently.

## Add a catalog view (`sys.*` / `INFORMATION_SCHEMA.*`)

1. Open `internal/extensions/sysviews/sysviews.go` (or `infoschema/infoschema.go`).
2. Add an entry to the `viewBuilders` map keyed by the view name, returning a builder's `(cols, data)`.
   The map keys are the authoritative wired set `SupportedViews` reports to the gate.
3. The builder reads the backend's `catalog.Schema` + routines and emits the view's columns as rows.
   An empty-but-shaped view (no data yet) still returns its full column set so clients can describe it.
4. Add a surface case (below) and flip its README cell. (Built-in `sp_*` procs work the same way via the
   `procDispatch` map in `internal/engine/proc.go`.)

Prefer **seam-first**: a view should *project over an SPI seam*, not hardcode rows — the `catalog.Schema`
(tables, plus optional `Indexes`/`Checks`/`TableTypes`) or the authenticated `tds.Principal` passed to
`Resolve` (for identity-aware views like `sys.database_principals`). If the concept isn't modeled yet, add
an additive field to `catalog` and have backends populate it; the view then lights up by *declared data*.

## Add a stored-object capability (views / procedures)

Work in `internal/extensions/views` or `internal/extensions/procedures`. These persist definitions through the public
`tds.RoutineStore` and execute via the `routines.Runner` seam — so they stay decoupled from the
engine. If you need a new execution primitive, extend `routines.Runner` (and the engine's adapter),
not the feature package's imports.

## The regression gate

Every wired element (registered function, env scalar, aggregate, `sys.*`/`INFORMATION_SCHEMA.*` view,
built-in `sp_*`, parser-special construct) must have at least one **surface case**. Cases are data tables
in `tds/tdstest/surface_<area>.go`: `{Element, Name, SQL, Want}` (exact single-row result) or
`{..., Check}` (column/shape assertion for catalog cases). `Element` is namespaced: `func:UPPER`,
`agg:SUM`, `sys:tables`, `infoschema:COLUMNS`, `proc:sp_help`, `parser:CAST`.

`tdstest.RunSurface(t, backend)` runs the whole suite against every backend; backend-independent cases
(tableless scalars, aggregates over the static `sys.types`, catalog column-shape) assert identically
everywhere. `TestSurfaceCompleteness` reflects the dispatch tables in code (`functions.RegisteredNames`,
`exec.AggregateNames`/`EnvScalarNames`, `sysviews`/`infoschema.SupportedViews`, `engine.SupportedProcs`,
`tsql.SpecialFuncs`) and fails on either a **gap** (a wired element with no case) or an **orphan** (a case
naming an element that is not wired). So: wire the element, add its case, flip its README cell; the gate
will not pass otherwise, and coverage can never silently fall behind the code.

## Implementing a backend (downstream / community)

You don't touch any of the above. Implement `tds.Backend` plus the optional interfaces you support —
`Scanner`/`QueryExecutor`, `Writer`, `DDL`, `Databaser`, `Authenticator`, and `RoutineStore` for
views/procedures — and advertise them via `Caps`. Validate with `tds/tdstest.RunConformance`. See the
runnable backends under [`examples/`](examples/).

## Before you open a PR

```sh
gofmt -w ./...
go vet ./...
go build ./...
golangci-lint run ./...
go test -race ./...
```

CI (`.github/workflows/ci.yml`) runs the same checks plus a live-Elasticsearch integration job; the
integration tests skip under `go test -short`.

Conventions: short comments (why, not what — one line where it isn't obvious), Apache-2.0 + SPDX
header on every `.go`, and each new file lands in the package its recipe above names. If a feature
needs a new package, follow the same rule — it depends *down* on `routines`/`tds`, never up into the
engine.
