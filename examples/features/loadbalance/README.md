## Loadbalance

This example demonstrates the use of load balancing in tRPC.

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
2023-06-19 14:17:14.077 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=10: CPU quota undefined
2023-06-19 14:17:14.078 INFO    server/service.go:164   process:87066, trpc service:trpc.examples.loadbalance.Loadbalance launch success, tcp:127.0.0.1:8000, serving ...
2023/06/19 14:17:18 Received msg from client : trpc-go-client 0
2023/06/19 14:17:18 Received msg from client : trpc-go-client 3
2023/06/19 14:17:18 Received msg from client : trpc-go-client 6
2023/06/19 14:17:18 Received msg from client : trpc-go-client 9
2023/06/19 14:17:18 Received msg from client : trpc-go-client 3
2023/06/19 14:17:18 Received msg from client : trpc-go-client 7
2023/06/19 14:17:18 Received msg from client : trpc-go-client 0
2023/06/19 14:17:18 Received msg from client : trpc-go-client 3
2023/06/19 14:17:18 Received msg from client : trpc-go-client 6
2023/06/19 14:17:18 Received msg from client : trpc-go-client 9
```

The client log will be displayed as follows:
```
Test Loadbalance with round_robin:
2023/06/19 14:17:18 Received error from client 1: type:framework, code:111, msg:tcp client transport dial, cost:152.583µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 2: type:framework, code:111, msg:tcp client transport dial, cost:123.375µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 4: type:framework, code:111, msg:tcp client transport dial, cost:111.167µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 5: type:framework, code:111, msg:tcp client transport dial, cost:124.375µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 7: type:framework, code:111, msg:tcp client transport dial, cost:125.458µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 8: type:framework, code:111, msg:tcp client transport dial, cost:108.042µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
Test Loadbalance with random:
2023/06/19 14:17:18 Received error from client 0: type:framework, code:111, msg:tcp client transport dial, cost:129.916µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 1: type:framework, code:111, msg:tcp client transport dial, cost:101.792µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 2: type:framework, code:111, msg:tcp client transport dial, cost:100.959µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 4: type:framework, code:111, msg:tcp client transport dial, cost:111.5µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 5: type:framework, code:111, msg:tcp client transport dial, cost:117.5µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 6: type:framework, code:111, msg:tcp client transport dial, cost:155.334µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 8: type:framework, code:111, msg:tcp client transport dial, cost:121.25µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 9: type:framework, code:111, msg:tcp client transport dial, cost:114.916µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
Test Loadbalance with weight_round_robin:
2023/06/19 14:17:18 Received error from client 1: type:framework, code:111, msg:tcp client transport dial, cost:111.875µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 2: type:framework, code:111, msg:tcp client transport dial, cost:125.167µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 4: type:framework, code:111, msg:tcp client transport dial, cost:118.208µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 5: type:framework, code:111, msg:tcp client transport dial, cost:116.375µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
2023/06/19 14:17:18 Received error from client 7: type:framework, code:111, msg:tcp client transport dial, cost:111.75µs, caused by dial tcp 127.0.0.1:8001: connect: connection refused
2023/06/19 14:17:18 Received error from client 8: type:framework, code:111, msg:tcp client transport dial, cost:127.292µs, caused by dial tcp 127.0.0.1:8002: connect: connection refused
```

## Explanation

When the service discovery returns a list of server addresses instead of a single address, it is necessary to use a load balancing algorithm to determine which address to communicate with the backend. This entire process is called load balancing, and tRPC-go currently uses client-side load balancing.

The tRPC-Go load balancing strategy defaults to a random strategy, Users can customize the load balancing algorithm. The algorithms provided by the framework include:

- random
- round robin
- weight round robin
- consistent hash

This demo uses a custom service discovery strategy, which can be referred to at [Discovery](../discovery/README.md). In this example, testLB uses an assigned strategy with parameter `balancerName`.

There are two points to note:

- The service is addressed through the serviceName and target cannot be set. If target is set through client.WithTarget, tRPC-go will default to using the target for service discovery and load balancing. For example, if the target is set to "ip://127.0.0.1:8000", this will directly take the IP addressing strategy.
- It is necessary to import the corresponding load balancing strategy package, otherwise a "loadbalance not exists" error will be reported.



