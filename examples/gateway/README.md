# examples/gateway

A runnable TDS gateway over the [`inmem`](../inmem) backend, the minimal "how do I
serve a backend on the wire" example.

## Run it

```sh
go run ./examples/gateway 127.0.0.1:1433     # plaintext (default addr if omitted)
HAYSTAK_TLS=1 go run ./examples/gateway       # self-signed TLS (TLS-in-TDS)
```

Then connect with any TDS client:

```sh
sqlcmd -S 127.0.0.1,1433 -U sa -P x -C -Q "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id"
```

`-C` trusts the self-signed cert. The gateway ignores the username and password here;
auth is the backend's concern, not the demo's.

## How it works

The entire wiring is:

```go
gw := &server.Server{Backend: inmem.New(), Logf: log.Printf}
// optionally gw.TLSConfig = <a *tls.Config>
log.Fatal(gw.ListenAndServe(addr))
```

`server.Server` runs the TDS handshake (PRELOGIN/LOGIN7, optional TLS), decodes
`SQL_BATCH` and RPC `sp_executesql`, drives the engine against the backend, and streams
the result tokens back. The `selfSignedTLS()` helper here just mints an ephemeral ECDSA
cert so you can try TLS without setup; in production you'd supply your own `*tls.Config`.

Swap `inmem.New()` for any `tds.Backend` (for example [`mongodb-community`](../mongodb-community))
and you have a gateway for that store.

## Authentication (demo)

By default the gateway is anonymous: any `-U/-P` works. Set `HAYSTAK_AUTH` to require
credentials, which wires up `server.StaticAuth` (operator-level auth):

```sh
HAYSTAK_AUTH="sa:secret,reader:ro" go run ./examples/gateway
sqlcmd -S 127.0.0.1,1433 -U sa -P wrong  -C -Q "SELECT 1"   # → Login failed for user 'sa'
sqlcmd -S 127.0.0.1,1433 -U sa -P secret -C -Q "SELECT name FROM users"
```

In a real backend you'd instead have the backend implement `tds.Authenticator` so it
owns the go/no-go and the per-user identity (see the root README's Authentication
section).

## License

Apache-2.0 — see [LICENSE](../../LICENSE).
