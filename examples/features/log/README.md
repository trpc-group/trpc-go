## Log

This example demonstrates the use of logs in tRPC.

## Usage

* Start the server
```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start the client
```shell
$ go run client/main.go
```

The server log will be displayed as follows:
```
2023-06-11 19:09:19.154 WARN    server/main.go:25       recv msg:msg:"Hello"
2023-06-11 19:09:19.154 ERROR   server/main.go:26       recv msg:msg:"Hello"
```

* The file log will be displayed as follows:
```
{"L":"DEBUG","T":"2023-06-11 19:09:19.154","C":"server/main.go:23","M":"recv msg:msg:\"Hello\""}
{"L":"INFO","T":"2023-06-11 19:09:19.154","C":"server/main.go:24","M":"recv msg:msg:\"Hello\""}
{"L":"WARN","T":"2023-06-11 19:09:19.154","C":"server/main.go:25","M":"recv msg:msg:\"Hello\""}
{"L":"ERROR","T":"2023-06-11 19:09:19.154","C":"server/main.go:26","M":"recv msg:msg:\"Hello\""}
```

The client log will be displayed as follows:
```
2023/06/11 19:09:19 Received response: trpc-go-server response: Hello
```

## Explanation

tRPC-Go uses the zap package from Uber by default to implement logging, which supports outputting to multiple endpoints at once and supports dynamically changing log levels at runtime. [uber-go/zap](https://github.com/uber-go/zap)

Configuring logging is implemented using a plugin style, which can be found at: [plugin](examples/features/plugin)




