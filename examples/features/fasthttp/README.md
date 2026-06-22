# FastHTTP

This example demonstrates a standard HTTP service implemented with FastHTTP in
tRPC-Go.

## Usage

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Run client.

```shell
$ go run client/main.go
```

* Or send a curl request.

```shell
$ curl -X POST -H "hello: curl" http://127.0.0.1:8080/v1/hello
```

The server replies with the request path, the `hello` header, and the HTTP
method used by the request.

## Explanation

This example uses `fasthttp_no_protocol` on the server side. The client shows
two calling styles:

* `NewFastHTTPClientProxy`, which keeps tRPC client options such as target
  routing and serialization options.
* `NewFastHTTPClient`, which exposes a direct FastHTTP-style client.

For more information, see [FastHTTP transport](/http/README.md#fasthttp-transport).
