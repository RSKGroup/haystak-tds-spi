# examples/elasticsearch-community

A real, non-SQL backend: serve a live Elasticsearch cluster over the SQL Server wire. Any
TDS client (sqlcmd, SSMS, Power BI, BI tools, MCP SQL connectors) can then read and write
your ES indices as if they were SQL tables: joins, aggregates, subqueries and all,
evaluated by the gateway engine on top of Elasticsearch.

This is the Elasticsearch analog of [`examples/mongodb-community`](../mongodb-community):
the same inferred-catalog model, against a different store. Indices are tables, document
fields are columns (inferred by sampling `_source`), and the document `_id` is the primary
key. A declared-catalog variant that adds real foreign keys lives in
[`examples/elasticsearch-community-2`](../elasticsearch-community-2).

## Prerequisites

An Elasticsearch cluster reachable over HTTP:

```sh
curl -s http://localhost:9200 | grep cluster_name   # should print your cluster
```

## Run it

```sh
cd examples/elasticsearch-community
go run .                          # connects to localhost:9200, seeds demo indices, serves :1433
go run . --db 'sales-*'           # serves your existing indices (no seeding)
go run . --host es.internal:9200 --db 'sales-*'
```

`--host <url:port>` is optional (default `http://localhost:9200`; a bare `host:port` gets
`http://`). `--db <pattern>` serves an existing set of indices (bring-your-own data): the
inferred-catalog backend samples whatever is there, so it just works — nothing is built
and nothing is seeded. With `--db` blank (the default) it seeds two demo indices (`users`,
`orders`) and serves them.

Environment overrides: `ES_URL` (used when `--host` is blank), `ADDR` (default
`127.0.0.1:1433`), and `ES_USER`/`ES_PASS` for a secured cluster. Seeding creates each
index explicitly and indexes the demo documents (idempotent, only if empty) — explicit
creation so the demo also works on clusters with `action.auto_create_index` disabled.

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM users WHERE age > 40"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "INSERT INTO users (id, name, age) VALUES (4, 'linus', 30)"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES"
```

## Authentication

There are two independent layers, same as the Mongo example:

1. Gateway → Elasticsearch. Supported via `ES_URL` (HTTPS, embedded credentials) or
   `ES_USER`/`ES_PASS`. The default local cluster runs without auth, so neither is needed.
2. TDS client → gateway. This demo does not authenticate incoming TDS logins; it ignores
   the LOGIN7 username and password. Front the gateway with TLS and/or network controls,
   or validate LOGIN7 in your own `server.Server` wrapper. ES-side credentials never reach
   the client.

## How it maps (`es/es.go`)

| SQL | Elasticsearch |
|---|---|
| database | the cluster (one logical database) |
| table | index |
| column | document field (inferred by sampling up to 100 docs), `_id` first as the PK |
| `SELECT … WHERE/JOIN/GROUP BY/…` | `Scan` runs `match_all` and returns hits; the engine does the query |
| `INSERT`/`UPDATE`/`DELETE` | `index` (refresh) / `_update_by_query` (Painless `$set`) / `_delete_by_query` |
| `CREATE`/`DROP TABLE` | create / delete index |

It's a thin backend (`Caps{Pushdown, Writable, DDL}`) using the inferred-catalog model:
`Describe` samples each index's `_source` to discover columns and their types, so a
schemaless document store presents a stable SQL catalog. The engine then layers all SQL
semantics on top. JSON numbers are decoded with `UseNumber`, so integers infer as `bigint`
and decimals as `float`, rather than everything collapsing to `float`.

ES has no native multi-database concept, so this backend exposes a single logical database
(the cluster) and does not implement `Databaser`. `Scan` returns up to 10,000 rows per
index (the cluster's default `max_result_window`); the declared-catalog variant is the
better fit for large indices anyway.

## Separate Go module

This example has its own `go.mod` (requiring `github.com/elastic/go-elasticsearch/v9`,
Apache-2.0) with a `replace` to the parent. That keeps the core library dependency-free:
`go build ./...` at the repo root never pulls the ES client. Build and test this example
from its own directory:

```sh
cd examples/elasticsearch-community && go build ./... && go test ./...   # the test skips if ES is down
```

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
