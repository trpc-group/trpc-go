## Cancellation

Cancellation demonstrates the different messages received by the client during normal requests and context canceled requests. 

## Usage

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

```shell
$ go run client/main.go
```

* Server output

```
2023-05-22 16:43:33.979 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined
2023-05-22 16:43:33.979 INFO    server/service.go:164   process:17610, trpc service:trpc.test.helloworld.Greeter launch success, tcp:127.0.0.1:8000, serving ...
2023-05-22 16:43:42.470 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-22 16:43:42.470 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

* Client output

```
2023-05-22 16:37:26.681 INFO    client/main.go:27       SayHello success rsp[msg:"Hello Hi trpc-go-client"]
2023-05-22 16:37:26.681 ERROR   client/main.go:35       canceled SayHello err[type:framework, code:161, msg:selector canceled after Select: context canceled] req[msg:"trpc-go-client"]
```