## Broadcast

This example demonstrates the use of broadcast in tRPC.

## Usage

* Start the client

```shell
go run client/main.go
```

The server log will be displayed as follows:

```log
2024-09-23 19:08:57.583 DEBUG   client/main.go:64       broadcast rpc receive from node 127.0.0.1:8080, with: msg:"trpc-go-client"
2024-09-23 19:08:57.583 DEBUG   client/main.go:64       broadcast rpc receive from node 127.0.0.1:8081, with: msg:"trpc-go-client"
2024-09-23 19:08:57.583 DEBUG   client/main.go:64       broadcast rpc receive from node 127.0.0.1:8082, with: msg:"trpc-go-client"
```

## Explanation

This example implements a custom discovery and a serviceRouter that supports broadcast calls. In actual use, please use them of naming-polaris. Additionally, custom transport is implemented to simulate server responses.

Based on the service name 'trpc.examples.broadcast.example', the broadcast call will be made to the three nodes '127.0.0.1:8000', '127.0.0.1:8001', and '127.0.0.1:8002', and all are expected to receive the anticipated responses like "trpc-go-client".
