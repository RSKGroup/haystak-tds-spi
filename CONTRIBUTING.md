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

The evaluator calls `functions.Eval` for any function it doesn't handle generically, so registration is
all it takes. Keep groups cohesive; split a file once it gets large.

Three things take more than a registration:

- **Catalog scalars that need the live schema** (`COL_NAME`, `OBJECTPROPERTY`, …): add a case in
  `internal/exec/value.go`'s `evalFunc` keyed off the query `Env` (built by `CatalogObjects`); the
  registry has no catalog access. These resolve in SELECT/ORDER BY only.
- **Parser-backed functions**: reserved-keyword names (`LEFT`/`RIGHT`), datepart keywords
  (`DATEADD`/`DATEPART`), `IIF`, and `TRY_CAST`/`TRY_CONVERT` need a case in `internal/tsql/parser.go`.
- **Aggregates**: add an `AggFunc` value in `tds`, a case in the parser's `isAggName`/`aggOf`, and the
  computation in `internal/exec/aggregate.go`.

Keep each SQL Server surface's own mapping: `COL_LENGTH`, `sys.columns.max_length`, `sp_columns` LENGTH,
and `CHARACTER_MAXIMUM_LENGTH` each own their length logic. Do not fold superficially-similar functions
into one shared helper just because their bodies match today — distinct surfaces evolve independently.

## Add a catalog view (`sys.*` / `INFORMATION_SCHEMA.*`)

1. Open `internal/extensions/sysviews/sysviews.go` (or `infoschema/infoschema.go`).
2. Add a `case "<view>"` in `Resolve` returning a builder's `(cols, data)`.
3. The builder reads the backend's `catalog.Schema` + routines and emits the view's columns as rows.
   An empty-but-shaped view (no data yet) still returns its full column set so clients can describe it.

Prefer **seam-first**: a view should *project over an SPI seam*, not hardcode rows — the `catalog.Schema`
(tables, plus optional `Indexes`/`Checks`/`TableTypes`) or the authenticated `tds.Principal` passed to
`Resolve` (for identity-aware views like `sys.database_principals`). If the concept isn't modeled yet, add
an additive field to `catalog` and have backends populate it; the view then lights up by *declared data*.

## Add a stored-object capability (views / procedures)

Work in `internal/extensions/views` or `internal/extensions/procedures`. These persist definitions through the public
`tds.RoutineStore` and execute via the `routines.Runner` seam — so they stay decoupled from the
engine. If you need a new execution primitive, extend `routines.Runner` (and the engine's adapter),
not the feature package's imports.

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
go test ./...
```

Conventions: short comments (why, not what — one line where it isn't obvious), Apache-2.0 + SPDX
header on every `.go`, and each new file lands in the package its recipe above names. If a feature
needs a new package, follow the same rule — it depends *down* on `routines`/`tds`, never up into the
engine.
