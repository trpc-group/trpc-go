# tnet udp transport

This example demonstrates the use of tnet udp transport in tRPC.

## Usage

### Normal

* Start server.

```shell
go run normal/server/main.go -conf normal/server/trpc_go.yaml
```

* Start client.

```shell
go run normal/client/main.go -conf normal/client/trpc_go.yaml
```

tnet UDP can be enabled through code or configuration files, similar to the usage of tnet TCP.

Code Example:

```go
// server option
server.WithTransport(transport.GetServerTransport("tnet")), 
server.WithNetwork("udp")

// client option
client.WithTransport(transport.GetClientTransport("tnet"))
client.WithNetwork("udp")
```

Configuration File:

```yaml
server:                                            # server configuration.
  service:                                         # business service configuration，can have multiple.
    - name: trpc.test.helloworld.Greeter           # the route name of the service.
      ip: 127.0.0.1                                # the service listening ip address, can use the placeholder ${ip}, choose one of ip and nic, priority ip.
      port: 8000                                   # the service listening port, can use the placeholder ${port}.
      transport: tnet                              # transport type for this service, default empty.
      network: udp                                 # the service listening network type,  tcp or udp.

client:                                            # configuration for client calls.
  service:                                         # configuration for a single backend.
    - name: trpc.test.helloworld.Greeter           # backend service name.
      transport: tnet                              # transport type for this service, default empty.
      network: udp                                 # backend service network type, tcp or udp, configuration takes precedence.
      target: ip://127.0.0.1:8000                  # service addr
```

### Exact buffer size

* Start server.

```shell
go run exactbuffersize/server/main.go -conf exactbuffersize/server/trpc_go.yaml
```

* Start client.

```shell
go run exactbuffersize/client/main.go -conf exactbuffersize/client/trpc_go.yaml
```

The options `WithServerExactUDPBufferSizeEnabled` and `WithClientExactUDPBufferSizeEnabled` control whether to allocate an exact-sized buffer for UDP packet. By default, this setting is false.

* True: Allocates an exact-sized buffer for each UDP packet. This approach requires two system calls per packet but ensures that the buffer is optimally sized for the data being received. Using exact buffer sizes can be beneficial in environments where memory usage is critical, or where the packet size varies significantly, allowing for more precise control over resource allocation.
* False: Uses a fixed buffer size of maxUDPPacketSize, which is 65536 by default. This method requires only one system call and may be more efficient in scenarios where packet size is consistently near the maximum.
