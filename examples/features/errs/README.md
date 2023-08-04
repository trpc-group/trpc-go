## Errs

This example demonstrates the use of errors in tRPC.

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
2023-06-13 15:40:30.546 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=10: CPU quota undefined
2023-06-13 15:40:30.547 INFO    server/service.go:164   process:97184, trpc service:trpc.examples.errs.Errs launch success, tcp:127.0.0.1:8000, serving ...
2023-06-13 15:40:46.247 DEBUG   server/service.go:245   service: trpc.examples.errs.Errs handle err (if caused by health checking, this error can be ignored): type:business, code:10001, msg:request missing required field: Msg
```

The client log will be displayed as follows:
```
2023/06/13 15:40:46 Calling SayHello with Name:"trpc-go-client"
2023/06/13 15:40:46 Received response: Hello trpc-go-client
2023/06/13 15:40:46 Calling SayHello with Name:""
2023/06/13 15:40:46 Received error: type:business, code:10001, msg:request missing required field: Msg
2023/06/13 15:40:46 Received error: type:framework, code:121, msg:client codec Marshal: proto: Marshal called with nil
```

