English | [中文](tnet.zh_CN.md)

# Using high-performance networking framework tnet with tRPC-Go


## Introduction

The Golang Net library provides a simple non-blocking interface, with a network model that employs `Goroutine-per-Connection`. In most scenarios, this model is straightforward and user-friendly. However, when dealing with thousands or even millions of connections, allocating a goroutine for each connection consumes a significant amount of memory, and managing a large number of goroutines becomes challenging.

To support the capability of handling millions of connections, it is essential to break away from the `Goroutine-per-Connection` model. The high-performance networking library [tnet](https://github.com/trpc-group/tnet) is built on a `Reactor` network model, enabling the handling of millions of connections. The tRPC-Go framework integrates the tnet network library, thereby providing support for handling millions of connections. In addition to this, tnet also offers features such as batch packet transmission and reception, zero-copy buffering, and fine-grained memory management optimizations, making it outperform the native Golang Net library in terms of performance.

## Principle

We use two diagrams to illustrate the basic principles of the `Goroutine-per-Connection` model and the `Reactor` model in Golang.

### Goroutine-per-Connection

![goroutine_per_connection](/.resources-without-git-lfs/user_guide/tnet/goroutine_per_connection.png)

In the Goroutine-per-Connection model, when the server accepts a new connection, it creates a goroutine for that connection, and then reads data from the connection, processes the data, and sends data back to the connection in that goroutine.

The scenario of millions of connections usually refers to long connection scenarios. Although the total number of connections is huge, only a small number of connections are active at any given time. Active connections refer to connections that have data to be read or written at a certain moment. Conversely, when there is no data to be read or written on a connection, it is called an idle connection. The idle connection goroutine will block on the Read call. Although the goroutine does not occupy scheduling resources, it still occupies memory resources, which ultimately leads to huge memory consumption. In this model, allocating a goroutine for each connection in a scenario of millions of connections is expensive.

For example, as shown in the above diagram, the server accepts 5 connections and creates 5 goroutines. At this moment, the first 3 connections are active connections, and data can be read from them smoothly. After processing the data, the server sends data back to the connections to complete a data exchange, and then starts the second round of data reading. The last 2 connections are idle connections, and when reading data from them, the process will be blocked. Therefore, the subsequent process is not triggered. As we can see, although only 3 connections can successfully read data at this moment, 5 goroutines are allocated, resulting in a 40% waste of resources. The larger the proportion of idle connections, the more resources will be wasted.

### Reactor

![reactor](/.resources-without-git-lfs/user_guide/tnet/reactor.png)

The Reactor model refers to using multiplexing (epoll/kqueue) to listen for events such as read and write on file descriptors (FDs), and then performing corresponding operations when events are triggered.

In the diagram, the poller structure is responsible for listening to events on FDs. Each poller occupies a goroutine, and the number of pollers is usually equal to the number of CPUs. We use a separate poller to listen for read events on the listener port to accept new connections, and then listen for read events on each connection. When a connection becomes readable, a goroutine is allocated to read data from the connection, process the data, and send data back to the connection. At this point, there will be no idle connections occupying goroutines. In a scenario of millions of connections, only active connections are allocated goroutines, which can make full use of memory resources.

For example, as shown in the above diagram, the server has 5 pollers, one of which is responsible for listening to the listener events and accepting new connections, and the other 4 pollers are responsible for listening to read events on connections. When 2 connections become readable at a certain moment, a goroutine is allocated for each connection to read data, process data, and send data back to the connection. Because it is already known that these two connections are readable, the Read process will not block, and the subsequent process can be executed smoothly. When writing data back to the connection, the goroutine registers a write event with the poller, and then exits. The poller listens for write events on the connection and sends data when the connection is writable, completing a round of data exchange.

## Quick start

### Enable tnet

There are two ways to enable tnet in tRPC-Go. Choose one of them for configuration. It is recommended to use the first method.

(1) Add tnet in the tRPC-Go framework configuration file.

(2) Use the WithTransport() method in the code to enable tnet.

#### Method 1: Configuration file (recommended)

Add tnet to the transport field in the tRPC-Go configuration file. Since the plugin currently only supports TCP, please do not configure the tnet plugin for UDP services. The server and client can each independently activate tnet, and they do not interfere with each other.

**Server**:

```yaml
server:
  transport: tnet # Applies to all services
  service:
    - name: trpc.app.server.service
      network: tcp
      transport: tnet # Applies only to the current service
```

After the server is started, the log indicates the successful activation of tnet:

`INFO tnet/server_transport.go service:trpc.app.server.service is using tnet transport, current number of pollers: 1`

**Client**:

```yaml
client:
  transport: tnet # Applies to all services
  service:
    - name: trpc.app.server.service
      network: tcp
      transport: tnet # Applies only to the current service
      conn_type: multiplexed # Using multiplexed connection mode
      multiplexed:
        enable_metrics: true # Enable metrics for multiplexed pool
```

It's recommended to use multiplexed connection mode with tnet to enhance performance, because it can fully leverage tnet's batch packet transmission capabilities.

After the client is started, the log indicates the successful activation of tnet (Trace level):

`Debug tnet/client_transport.go roundtrip to:127.0.0.1:8000 is using tnet transport, current number of pollers: 1`

#### Method 2: Code configuration

**Server**:

Notics: This method will enable tnet for all services of the server.

```go
import "trpc.group/trpc-go/trpc-go/transport/tnet"

func main() {
    // Create a ServerTransport
    trans := tnet.NewServerTransport()
    // Create a trpc server
    s := trpc.NewServer(server.WithTransport(trans))
    pb.RegisterGreeterService(s, &greeterServiceImpl{})
    s.Serve()
}
```

**Client**:

```go
import "trpc.group/trpc-go/trpc-go/transport/tnet"

func main() {
    proxy := pb.NewGreeterClientProxy()
    trans := tnet.NewClientTransport()
    rsp, err := proxy.SayHello(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "Hello"}, client.WithTransport(trans))
}
```

## Use Cases

According to the benchmark result, tnet transport outperforms gonet transport in specific scenarios. However, not all scenarios exhibit these advantages. Here, we summarize the scenarios in which tnet transport excels.

**Advantageous Scenarios for tnet：**

- When using tnet in server, if the client sends requests using multiplexed connection mode, it can fully utilize tnet's batch packet transmission capabilities, leading to increased QPS and reduced CPU usage.

- When using tnet in server, if there are a large number of idle connections, it can reduce memory usage by lowering the number of goroutines.

- When using tnet in client, if the multiplexed mode is enabled, it can fully leverage tnet's batch packet transmission capabilities, resulting in improved QPS.

**Other Scenarios:**

- When using tnet in server, if the client sends requests not using multiplexed connection mode, performance is similar to gonet.

- When using tnet in client, if the multiplexed mode is disable, performance is similar to gonet.

## FAQ

#### Q：Does tnet support HTTP？

Tnet doesn't support HTTP. When tnet is used in HTTP server/client, it automatically falls back to using the golang net package.

#### Q：Why doesn't performance improve after enabling tnet?

Tnet is not a universal solution, and it can significantly boost service performance by fully utilizing Writev for batch packet transmission and reducing system calls in specific scenarios. If you find that the service performance is still not satisfactory in tnet's advantageous scenarios, you can consider optimizing your service using the following steps:

Enable the client-side multiplexed connection mode with tnet and make full use of Writev for batch packet transmission whenever possible;

Enable tnet and multiplexed connection mode for the entire service chain. If the upstream server utilizes multiplexed, the current server can also take advantage of Writev for batch packet transmission;

If you have enabled the multiplexed connection mode, you can enable metrics to inspect the number of virtual connections on each connection. If there is substantial concurrency, causing an excessive number of virtual connections on a single connection, it can also impact performance. Configure and enable multiplexed metrics accordingly.

```yaml
client:
  service:
    - name: trpc.test.helloworld.Greeter1
      transport: tnet
      conn_type: multiplexed
      multiplexed:
        enable_metrics: true # Enable metrics for multiplexed pool
```

Every 3 seconds, logs containing the multiplexed status are printed. For example, you can see that the current number of active connections is 1, and the total number of virtual connections is 98.

`DEBUG tnet multiplex status: network: tcp, address: 127.0.0.1:7002, connections number: 1, concurrent virtual connection number: 98`

It also reports status to custom metrics, and the format of the metrics items is as follows:

Active connections：`trpc.MuxConcurrentConnections.$network.$address`

Virtual connections：`trpc.MuxConcurrentVirConns.$network.$address`

Assuming you want to set the maximum concurrent virtual connections per connection to 25, you can add the following configuration:

```yaml
client:
  service:
    - name: trpc.test.helloworld.Greeter1
      transport: tnet
      conn_type: multiplexed
      multiplexed:
        enable_metrics: true
        max_vir_conns_per_conn: 25 # maximum number of concurrent virtual connections per connection
```

#### Q：Why does it log `switch to gonet default transport, tnet server transport doesn't support network type [udp]` after enabling tnet？

The log indicates tnet transport does't support UDP. It will automatically falls back to using golang net package.
