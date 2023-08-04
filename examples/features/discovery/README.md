## Discovery

This example demonstrates the use of service discovery in tRPC.

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
2023-06-19 11:16:38.786 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=10: CPU quota undefined
2023-06-19 11:16:38.787 INFO    server/service.go:164   process:50798, trpc service:trpc.examples.discovery.Discovery launch success, tcp:127.0.0.1:8000, serving ...
2023/06/19 11:17:03 Received msg from client : trpc-go-client 3
2023/06/19 11:17:03 Received msg from client : trpc-go-client 7
2023/06/19 11:17:03 Received msg from client : trpc-go-client 8
```

The client log will be displayed as follows:
```
2023/06/19 11:17:03 Received error from client 0: type:framework, code:111, msg:tcp client transport dial, cost:809.166µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 11:17:03 Received error from client 1: type:framework, code:111, msg:tcp client transport dial, cost:132.917µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 11:17:03 Received error from client 2: type:framework, code:111, msg:tcp client transport dial, cost:143.5µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 11:17:03 Received error from client 4: type:framework, code:111, msg:tcp client transport dial, cost:143.334µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 11:17:03 Received error from client 5: type:framework, code:111, msg:tcp client transport dial, cost:115.542µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 11:17:03 Received error from client 6: type:framework, code:111, msg:tcp client transport dial, cost:114.083µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 11:17:03 Received error from client 9: type:framework, code:111, msg:tcp client transport dial, cost:110.292µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
```

## Explanation

This example implemented a custom service discovery strategy. Each time it returns three nodes: "127.0.0.1:8000", "127.0.0.1:8001", and "127.0.0.1:8002".

By default, tRPC uses random load balancing, which means that it will randomly select one of the three nodes above for the request. Since the server only provides services at the address "127.0.0.1:8000", only requests to port 8000 will be successful, and requests to ports 8001 and 8002 will both fail.
                   		