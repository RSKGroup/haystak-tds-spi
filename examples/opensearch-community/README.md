# examples/opensearch-community

A real, non-SQL backend: serve a live OpenSearch cluster over the SQL Server wire. Any TDS
client (sqlcmd, SSMS, Power BI, BI tools, MCP SQL connectors) can then read and write your
OpenSearch indices as if they were SQL tables: joins, aggregates, subqueries and all,
evaluated by the gateway engine on top of OpenSearch.

This is the OpenSearch analog of [`examples/mongodb-community`](../mongodb-community) and
[`examples/elasticsearch-community`](../elasticsearch-community): the same inferred-catalog
model, against OpenSearch. Indices are tables, document fields are columns (inferred by
sampling `_source`), and the document `_id` is the primary key. A declared-catalog variant
that adds real foreign keys lives in
[`examples/opensearch-community-2`](../opensearch-community-2).

## Prerequisites

An OpenSearch cluster reachable over HTTP. A quick one in Docker (security disabled, on
9201 so it doesn't collide with a local Elasticsearch on 9200):

```sh
docker run -d --name os -p 9201:9200 \
  -e discovery.type=single-node -e DISABLE_SECURITY_PLUGIN=true \
  opensearchproject/opensearch:2.17.1
curl -s http://localhost:9201 | grep cluster_name
```

## Run it

```sh
cd examples/opensearch-community
go run .                          # connects to localhost:9201, seeds demo indices, serves :1433
go run . --db 'sales-*'           # serves your existing indices (no seeding)
go run . --host os.internal:9200 --db 'sales-*'
```

`--host <url:port>` is optional (default `http://localhost:9201`; a bare `host:port` gets
`http://`). `--db <pattern>` serves an existing set of indices (bring-your-own data): the
inferred-catalog backend samples whatever is there, so it just works — nothing is built and
nothing is seeded. With `--db` blank (the default) it seeds two demo indices (`users`,
`orders`) and serves them.

Environment overrides: `OS_URL` (used when `--host` is blank), `ADDR` (default
`127.0.0.1:1433`), and `OS_USER`/`OS_PASS` for a secured cluster. Seeding creates each
index explicitly and indexes the demo documents (idempotent, only if empty).

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM users WHERE age > 40"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "INSERT INTO users (id, name, age) VALUES (4, 'linus', 30)"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES"
```

## Authentication

There are two independent layers, same as the Mongo and ES examples:

1. Gateway → OpenSearch. Supported via `--host`/`OS_URL` (HTTPS, embedded credentials) or
   `OS_USER`/`OS_PASS`. A security-disabled local cluster needs neither.
2. TDS client → gateway. This demo does not authenticate incoming TDS logins; it ignores
   the LOGIN7 username and password. Front the gateway with TLS and/or network controls, or
   validate LOGIN7 in your own `server.Server` wrapper. OpenSearch-side credentials never
   reach the client.

## How it maps (`opensearch/opensearch.go`)

| SQL | OpenSearch |
|---|---|
| database | the cluster (one logical database) |
| table | index |
| column | document field (inferred by sampling up to 100 docs), `_id` first as the PK |
| `SELECT … WHERE/JOIN/GROUP BY/…` | `Scan` runs `match_all` and returns hits; the engine does the query |
| `INSERT`/`UPDATE`/`DELETE` | `index` (refresh) / `_update_by_query` (Painless `$set`) / `_delete_by_query` |
| `CREATE`/`DROP TABLE` | create / delete index |

It's a thin backend (`Caps{Pushdown, Writable, DDL}`) using the inferred-catalog model:
`Describe` samples each index's `_source` to discover columns and their types, so a
schemaless document store presents a stable SQL catalog. JSON numbers are decoded with
`UseNumber`, so integers infer as `bigint` and decimals as `float`. OpenSearch has no native
multi-database concept, so the backend exposes a single logical database (the cluster) and
does not implement `Databaser`. `Scan` returns up to 10,000 rows per index (the default
`max_result_window`); the declared-catalog variant is the better fit for large indices.

## Separate Go module

This example has its own `go.mod` (requiring
`github.com/opensearch-project/opensearch-go/v2`, Apache-2.0) with a `replace` to the
parent, so the core library stays dependency-free.

```sh
cd examples/opensearch-community && go build ./... && go test ./...   # the test skips if OpenSearch is down
```

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
