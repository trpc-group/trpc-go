[TOC]

tRPC-Go access to high-performance network library



# Introduction

The Net library of Golang provides a simple non-blocking call interface, and its network model adopts a **Goroutine-per-Connection** approach. In most scenarios, this model is simple and easy to use. However, when the number of connections reaches thousands or even millions, allocating a Goroutine for each connection will consume a huge amount of memory, and scheduling a large number of Goroutines will become very difficult. To support millions of connections, it is necessary to break the Goroutine-per-Connection model. [The high-performance network library TNET](https://git.woa.com/trpc-go/tnet) is based on the **event-driven (Reactor) network model**, which can provide the ability to handle millions of connections. The tRPC-Go framework can now use the TNET network library through plugins to support millions of connections. In addition to supporting millions of connections, TNET has better performance than Golang's native Net library, and its high-performance network capabilities can be used through plugins.

The main advantages of TNET are:

- Supports more connections (up to millions)
- Higher performance (higher QPS, lower latency)
- Less memory usage (only about 10% of the memory used by go/net)
- Easy to use (consistent with the interface provided by Golang Net library)

**For more implementation details about TNET, please refer to the [article](https://km.woa.com/articles/show/542878).**

# Principle

In this chapter, we use two diagrams to illustrate the basic principles of the Goroutine-per-Connection model and the event-driven model in Golang. 

## Goroutine-per-Connection

![one_connection_one_coroutine](/.resources/user_guide/tnet/one_connection_one_coroutine.png)

In the Goroutine-per-Connection model, when the server accepts a new connection, it creates a Goroutine for that connection, and then reads data from the connection, processes the data, and sends data back to the connection in that Goroutine.

The scenario of millions of connections usually refers to long connection scenarios. Although the total number of connections is huge, only a small number of connections are active at any given time. Active connections refer to connections that have data to be read or written at a certain moment. Conversely, when there is no data to be read or written on a connection, it is called an idle connection. The idle connection Goroutine will block on the Read call. Although the Goroutine does not occupy scheduling resources, it still occupies memory resources, which ultimately leads to huge memory consumption. In this model, allocating a Goroutine for each connection in a scenario of millions of connections is expensive.

For example, as shown in the above diagram, the server accepts 5 connections and creates 5 Goroutines. At this moment, the first 3 connections are active connections, and data can be read from them smoothly. After processing the data, the server sends data back to the connections to complete a data exchange, and then starts the second round of data reading. The last 2 connections are idle connections, and when reading data from them, the process will be blocked. Therefore, the subsequent process is not triggered. As we can see, although only 3 connections can successfully read data at this moment, 5 Goroutines are allocated, resulting in a 40% waste of resources. The larger the proportion of idle connections, the more resources will be wasted.

## Event-driven

![event_driven](/.resources/user_guide/tnet/event_driven.png)



The event-driven model refers to using multiplexing (epoll/kqueue) to listen for events such as read and write on file descriptors (FDs), and then performing corresponding operations when events are triggered.

In the diagram, the Poller structure is responsible for listening to events on FDs. Each Poller occupies a Goroutine, and the number of Pollers is usually equal to the number of CPUs. We use a separate Poller to listen for read events on the listener port to accept new connections, and then listen for read events on each connection. When a connection becomes readable, a Goroutine is allocated to read data from the connection, process the data, and send data back to the connection. At this point, there will be no idle connections occupying Goroutines. In a scenario of millions of connections, only active connections are allocated Goroutines, which can make full use of memory resources.

For example, as shown in the above diagram, the server has 5 Pollers, one of which is responsible for listening to the listener and accepting new connections, and the other 4 Pollers are responsible for listening to read events on connections. When 2 connections become readable at a certain moment, a Goroutine is allocated for each connection to read data, process data, and send data back to the connection. Because it is already known that these two connections are readable, the Read process will not block, and the subsequent process can be executed smoothly. When writing data back to the connection, the Goroutine registers a write event with the Poller, and then exits. The Poller listens for write events on the connection and sends data when the connection is writable, completing a round of data exchange.

# Performance improvement effect

Thanks to optimizations such as batch packet sending and receiving and fine-grained memory management in TNET, TNET not only has performance advantages in scenarios with millions of connections, but also has significant performance improvements in scenarios with a small number of connections

To demonstrate the performance improvement of tRPC-Go using TNET, we conducted a comparison test on a 48-core 2.3GHz physical machine, using TNET and Go/net as the network transport layer for tRPC-Go, with the following settings:

- Client: 25 cores, Server: 8 cores
- Number of connections: 100, 500, 1000
- Server configuration: synchronous mode, asynchronous mode (configured through Server.WithServerAsync in tRPC-Go)
- Server program: echo service, received packet length: 122 bytes
- Load testing tool: eab
- Fixed P99 <= 10ms to see the maximum QPS that can be achieved

The test results showed that TNET outperformed Go/net in scenarios with 100, 500, and 1000 connections.

![tnet](/.resources/user_guide/tnet/tnet.png)

# Quick Startup

![transport_module](/.resources/user_guide/tnet/transport_module.png)

The transport module of tRPC-Go adopts a [pluggable](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/overview.md) design. We customized its transport module and used the tnet network library as the underlying network transport layer for tRPC-Go. The tRPC-Go framework (version 0.11.0 and above) has integrated the tnet network library.

(1) The websocket protocol also has its tnet version: https://git.woa.com/trpc-go/tnet/tree/master/extensions/websocket

And the tnet-transport version: https://git.woa.com/trpc-go/trpc-tnet-transport/tree/master/websocket

If users of the tRPC-Go framework need to use the websocket protocol, they can directly use the tnet-transport version.

(2) The HTTP protocol currently has an invasive modified version for fasthttp (https://git.woa.com/wineguo/fasthttp/tree/tnet)

Example usage can be found here: https://git.woa.com/wineguo/fasthttp/blob/tnet/tnetexamples/echo/tnet/main.go

(Please use with caution)

(3) For support of other business protocols (non-tRPC protocols):

As long as the codec implementation is similar to the ones provided in https://git.woa.com/trpc-go/trpc-codec, generally adding protocol: your_protocol and transport: tnet to the configuration can use the tnet capability (specific protocols can be handled on a case-by-case basis by contacting wineguo or leoxhyang).

## Usage

There are two ways to configure the tnet transport module. Users can choose one of them for configuration. It is recommended to use the first method.

(1) Add the plugin in the tRPC-Go framework configuration file.

(2) Call the WithTransport() method in the code to add the plugin.

### Method 1: Configuration file configuration (recommended)

**Note: tRPC-Go main framework version v0.11.0 and above is required**

Add tnet to the transport field in the tRPC-Go configuration file. Since the plugin currently only supports TCP, please do not configure the tnet plugin for UDP services.

**Server**:

``` yaml
server:   
  transport: tnet       # Applies to all services
  service:                                         
    - name: trpc.app.server.service             
      network: tcp
      transport: tnet   # Applies only to the current service 
```

After starting the server, confirm that the plugin is successfully enabled through the log:

INFO tnet/server_transport.go service:trpc.app.server.service is using tnet transport, current number of pollers: 1

**Client**:

``` yaml
client:   
  transport: tnet       # Applies to all services
  service:                                         
    - name: trpc.app.server.service             
      network: tcp
      transport: tnet   # Applies only to the current service 
```

After starting the client, confirm that the plugin is successfully enabled through the log (Trace level):

Debug tnet/client_transport.go roundtrip to:127.0.0.1:8000 is using tnet transport, current number of pollers: 1

### Method 2: Code configuration

**Note: tRPC-Go main framework version v0.11.0 and above is required**

**Server**:

This method will configure all services in the server. If there is an HTTP protocol service in the server, an error will occur.

``` go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
  // Create a serverTransport
  trans := tnet.NewServerTransport()
  // Create a trpc service
  s := trpc.NewServer(server.WithTransport(trans))
  pb.RegisterGreeterService(s, &greeterServiceImpl{})
  s.Serve()
}
```

**Client**:

``` go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
	proxy := pb.NewGreeterServiceClientProxy()
	trans := tnet.NewClientTransport()
	rsp, err := proxy.SayHello(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "Hello"}, client.WithTransport(trans))
}
```

## Supported options

There are two main options related to performance tuning:

1. `tnet.SetNumPollers` is used to set the number of pollers. Its default value is 1. Depending on the business scenario, this number needs to be increased accordingly (you can try 2 to the power of CPU cores, such as 2, 4, 8, 16... during business load testing). This setting can be customized through flags or read from environment variables to avoid repeatedly compiling binaries.
2. `server.WithServerAsync` is used to set the synchronous and asynchronous mode. Its default value is true (asynchronous). Depending on the business scenario, users can try setting it to synchronous mode by using `server.WithServerAsync(false)` during load testing to make comparisons.

The following are examples of setting the above two options:

Set the number of poller:

``` go
import "git.woa.com/trpc-go/tnet"

var num uint

func main() {
    flag.UintVar(&num, "n", 4, "set the number of tnet poller")
    tnet.SetNumPollers(num)
    // ..
}
```

Set synchronous mode:

``` go
import (
    "git.code.oa.com/trpc-go/trpc-go/server"
    "git.code.oa.com/trpc-go/trpc-go"
)

func main() {
    // This is the configuration for all services globally
    // The following configuration is recommended, which can configure a single server service separately
    s := trpc.NewServer(server.WithServerAsync(false))
    // ..
}
```

Configure synchronous mode for a specific service in the configuration file:

``` yaml
server:  # Server configuration
  app: yourAppName  # Application name of the business
  server: helloworld_svr  # Process service name
  service:  # Services provided by the business, there can be multiple
    - name: helloworld.helloworld_svr  # Routing name of the service
      ip: 127.0.0.1  # Service listening IP address. Placeholders ${ip} can be used. Either ip or nic can be used, and ip has priority.
      # nic: eth0
      port: 8000  # Service listening port. Placeholders ${port} can be used.
      network: tcp  # Network listening type: tcp or udp
      protocol: trpc  # Application layer protocol: trpc or http
      transport: tnet # Use the tnet transport network library
      server_async: false # Set to synchronous mode
      timeout: 1000  # Maximum processing time for a request, in milliseconds
```

In addition, the plugin supports the KeepAlive option, and KeepAlive is enabled by default in the tnet network library.

``` go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
    t := tnet.NewServerTransport(tnet.WithKeepAlivePeriod(20 * time.Second))
    //...
}
```

# Business integration cases and effects

[Records of businesses that have integrated tnet](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMiax1Z20yRUSK67eOsW?scode=AJEAIQdfAAoT0g9EAMAGkAxgZOAFM)

# To-do list

1. UDP service

# Related sharing

Technical salon sharing by IEG Value-added Services Department in September 2022:

[Design, implementation, and performance optimization of tnet high-performance network library](todo)

# OWNER

leoxhyang
wineguo