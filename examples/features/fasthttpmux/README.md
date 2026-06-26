# FastHTTP

This example demonstrates the use of HTTP Standard Service in tRPC.

## Usage

* Start server.

```shell
cd server
clear && go run main.go
```

* Run client.

```sh
cd client
clear && go run main.go
```

The server log will be displayed as follows:

```log
2024-09-26 11:45:35.895 DEBUG   maxprocs/maxprocs.go:48 maxprocs: Leaving GOMAXPROCS=32: CPU quota undefined
2024-09-26 11:45:35.895 INFO    server/service.go:202   process: 1854269, fasthttp_no_protocol service: trpc.app.server.fasthttp launch success, tcp: 127.0.0.1:8080, serving ...
```

The client log will be displayed as follows:

```log
2024-09-26 11:54:02.135 INFO    client/main.go:60       Msg is "/v1/hello, fcp-post[POST]", response head is "response head"
2024-09-26 11:54:02.136 INFO    client/main.go:74       Msg is "/v2/hello, fcp-post[POST]", response head is "response head"
2024-09-26 11:54:02.136 INFO    client/main.go:117      Msg is "/v1/hello, fcp-get", response head is "response head"
2024-09-26 11:54:02.136 INFO    client/main.go:131      Msg is "/v2/hello, fcp-get", response head is "response head"
2024-09-26 11:54:02.136 INFO    client/main.go:190      Msg is "/v1/hello, fc-post[POST]", response head is "response head"
2024-09-26 11:54:02.136 INFO    client/main.go:203      Msg is "/v2/hello, fc-post[POST]", response head is "response head"
2024-09-26 11:54:02.136 INFO    client/main.go:157      Msg is "/v1/hello, fc-get", response head is "response head"
2024-09-26 11:54:02.137 INFO    client/main.go:170      Msg is "/v2/hello, fc-get", response head is "response head"
```

no routing:

```bash
curl http://127.0.0.1:8080/123
no routing
```

## Explanation

For more Information, please refer to:

* [Building a Generic HTTP Standard Service with tRPC-Go](/http/README.md#pan-http-standard-services)
