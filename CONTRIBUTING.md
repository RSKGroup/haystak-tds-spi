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

1. Open the matching group file under **`internal/extensions/catalog/funcs/`** (`catalog.go` for catalog/metadata
   functions; add `string.go` / `datetime.go` / `security.go` if a new group is warranted).
2. In that file's `init`, `register("DB_NAME", func(a []any) any { … })`.

The evaluator calls `funcs.Eval` for any function it doesn't handle generically, so registration is
all it takes. Keep groups cohesive; split a file once it gets large.

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
