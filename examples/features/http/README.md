# HTTP

This example demonstrates the use of HTTP Standard Service in tRPC.

## Usage

* Start server.

```shell
go run server/main.go -conf server/trpc_go.yaml
```

* Curl request.

```shell
curl -X POST -d '{"msg":"hello"}' -H "Content-Type:application/json" "http://127.0.0.1:8080/v1/hello"
```

The server log will be displayed as follows:

```shell
2024-08-22 11:51:36.172 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=10: CPU quota undefined
2024-08-22 11:51:36.172 INFO    server/service.go:202   process: 131426, http_no_protocol service: trpc.app.server.stdhttp launch success, tcp: 127.0.0.1:8080, serving ...
```