# Examples

Runnable backends for `haystak-tds-spi`. Each serves real data over the SQL Server (TDS) wire, so any
TDS client — `sqlcmd`, SSMS, Power BI, ODBC/JDBC, MCP SQL connectors — can query it. Every directory
has its own README with run steps.

## Start here

- [`inmem`](inmem) — the simplest possible backend, about one file. Read it first to learn the SPI.
- [`gateway`](gateway) — the minimal "how do I serve a backend on the wire" program, over `inmem`.

These two are part of the root module (no external dependencies).

## Real backends

Each real backend ships in a few **catalog flavors** — how the SQL schema (tables, columns, primary
and foreign keys) is determined:

| Backend | Inferred (sampled from data) | Declared (explicit schema) | Hardcoded (Go literal) |
|---------|------------------------------|----------------------------|------------------------|
| Elasticsearch | [`elasticsearch-community`](elasticsearch-community) | [`elasticsearch-community-2`](elasticsearch-community-2) | — |
| OpenSearch | [`opensearch-community`](opensearch-community) | [`opensearch-community-2`](opensearch-community-2) | — |
| MongoDB | [`mongodb-community`](mongodb-community) | [`mongodb-community-2`](mongodb-community-2) | [`mongodb-community-3`](mongodb-community-3) |

Each real backend is its own Go module, so its driver dependency stays out of the core library.
