# FastHTTP

This example demonstrates the use of HTTP Standard Service in tRPC.

## Usage

* Start server.

```shell
go run server/main.go -conf server/trpc_go.yaml
```

* Curl request.

```sh
curl -X POST -d '{"msg":"hello"}' -H "Content-Type:application/json" "http://127.0.0.1:8000/trpc.test.helloworld.Greeter/SayHello"
```

The server log will be displayed as follows:

```log
2024-08-19 15:40:03.297 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=32: CPU quota undefined
2024-08-19 15:40:03.298 INFO    server/service.go:202   process: 106057, fasthttp_no_protocol service: trpc.app.server.fasthttp launch success, tcp: 127.0.0.1:8080, serving ...
```

The client log will be displayed as follows:

```log
2024-08-19 15:40:08.449 INFO    client/main.go:61       Msg is "Hello, fcp-post[POST]", response head is "response head"
2024-08-19 15:40:08.450 INFO    client/main.go:106      Msg is "Hello, fcp-get", response head is "response head"
2024-08-19 15:40:08.450 INFO    client/main.go:151      Msg is "Hello, fc-post[POST]", response head is "response head"
2024-08-19 15:40:08.450 INFO    client/main.go:131      Msg is "Hello, fc-get", response head is "response head"
```

## Explanation

For more Information, please refer to:

* [Building a Generic HTTP Standard Service with tRPC-Go](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)
