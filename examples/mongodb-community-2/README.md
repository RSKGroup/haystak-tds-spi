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
| Foreign keys | none (inference can't recover them) | declared edges → joins + `sys.foreign_keys` work |
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
engine performs the join. Declared FKs are the prerequisite a planner needs to push
joins down (using the indexed key), and that pushdown is where big-join speedups land;
the wins above are guaranteed today.

## Prerequisites

A MongoDB instance reachable without auth (the default `mongodb-community` install):

```sh
mongosh --eval 'db.runCommand({ping:1})'   # should print { ok: 1 }
```

## Run it

```sh
cd examples/mongodb-community-2
go run .                          # connects to localhost:27017, seeds db "haystakcatalog", serves :1433
```

Environment overrides: `MONGO_URI` (default `mongodb://localhost:27017`), `MONGO_DB`
(default `haystakcatalog`), `ADDR` (default `127.0.0.1:1433`). On start it seeds
(idempotently) three data collections (`customers`, `products`, `orders`) and the
`__haystak_catalog` system collection that declares their columns, PKs, and the two FK
edges from `orders`.

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

Because the catalog is declared, the system collection must exist and be seeded (this
example seeds it on start). DDL keeps it consistent. `CREATE TABLE` creates the
collection and writes a catalog document (columns and types from the statement;
relationships are added by editing the catalog collection, since T-SQL `CREATE TABLE`
here doesn't carry PK/FK clauses). `ALTER TABLE … ADD` appends columns to the document,
and `DROP TABLE` removes both the collection and its declaration.

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
