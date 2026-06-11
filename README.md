# haystak-tds-spi

[![Go Reference](https://pkg.go.dev/badge/github.com/RSKGroup/haystak-tds-spi.svg)](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi)

A SQL Server wire-protocol (TDS) gateway, shipped as importable Go modules. It speaks
TDS (the protocol SQL Server clients talk) on the front, and translates through a
pluggable backend SPI to whatever store sits behind it. Any TDS client (sqlcmd, SSMS,
Power BI, the go-mssqldb/.NET/JDBC drivers, MCP SQL tools) can connect, browse the
catalog, and run real SQL against your backend: rich reads plus INSERT/UPDATE/DELETE
and CREATE/DROP TABLE.

Pure Go, no CGO, permissive dependencies only.

Status: read and write are complete and live-validated over the wire (plaintext and
TLS). That covers the full read query surface plus `INSERT`/`UPDATE`/`DELETE` and
`CREATE`/`DROP TABLE`/`DATABASE`, routed to the backend's write interfaces and
fail-closed when unsupported. Out of the box, the `examples/` directory gives you a
working MSSQL/TDS interface to **MongoDB, Elasticsearch, and OpenSearch right now** — run
one, point `sqlcmd`/SSMS/Power BI at it, and query a NoSQL store as if it were SQL Server.

## Ready-to-run backends

These examples are complete, live-validated gateways — `go run` one and a real document
store answers T-SQL on `:1433` (joins, aggregates, `INSERT`/`UPDATE`/`DELETE`,
`INFORMATION_SCHEMA`/`sys.*` and all). Each takes an optional `--host <url:port>` (default
localhost) and `--db <name>` (the database/index scope; blank seeds demo data so it works
immediately):

| Store | Inferred catalog (zero config) | Declared catalog (real FKs) |
|---|---|---|
| MongoDB | [`mongodb-community`](examples/mongodb-community) — fields sampled from documents | [`mongodb-community-2`](examples/mongodb-community-2) — `__haystak_catalog` collection |
| Elasticsearch | [`elasticsearch-community`](examples/elasticsearch-community) — fields sampled from `_source` | [`elasticsearch-community-2`](examples/elasticsearch-community-2) — columns from `_mapping`, keys from `haystak_catalog` |
| OpenSearch | [`opensearch-community`](examples/opensearch-community) — fields sampled from `_source` | [`opensearch-community-2`](examples/opensearch-community-2) — columns from `_mapping`, keys from `haystak_catalog` |

```sh
go run ./examples/elasticsearch-community               # seeds + serves a local ES on :1433
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id"
```

The *inferred* variants need no schema — point them at a store and they sample documents to
build the SQL catalog. The *declared* variants add the one thing inference can't recover,
foreign keys, so cross-table `JOIN`s and `sys.foreign_keys` light up. (`examples/inmem` and
`examples/gateway` are the dependency-free reference backends.)

## Install

```sh
go get github.com/RSKGroup/haystak-tds-spi
```

Implement `tds.Backend`, then serve it with `server.ListenAndServe`. See
[Build a backend](#build-a-backend) below, the runnable examples on
[pkg.go.dev](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi/tds#example-package),
and the complete backends in [`examples/`](examples/) to copy from.

## Architecture

```text
TDS client ──wire──> server ──> engine ──> your Backend (SPI)
              │         │          │            implements tds.Backend (+ Scanner/QueryExecutor,
        PRELOGIN/LOGIN7 dispatch   parse T-SQL  optionally Writer/DDL/Databaser/…)
        TLS, SQL_BATCH, RPC        + evaluate
```

- `server` listens, runs the TDS handshake (including TLS-in-TDS), decodes `SQL_BATCH`
  and RPC `sp_executesql`, and streams `COLMETADATA`/`ROW`/`DONE` back.
- `internal/{tsql,engine,exec}` parse a T-SQL read subset, route by capability, and
  evaluate it (filter/join/aggregate/set-ops/subqueries/CTEs/expressions) for thin
  backends.
- `internal/{infoschema,sysviews}` answer `INFORMATION_SCHEMA.*` and `sys.*` from your
  declared catalog (tables, columns, types, foreign keys).
- `tds` is the SPI you implement.

## Run the demo gateway

```sh
go run ./examples/gateway 127.0.0.1:1433        # plaintext
HAYSTAK_TLS=1 go run ./examples/gateway          # self-signed TLS
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT name FROM users"
```

## Build a backend

Implement `tds.Backend` plus one query path, and advertise it via `Caps`:

```go
type Backend interface {
    Describe(ctx) (catalog.Schema, error) // your tables/columns/PKs/FKs
    Capabilities() Caps                   // what you support
}
```

The easiest path is a thin backend: implement `Scanner` (`Scan` returns whole tables)
and set `Caps{Pushdown:true}`. The core engine then does all the WHERE/JOIN/GROUP
BY/ORDER BY/paging/subqueries/CTEs work for you. For a thick backend, implement
`QueryExecutor` (`ExecuteQuery` handles a whole logical query) and set
`Caps{FullQuery:true}` to push down to your store's native query language.

The rest is optional, advertised via `Caps` and detected by interface assertion:

| Interface | For | Caps |
|---|---|---|
| `Writer` | INSERT/UPDATE/DELETE | `Writable` |
| `DDL` | CREATE/ALTER/DROP TABLE | `DDL` |
| `DatabaseDDL` | CREATE/DROP DATABASE | `DDL` |
| `Databaser` | more than one database (→ `sys.databases`, `Query.Database`) | — |
| `TxBeginner`/`Tx` | transactions | `Tx` |
| `Authenticator` | authenticate TDS logins (go/no-go) + identity | — |

### Authentication

The backend owns authentication. Implement `tds.Authenticator` and the gateway hands
you each LOGIN7 credential set (`tds.Login{Username, Password, Database, …}`); return a
`tds.Principal` to allow the login, or an error to reject it (a real `18456`
login-failed reaches the client). The authenticated `Principal` then rides in `ctx` on
every `Scan`/`Insert`/… call, so you authorize and audit per user with the same
identity. Operators who'd rather not push auth into the backend can set
`server.Server.Auth` (for example `server.StaticAuth(map)`) for gateway-level auth
instead; with neither set, connections are anonymous, which is the demos' default.

Then run it:

```go
server.ListenAndServe("127.0.0.1:1433", myBackend)
// or &server.Server{Backend: myBackend, TLSConfig: cfg, ServerName: "...", Database: "..."}
```

`examples/inmem` is a complete thin reference backend.

## Validate your backend

```go
func TestConformance(t *testing.T) { tdstest.RunConformance(t, myBackend.New()) }
```

`tds/tdstest.RunConformance` checks that `Caps` and the implemented interfaces agree,
then drives real `SELECT` and catalog queries through the engine.

## Read surface

Single- and multi-table queries, evaluated entirely over the wire.

Query shape:

- `SELECT` with projection, `DISTINCT`, `TOP n [PERCENT]`, and aliases (`AS` or `alias = expr`)
- `FROM` with table aliases and qualified columns
- JOINs: INNER, LEFT, RIGHT, FULL, CROSS
- `WHERE`, `GROUP BY`, `HAVING`, `ORDER BY` (by column or ordinal, `ASC`/`DESC`)
- `OFFSET … FETCH` paging
- `UNION`, `UNION ALL`, `INTERSECT`, `EXCEPT`
- Subqueries: `IN (…)`, `EXISTS`, scalar `= (…)`, derived tables `FROM (SELECT …) t`, and correlated forms
- CTEs (`WITH`), including recursive
- No-`FROM` scalar `SELECT`, `;`-separated multi-statement batches, and RPC `sp_executesql`

Predicates and operators:

- Comparison `= <> < > <= >=`; `AND`, `OR`, `NOT`, parentheses
- `IN` (value list or subquery), `BETWEEN`, `LIKE`, `IS NULL`, `IS NOT NULL`, `EXISTS`
- Arithmetic `+ - * / %` and string concatenation, in both `SELECT` and `WHERE`

Functions:

- Aggregate: `COUNT`, `SUM`, `AVG`, `MIN`, `MAX`
- String: `LEN`/`DATALEN`, `UPPER`, `LOWER`, `LTRIM`, `RTRIM`, `TRIM`, `SUBSTRING`, `REPLACE`, `CONCAT`
- Conditional: `ISNULL`, `COALESCE`, `NULLIF`, `CASE` (simple and searched)
- Numeric: `ABS`
- Date: `YEAR`, `MONTH`, `DAY`, `GETDATE`, `GETUTCDATE`, `SYSDATETIME`, `SYSUTCDATETIME`
- `CAST`/`CONVERT` to int/bigint/smallint/tinyint, float/real/decimal/numeric/money, bit,
  char/varchar/nchar/nvarchar/text, plus the typed wire encodings (date, datetime,
  uniqueidentifier, varbinary)

Catalog:

- `INFORMATION_SCHEMA`: `TABLES`, `COLUMNS`, `TABLE_CONSTRAINTS`, `KEY_COLUMN_USAGE`, `REFERENTIAL_CONSTRAINTS`
- `sys.*`: `databases`, `schemas`, `tables`, `columns`, `types`, `objects`, `foreign_keys`

Connect-time probes, so real drivers and BI tools finish their handshake:

- `@@VERSION`, `@@SPID`, `@@SERVERNAME`, `@@LANGUAGE`, `@@ROWCOUNT`, `@@ERROR`, `@@TRANCOUNT`, `@@FETCH_STATUS`
- `DB_NAME()`, `SCHEMA_NAME()`, `SYSTEM_USER`, `CURRENT_USER`, `SESSION_USER`, `USER_NAME()`, `SUSER_SNAME()`, `HOST_NAME()`, `APP_NAME()`
- `SERVERPROPERTY(…)`, `DATABASEPROPERTYEX(…)`; `SET …` and `USE …` accepted as no-ops

## Write surface

Write statements are parsed over the wire and dispatched to the backend's `Writer` /
`DDL` / `DatabaseDDL` interfaces, each fail-closed: if the backend doesn't implement the
interface, the client gets a clean error rather than a silent success.

- `INSERT INTO t [(cols)] VALUES (…)[, (…) …]` — single or multi-row
- `UPDATE t SET col = val [, …] [WHERE col op val [AND …]]`
- `DELETE FROM t [WHERE col op val [AND …]]`
- `CREATE TABLE t (col type [, …])` — types map to the SPI kinds: int/bigint/smallint/tinyint,
  bit, float/real, decimal/numeric/money, date/datetime/time, uniqueidentifier, varbinary,
  anything else → varchar
- `ALTER TABLE t ADD col type [, …]` and `ALTER TABLE t DROP COLUMN col [, …]`
- `DROP TABLE t`
- `CREATE DATABASE name`, `DROP DATABASE name`

The write `WHERE` is the simple form (`col op value`, joined by `AND`), and `VALUES`/`SET`
take literals; the rich expression tree from the read path doesn't apply here.

## docs/

Pinned protocol specs: MS-TDS (the wire), MC-SQLR (instance resolution), and MS-BINXML
(the `xml` type). See [docs/README.md](docs/README.md) for provenance and the licensing
notice.

## Documentation

Full API reference and runnable examples are on
[pkg.go.dev](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi):

- [`tds`](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi/tds) — the SPI: `Backend`, `Scanner`/`QueryExecutor`, `Writer`/`DDL`/`DatabaseDDL`/`Databaser`, `Authenticator`, `Caps`.
- [`server`](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi/server) — `ListenAndServe`, `Server`, `StaticAuth`.
- [`tds/catalog`](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi/tds/catalog) — the schema model (`Schema`/`Table`/`Column`/`ForeignKey`).
- [`tds/types`](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi/tds/types) — the backend-neutral type system.
- [`tds/tdstest`](https://pkg.go.dev/github.com/RSKGroup/haystak-tds-spi/tds/tdstest) — the `RunConformance` harness.

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE). "Haystak" is a registered trademark of
RSKGroup, LLC; the license grants no rights to the name (Apache-2.0 §6).
