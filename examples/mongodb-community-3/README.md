# examples/mongodb-community-3

Serve a live MongoDB over the SQL Server wire with a hardcoded catalog: the schema
(columns, primary keys, and foreign keys) is a Go literal
([`mongo.staticSchema`](mongo/mongo.go)). `Describe` never reads Mongo; it returns the
literal. `Scan` still reads the real collections for data. This is the pure hardcoded
(`Static`) catalog model: the schema is fixed in code.

## The three Mongo catalog models: where does catalog truth live?

This is the third of three examples over the same Mongo data, each sourcing the catalog
differently:

| | [`mongodb-community`](../mongodb-community) | [`mongodb-community-2`](../mongodb-community-2) | `mongodb-community-3` (this) |
|---|---|---|---|
| catalog truth lives in | the data (sampled) | a Mongo system collection | Go source |
| relationships (FK) | none (uninferable) | declared rows | Go literals |
| discovery cost | samples ≤100 docs/collection, every `Describe` | one read | zero store reads |
| bootstrap | none | seed `__haystak_catalog` | none |
| reflects runtime schema change | yes (next sample) | yes (DDL writes the store) | no — needs recompile |
| runtime DDL (`CREATE TABLE`) | yes | yes | no |
| capabilities | `{Pushdown, Writable, DDL}` | `{Pushdown, Writable, DDL}` | `{Pushdown, Writable}` |

Pointed at the same `haystakcatalog` data as example 2, this backend produces the
identical SQL surface (real `id` PKs, joins, `sys.foreign_keys`, no `_id` leak), so the
two are a clean A/B: same data, same result, catalog truth in code versus in the store.

### Why hardcoded, and its honest cost

- The wins: the fastest possible discovery (no reads at all), zero bootstrap, dead
  simple.
- The cost: it can silently lie. Add a collection or a field in Mongo and it stays
  invisible until this code changes. The catalog is a promise the code makes, not a
  reflection of the store.
- No DDL by design. Capabilities are `{Pushdown, Writable}`: data writes (`INSERT`/
  `UPDATE`/`DELETE`) work on the known tables, but there's no runtime `CREATE TABLE`. A hardcoded
  catalog can't reflect a new table without a recompile. (Example 2 is
  DDL-capable because it writes the catalog store; that contrast is the point.)

Best for genuinely fixed, known schemas: a frozen reference dataset, or a controlled app
schema you own and version with the code.

## Prerequisites

A MongoDB instance reachable without auth (the default `mongodb-community` install):

```sh
mongosh --eval 'db.runCommand({ping:1})'   # should print { ok: 1 }
```

## Run it

```sh
cd examples/mongodb-community-3
go run .                          # connects to localhost:27017, seeds db "haystakcatalog", serves :1433
```

Environment overrides: `MONGO_URI` (default `mongodb://localhost:27017`), `MONGO_DB`
(default `haystakcatalog`), `ADDR` (default `127.0.0.1:1433`). On start it seeds
(idempotently) only the three data collections: `customers`, `products`, `orders`.
There is no system collection.

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM customers WHERE age > 40"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT c.name, p.name, o.qty FROM orders o JOIN customers c ON o.customer_id = c.id JOIN products p ON o.product_id = p.id"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name, type_desc FROM sys.foreign_keys"
```

The FKs in `sys.foreign_keys` come straight from `staticSchema`; nothing in Mongo
declares them.

## Separate Go module

This example has its own `go.mod` (requiring `go.mongodb.org/mongo-driver`, Apache-2.0)
with a `replace` to the parent, so the core library stays dependency-free. Build and
test from its own directory:

```sh
cd examples/mongodb-community-3 && go build ./... && go test ./...   # the test skips if mongod is down
```

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
