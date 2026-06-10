# examples/inmem

The simplest possible backend: an in-memory store, about one file. Read it first to
learn the SPI.

## What it is

A `package inmem` that implements:

- `tds.Backend` — `Describe` (returns its tables/columns/PKs/FKs) and `Capabilities`.
- `tds.Scanner` — the thin path. `Scan` returns a snapshot of a whole table and the
  gateway engine does all the query work (WHERE/JOIN/GROUP BY/ORDER BY/paging/
  subqueries/CTEs/expressions).
- `tds.Writer` and `tds.DDL` — INSERT/UPDATE/DELETE and CREATE/DROP TABLE, so writes
  work too.

It advertises `Caps{Pushdown, Writable, DDL}`.

## How it works

Tables live in a `map[string]*table` (`{def catalog.Table; rows [][]any}`) guarded by a
`sync.Mutex`. `Scan` copies the rows under the lock and hands them to the engine, so
concurrent reads see a stable snapshot while writes mutate in place. `Insert`/`Update`/
`Delete` walk the rows applying the parsed predicates; DDL adds or removes entries in
the table map. Being a thin backend, it carries no query logic of its own. That's the
whole point.

## Fixtures (what you can query)

`users`, `orders` (FK → users), `items` (typed columns: decimal / uniqueidentifier /
datetime2 / varbinary), and `depts` / `emps` (with a deliberate orphan and an empty
dept, to exercise LEFT/RIGHT/FULL joins and correlated subqueries).

## Used by

- [`examples/gateway`](../gateway) runs a TDS gateway over this backend.
- `inmem_test.go` runs `tdstest.RunConformance` against it.

To build your own backend, copy this file and replace the map with your store.

## License

Apache-2.0 — see [LICENSE](../../LICENSE).
