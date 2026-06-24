# Server-Sent Events

This example demonstrates how to consume server-sent events with the tRPC-Go
HTTP client codec and how to write SSE events with `http.WriteSSE`.

## Usage

* Start server.

```shell
$ go run server/main.go
```

* Start client.

```shell
$ go run client/main.go
```

The client sets `http.ClientRspHeader.SSEHandler`. When the response is an SSE
stream, the framework parses each event and invokes the handler.

## Explanation

The server is a small standard-library HTTP server so the example stays focused
on the public SSE APIs:

* `http.WriteSSE`
* `http.ClientRspHeader.SSEHandler`
* `http.ClientRspHeader.SSECondition`

No additional transport plugin or framework extension is required.
