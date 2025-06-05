# HTTP

This example demonstrates the use of HTTP SSE(Server-Sent Events) Service in tRPC.

## Usage

### Normal, based on tRPC-Go

Normal SSE case using tRPC-Go.

* Start server.

```shell
go run normal/server/main.go -conf normal/server/trpc_go.yaml
```

* Start client.

Implement the client in `client/main.go` in your favorite mode, manually or not.
And then run the client.

```shell
go run normal/client/main.go
```

The server log will be displayed as follows:

```log
2024-07-23 14:56:46.113 INFO    server/service.go:202   process:28827, http_no_protocol service:trpc.app.server.ServiceSSE launch success, tcp:127.0.0.1:8080, serving ...
2024/07/23 14:56:56 http: superfluous response.WriteHeader call from git.code.oa.com/trpc-go/trpc-go/http.init.func2 (codec.go:659)
2024/07/23 14:59:47 http: superfluous response.WriteHeader call from git.code.oa.com/trpc-go/trpc-go/http.init.func2 (codec.go:659)
2024/07/23 15:00:42 http: superfluous response.WriteHeader call from git.code.oa.com/trpc-go/trpc-go/http.init.func2 (codec.go:659)
```

As for the client, you will see the two kinds of output.
If `ManualReadBody` is set to `true`, you should read the body from `rspHead.Response.Body` manually.
The body will contain the whole message:

```log
Received message: 
event: message
data: hello0


Received message: 
event: message
data: hello1


Received message: 
event: message
data: hello2
```

On the other hand, if `ManualReadBody` is set to `false` and your own `SSEHandler` is defined,
the body will be read automatically into the `sse.Event` struct, and the output will be:

```log
Processing event: message, data: hello0
Processing event: message, data: hello1
Processing event: message, data: hello2
Received data: hello0hello1hello2
```

### Multiple, based on tRPC-Go

Multiple SSE case using tRPC-Go, mainly for APIs that might return SSE and non-SSE responses.

* Start server.

```shell
go run multiple/server/main.go -conf multiple/server/trpc_go.yaml
```

* Start client.

Implement the client in `client/main.go` in auto mode, and the interfaces such as `ResponseHandler`, `SSEHandler`, etc.
And then run the client.

```shell
go run multiple/client/main.go
```

* Start proxy.

We also provide an example for proxy the SSE and non-SSE responses.

```shell
go run multiple/proxy/main.go
```

### HunYuan Model

Since that SSE is widely used to implement real-time communication, as for the HunYuan Model, there is an example in
`hunyuan/client.go` about [HunYuan App Create](https://iwiki.woa.com/p/4008515885#AppCreate).

### Complex, based on R3Lab/SSE

> Pay attention: This example is based on [r3Labs/sse](https://github.com/r3Labs/sse).
> It does not support custom `http.Client` and only supports `http.MethodGet`.
> Since the lack of the more custom features, it is not recommended to use it.

* Start server.

```shell
go run r3labs/server/main.go
```

* Start client.

```shell
go run r3labs/client/main.go
```

## Explanation

For more Information, please refer to:

* [Building a Generic HTTP Standard Service with tRPC-Go](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)
* [HTML standard](https://html.spec.whatwg.org/multipage/server-sent-events.html#server-sent-events)
* [Server-sent_events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
* [混元助手太极一站式平台](https://iwiki.woa.com/space/HunyuanaideTaiij)
