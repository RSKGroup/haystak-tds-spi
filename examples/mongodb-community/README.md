# examples/mongodb-community

A real, non-SQL backend: serve a live MongoDB over the SQL Server wire. Any TDS client
(sqlcmd, SSMS, Power BI, BI tools, MCP SQL connectors) can then read and write your
Mongo collections as if they were SQL tables: joins, aggregates, subqueries and all,
evaluated by the gateway engine on top of Mongo.

This is the worked example showing the SPI generalizes beyond relational stores.

## Prerequisites

A MongoDB instance reachable without auth (the default `mongodb-community` install):

```sh
mongosh --eval 'db.runCommand({ping:1})'   # should print { ok: 1 }
```

## Run it

```sh
cd examples/mongodb-community
go run .                          # connects to localhost:27017, seeds db "haystakdemo", serves :1433
go run . --db mydata             # serves your existing database "mydata" (no seeding)
```

`--db <name>` serves an existing database (bring-your-own data): the inferred-catalog
backend samples whatever collections are there, so it just works — nothing is built and
nothing is seeded. With `--db` blank (the default) it seeds the demo database instead.

Environment overrides: `MONGO_URI` (default `mongodb://localhost:27017`), `MONGO_DB`
(the demo database name, default `haystakdemo`), `ADDR` (default `127.0.0.1:1433`). In
demo mode it seeds using Mongo's dynamic create-collection and insert (idempotent, only
if empty), so there's data to query immediately.

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM users WHERE age > 40"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "INSERT INTO users (id, name, age) VALUES (4, 'linus', 30)"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES"
```

## Authentication

There are two independent layers:

1. Gateway → MongoDB. Supported. Put credentials in the connection string (the driver
   parses standard SCRAM / X.509 / `authSource` auth):

   ```sh
   MONGO_URI='mongodb://user:pass@localhost:27017/?authSource=admin' go run .
   ```

   …or use the convenience env vars, applied on top of the URI:

   ```sh
   MONGO_USER=app MONGO_PASS=secret MONGO_AUTHDB=admin go run .
   ```

   The default `mongodb-community` install runs without auth, so neither is needed
   locally.

2. TDS client → gateway. This demo does not authenticate incoming TDS logins; it
   ignores the LOGIN7 username and password, so any `-U/-P` works. That's a
   gateway/deployment concern, not a backend one: front the gateway with TLS
   (`HAYSTAK_TLS`-style config) and/or network controls, or validate LOGIN7 in your own
   `server.Server` wrapper. Mongo-side credentials never reach the client.

## How it maps (`mongo/mongo.go`)

| SQL | Mongo |
|---|---|
| database | database |
| table | collection |
| column | document field (inferred by sampling up to 100 docs) |
| `SELECT … WHERE/JOIN/GROUP BY/…` | `Scan` returns whole collections; the engine does the query |
| `INSERT`/`UPDATE`/`DELETE` | `InsertMany` / `UpdateMany($set)` / `DeleteMany` (predicates → Mongo filter) |
| `CREATE`/`DROP TABLE` | create / drop collection |
| `CREATE`/`DROP DATABASE` | create marker collection / drop database |
| `sys.databases`, `Query.Database` | `Databaser` (`ListDatabaseNames` + `DescribeDatabase`) |

It's a thin backend (`Caps{Pushdown, Writable, DDL}`) using the inferred-catalog model:
`Describe` samples each collection to discover columns and their types (`_id` first), so
a schemaless store presents a stable SQL catalog. The engine then layers all SQL
semantics on top.

## Separate Go module

This example has its own `go.mod` (requiring `go.mongodb.org/mongo-driver`, Apache-2.0)
with a `replace` to the parent. That keeps the core library dependency-free: `go build
./...` at the repo root never pulls the Mongo driver. Build and test this example from
its own directory:

```sh
cd examples/mongodb-community && go build ./... && go test ./...   # the test skips if mongod is down
```

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
