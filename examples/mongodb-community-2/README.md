# examples/mongodb-community-2

Same idea as [`examples/mongodb-community`](../mongodb-community) (serve a live MongoDB
over the SQL Server wire), but with a declared catalog instead of an inferred one. A
reserved system collection (`__haystak_catalog`) holds one document per table describing
its columns, primary key, and foreign keys. `Describe` reads that collection; it does
not sample documents.

This is the declared-catalog model: the schema is authoritative data the backend states
up front, not something inferred from the documents. The contrast:

| | `mongodb-community` (inferred) | `mongodb-community-2` (declared) |
|---|---|---|
| Columns | inferred by sampling ≤100 docs per collection, every `Describe` | read once from the catalog collection, no sampling |
| Primary key | always `_id` | the real business key (`id`), as declared |
| Foreign keys | none (inference can't recover them) | declared edges → `sys.foreign_keys` + join discovery (ad-hoc `JOIN … ON` works either way) |
| `_id` leak | `_id` is exposed as a column | clean SQL surface; `_id` is hidden |
| Authority | guesses, can mis-type rare fields | authoritative; the catalog is the contract |

## Why this is faster (and more correct) in production

- Discovery does no sampling. The inferred backend samples up to 100 docs per collection
  on every `Describe`; this one reads a single small, indexed collection. On a database
  with many large collections, that's the difference between repeated multi-collection
  scans and one cheap read.
- Relationships become real. FK edges can't be inferred from schemaless data. Declaring
  them is what makes `JOIN` planning, `sys.foreign_keys`, and
  `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS` work at all, not merely faster.
- Correctness. Sampling can miss a rare field or mis-type a column (null in the first
  100 docs, int in some rows and string in others). A declared catalog never guesses.

The data path itself is still thin: `Scan` returns whole collections and the gateway
engine performs the join. Join pushdown keys off the query's `ON` clause, not the
declared FK: for an equi-join the engine pushes the left-side keys as a right-side
`IN` filter (a semi-join), so the right collection is scanned for matching rows only.
That speedup applies to any `JOIN … ON` — declared FK or not, in either example. What
declaring the FK uniquely buys is the catalog relationship surface (`sys.foreign_keys`,
`INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS`) that lets BI tools and ORMs auto-discover
the join instead of you hand-writing every `ON`.

## Prerequisites

A MongoDB instance reachable without auth (the default `mongodb-community` install):

```sh
mongosh --eval 'db.runCommand({ping:1})'   # should print { ok: 1 }
```

## Run it

```sh
cd examples/mongodb-community-2
go run .                          # connects to localhost:27017, seeds db "haystakcatalog", serves :1433
go run . --db mydata             # serves your existing database "mydata" (bootstraps a catalog if absent)
```

`--db <name>` serves an existing database (bring-your-own data). Because this is the
declared-catalog model, a foreign database needs a `__haystak_catalog` to query; if one
is absent we **build it for you by inference** — sampling each collection into a draft
catalog document (columns and types inferred, primary key `id` when present else `_id`,
no foreign keys). That makes the database immediately queryable, and you then edit the
catalog to declare the PK/FK relationships inference can't recover (see below). An
existing catalog is never overwritten. With `--db` blank (the default) it seeds the demo
instead.

Environment overrides: `MONGO_URI` (default `mongodb://localhost:27017`), `MONGO_DB`
(the demo database name, default `haystakcatalog`), `ADDR` (default `127.0.0.1:1433`). In
demo mode it seeds (idempotently) three data collections (`customers`, `products`,
`orders`) and the `__haystak_catalog` system collection that declares their columns, PKs,
and the two FK edges from `orders`.

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM customers WHERE age > 40"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT c.name, p.name, o.qty FROM orders o JOIN customers c ON o.customer_id = c.id JOIN products p ON o.product_id = p.id"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name, type FROM sys.foreign_keys"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT CONSTRAINT_NAME, TABLE_NAME FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS"
```

## The system collection (`__haystak_catalog`)

One document per table. The backend reads these and assembles the SQL catalog verbatim:

```json
{
  "table": "orders",
  "columns": [
    { "name": "id",          "type": "bigint" },
    { "name": "customer_id", "type": "bigint" },
    { "name": "product_id",  "type": "bigint" },
    { "name": "qty",         "type": "bigint" }
  ],
  "primary_key": ["id"],
  "foreign_keys": [
    { "columns": ["customer_id"], "ref_table": "customers", "ref_columns": ["id"] },
    { "columns": ["product_id"],  "ref_table": "products",  "ref_columns": ["id"] }
  ]
}
```

Type names map to the SPI's kinds (`bigint`, `int`, `bit`, `float`, `decimal`,
`datetime`, `uniqueidentifier`, `varbinary`; anything else becomes `varchar`). A
collection without a catalog document is not a SQL table; the declared catalog is
authoritative.

### Bootstrap contract

Because the catalog is declared, the system collection must exist before a collection is
a SQL table. There are three ways it gets populated:

- **Demo mode** (`go run .`) seeds `__haystak_catalog` along with the demo data.
- **BYO mode** (`go run . --db <name>`) bootstraps it for you: we infer each collection's
  definition (columns, types, and a best-guess primary key) and write one catalog
  document per collection. The database is queryable immediately; what we can't infer is
  relationships, so the FK list starts empty and you declare the edges yourself.
- **DDL** keeps it consistent at runtime: `CREATE TABLE` creates the collection and writes
  a catalog document, `ALTER TABLE … ADD` appends columns, `DROP TABLE` removes both the
  collection and its declaration. T-SQL `CREATE TABLE` here doesn't carry PK/FK clauses,
  so those are declared by editing the catalog too.

Declaring a relationship is a single edit to the catalog document — add the column (if
new) and push the FK edge. For a bootstrapped `events` collection that points at
`widgets`:

```js
db.getCollection("__haystak_catalog").updateOne(
  { table: "events" },
  { $set: { primary_key: ["id"],
            foreign_keys: [
              { columns: ["widget_id"], ref_table: "widgets", ref_columns: ["id"] }
            ] } }
)
```

`Describe` reads the catalog on every query, so the change is live with no restart:
`sys.foreign_keys` / `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS` now show the edge, and
`SELECT … FROM events e JOIN widgets w ON e.widget_id = w.id` resolves through it.

## Authentication

Identical to `mongodb-community`: the gateway→Mongo layer parses standard
connection-string auth (`MONGO_URI`, or `MONGO_USER`/`MONGO_PASS`/`MONGO_AUTHDB`); the
TDS-client→gateway layer is not authenticated in this demo (front it with TLS or network
controls, or validate LOGIN7 in your own `server.Server` wrapper).

## Separate Go module

This example has its own `go.mod` (requiring `go.mongodb.org/mongo-driver`, Apache-2.0)
with a `replace` to the parent, so the core library stays dependency-free. Build and
test from its own directory:

```sh
cd examples/mongodb-community-2 && go build ./... && go test ./...   # the test skips if mongod is down
```

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
