# examples/elasticsearch-community-2

Same idea as [`examples/elasticsearch-community`](../elasticsearch-community) (serve a live
Elasticsearch cluster over the SQL Server wire), but with a **declared** catalog instead of
a purely inferred one — adapted to a store that already has a schema.

Elasticsearch is not schemaless: every index has a native `_mapping`. So this example is a
hybrid, and that split is the whole point:

| | source |
|---|---|
| Columns & types | the index's native `_mapping` (authoritative, no sampling) |
| Primary key | the reserved `haystak_catalog` index |
| Foreign keys | the reserved `haystak_catalog` index |

ES already knows the fields and their types, so re-declaring them by hand (as the Mongo
[`mongodb-community-2`](../mongodb-community-2) example must) would be busywork. What ES
mappings *never* carry is relationships — primary keys and foreign keys — and those are
exactly what unlock joins discovery, `sys.foreign_keys`, and
`INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS`. So the catalog index declares only the keys.

> The reserved index is `haystak_catalog`, not Mongo's `__haystak_catalog`: Elasticsearch
> index names may not start with `_`.

## Why this is better than inference on ES

- Types are correct, not guessed. Sampling can mis-type a column (an integer that's null in
  the first 100 docs, a field that's `keyword` in the mapping but looks like free text).
  The mapping is authoritative.
- Discovery does no sampling — one read of the mapping per index, not a scan of documents.
- Relationships become real. FK edges can't be inferred from documents *or* recovered from
  mappings; declaring them is what makes `JOIN` discovery and the catalog views work.

## Prerequisites

An Elasticsearch cluster reachable over HTTP:

```sh
curl -s http://localhost:9200 | grep cluster_name
```

## Run it

```sh
cd examples/elasticsearch-community-2
go run .                          # connects to localhost:9200, seeds demo indices + catalog, serves :1433
go run . --db 'sales-*'           # serves your existing indices; bootstraps the catalog from mappings if absent
go run . --host es.internal:9200 --db 'sales-*'
```

`--host <url:port>` is optional (default `http://localhost:9200`; bare `host:port` gets
`http://`). `--db <pattern>` serves an existing set of indices (bring-your-own data); if
`haystak_catalog` is absent we **bootstrap it from the mappings** — one document per index
with the columns already known to ES, primary key set to `id` when that field exists, and
foreign keys left empty for you to declare. With `--db` blank it seeds three demo indices
(`customers`, `products`, `orders`) with explicit mappings and the catalog that declares
their PKs and the two FK edges from `orders`.

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM customers WHERE age > 40"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT c.name, p.name, o.qty FROM orders o JOIN customers c ON o.customer_id = c.id JOIN products p ON o.product_id = p.id"
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name, type FROM sys.foreign_keys"
```

## Declaring PK/FK relationships

The catalog is just data — one document per table in `haystak_catalog`, keyed by table
name. Columns come from the mapping, so a catalog document holds only `primary_key` and
`foreign_keys`:

```json
{ "table": "orders", "primary_key": ["id"],
  "foreign_keys": [
    { "columns": ["customer_id"], "ref_table": "customers", "ref_columns": ["id"] },
    { "columns": ["product_id"],  "ref_table": "products",  "ref_columns": ["id"] }
  ] }
```

After `--db` bootstraps a foreign database, its FK lists start empty. Declaring a
relationship is a single edit to the document (keyed by table name). To wire a bootstrapped
`events` index to `widgets`:

```sh
curl -XPOST 'http://localhost:9200/haystak_catalog/_update/events?refresh=true' \
  -H 'Content-Type: application/json' -d '{
  "doc": {
    "primary_key": ["id"],
    "foreign_keys": [
      { "columns": ["widget_id"], "ref_table": "widgets", "ref_columns": ["id"] }
    ]
  }
}'
```

`Describe` reads the catalog on every query, so the change is live with no restart:
`sys.foreign_keys` / `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS` now show the edge, and
`SELECT … FROM events e JOIN widgets w ON e.widget_id = w.id` resolves through it. DDL keeps
the two stores consistent: `CREATE TABLE` builds the index mapping and writes the catalog
document, `ALTER TABLE … ADD` extends the mapping, and `DROP TABLE` removes both.

## How it maps (`es/es.go`)

| SQL | Elasticsearch |
|---|---|
| table | index that has a `haystak_catalog` document |
| column | `_mapping` field (ES type → SQL type); `_id` stays hidden |
| primary / foreign keys | the `haystak_catalog` document for that table |
| `SELECT … WHERE/JOIN/…` | `Scan` returns the index's docs; the engine does the query |
| `INSERT`/`UPDATE`/`DELETE` | `index` / `_update_by_query` / `_delete_by_query` |
| `CREATE`/`ALTER`/`DROP TABLE` | create index + write catalog / `PUT _mapping` / delete index + catalog doc |

It's a thin backend (`Caps{Pushdown, Writable, DDL}`). ES has no native multi-database
concept, so it exposes a single logical database (the cluster) and does not implement
`Databaser`. An index without a catalog document is not a SQL table — the declared catalog
is authoritative for the table set.

## Separate Go module

Its own `go.mod` (requiring `github.com/elastic/go-elasticsearch/v9`, Apache-2.0) with a
`replace` to the parent keeps the core library dependency-free.

```sh
cd examples/elasticsearch-community-2 && go build ./... && go test ./...   # the test skips if ES is down
```

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
