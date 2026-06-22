# Prewarm

This example demonstrates how to explicitly initialize a client and prewarm its
connection pool before sending requests.

## Usage

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

```shell
$ go run client/main.go
```

The client initializes `client.DefaultClient` with `client.WithPreWarm` before
creating the generated proxy. If initialization succeeds, the connection pool
has already established the configured number of connections for the target
node, and the following RPC can reuse the warmed pool.

## Explanation

Prewarming is opt-in. Set `transport.PreWarmOptions.ConnsPerNode` to a positive
value to enable it. `Timeout` bounds the initialization phase.

This example only uses public APIs:

* `client.InitializableClient`
* `client.WithPreWarm`
* `transport.PreWarmOptions`

It does not inspect internal pool metrics or connection counters.
