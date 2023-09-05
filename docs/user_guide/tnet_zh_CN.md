# tRPC-Go 接入高性能网络库 tnet

[English](./tnet.md) | 中文

## 前言

Golang 的 Net 库提供了简单的非阻塞调用接口，网络模型采用`一个连接一个协程`。在多数的场景下，这个模型简单易用，但是当连接数量成千上万之后，在百万连接的级别，为每个连接分配一个协程将消耗极大的内存，并且调度大量协程也变的非常困难。为了支持百万连接的功能，必须打破一个连接一个协程模型，高性能网络库 [tnet](https://github.com/trpc-group/tnet) 基于`事件驱动`的网络模型，能够提供百万连接的能力。tRPC-Go 框架集成了 tnet 网络库，从而支持百万连接功能。除此之外，tnet 还支持批量收发包功能，零拷贝缓存，精细化内存管理等优化，因此拥有比 Golang 原生 net 库更优秀的性能。


## 原理

我们通过两张图展示 Golang 中一个连接一个协程模型和基于事件驱动模型的基本原理。

### 一个连接一个协程

![goroutine_per_connection](/.resources/user_guide/tnet/goroutine_per_connection_zh_CN.png)

一个连接一个协程的模式下，服务端 Accept 一个新的连接，就为该连接起一个协程，然后在这个协程中从连接读数据、处理数据、向连接发数据。

百万连接场景通常指的是长连接场景，虽然连接总数巨大，但是活跃的连接数量只占少数，活跃的连接指的是某一时刻连接上有数据可读/写，相对的当连接上没有数据可读/写，此连接被称为空闲连接。空闲连接协程会阻塞在 Read 调用，此时协程虽然不会占用调度资源，但是依然会占用内存资源，最终导致消耗巨大的内存。按照这种模式，在百万连接场景下，为每个连接都分配一个协程成本是昂贵的。

例如上图所示，服务端 Accept 了 5 个连接，创建了 5 个协程，在这一时刻，前 3 个连接是活跃连接，可以顺利的从连接中读取得到数据，处理数据后向连接发送数据完成一次数据交互，然后进行第二轮数据读取。而后 2 个连接是空闲连接，从连接中读取数据的时候会阻塞，于是后续的流程没有触发。可以看到，这一时刻，虽然只有 3 个连接是可以成功地读取到数据，但是却分配了 5 个协程，资源浪费了 40%，空闲连接占比越大，资源浪费就越多。

### 事件驱动

![reactor](/.resources/user_guide/tnet/reactor.png)

事件驱动模式是指利用多路复用（epoll / kqueue）监听 FD 的可读、可写等事件，当有事件触发的时候做相应的处理。

图中 Poller 结构负责监听 FD 上的事件，每个 Poller 占用一个协程，Poller 的数量通常等于 CPU 的数量。我们采用了单独的 Poller 来监听 listener 端口的可读事件来 Accept 新的连接，然后监听每个连接的可读事件，当连接变得可读时，再分配协程从连接读数据、处理数据、向连接发数据。此时不会再有空闲连接占用协程，在百万连接场景下，只为活跃连接分配协程，可以充分利用内存资源。

例如上图所示，服务端有 5 个 Poller，其中有 1 个单独的 Poller 负责监听 Listener 事件，接收新连接，其余 4 个 Poller 负责监听连接可读事件，在连接可读时，触发处理过程。在这一时刻，Poller 监听到有 2 个连接可读，于是为每个连接分配一个协程，从连接中读取数据、处理数据、写回数据，因为此时已经知道这两个连接可读，所以 Read 过程不会阻塞，后续的流程可以顺利执行，最终 Write 的时候，会向 Poller 注册可写事件，然后协程退出，Poller 监听连接可写，在连接可写的时候发送数据，完成一轮数据交互。

## 快速上手

### 使用方法

支持两种配置方式，用户选择其一进行配置即可，推荐使用第一种配置方法。

（1）在 tRPC-Go 框架配置文件中启用 tnet

（2）在代码中调用 WithTransport() 方法启用 tnet

#### 方法一：配置文件（推荐）

在 tRPC-Go 的配置文件中的 transport 字段添加 tnet。因为插件现阶段只支持 TCP，所以 UDP 服务请不要配置 tnet 插件。服务端和客户端可以单独开启 tnet，二者互不影响。

**服务端**：

``` yaml
server:   
  transport: tnet       # 对所有 service 全部生效
  service:                                         
    - name: trpc.app.server.service             
      network: tcp
      transport: tnet   # 只对当前 service 生效  
```

服务端启动后，日志提示启用 tnet 成功：

`INFO tnet/server_transport.go service:trpc.app.server.service is using tnet transport, current number of pollers: 1`

**客户端**：

``` yaml
client:   
  transport: tnet       # 对所有 service 全部生效
  service:                                         
    - name: trpc.app.server.service             
      network: tcp
      transport: tnet   # 只对当前 service 生效 
      conn_type: multiplexed # 使用多路复用连接模式 
      multiplexed:
        enable_metrics: true # 开启多路复用运行状态的监控
```

推荐客户端开启 tnet 的同时使用多路复用连接模式，充分利用 tnet 批量收发包的能力，提高性能。

客户端启动服务后通过 log 确认插件启用成功（Trace 级别）：

`Debug tnet/client_transport.go roundtrip to:127.0.0.1:8000 is using tnet transport, current number of pollers: 1`

#### 方法二：代码配置

**服务端**：

注意：这种方式会对 server 的所有 service 都启动 tnet。

``` go
import "trpc.group/trpc-go/trpc-go/transport/tnet"

func main() {
  // 创建一个 ServerTransport
  trans := tnet.NewServerTransport()
  // 创建一个 trpc 服务
  s := trpc.NewServer(server.WithTransport(trans))
  pb.RegisterGreeterService(s, &greeterServiceImpl{})
  s.Serve()
}
```

**客户端**：

``` go
import "trpc.group/trpc-go/trpc-go/transport/tnet"

func main() {
  proxy := pb.NewGreeterClientProxy()
  trans := tnet.NewClientTransport()
  rsp, err := proxy.SayHello(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "Hello"}, client.WithTransport(trans))
}
```

## 适用场景

我们使用 tnet 进行了压力测试，从测试结果来看，tnet transport 相比 gonet transport 在特定场景下可以提供更好的性能，但是不是所有场景都有优势。在此总结 tnet transport 的优势场景。

**tnet 优势场景：**

- 作为服务端使用 tnet，客户端发送请求使用多路复用的模式，可以充分发挥 tnet 批量收发包的能力，可以提高 QPS，降低 CPU 占用

- 作为服务端使用 tnet，存在大量的不活跃连接的场景，可以通过减少协程数等逻辑降低内存占用

- 作为客户端使用 tnet，开启多路复用模式，可以充分发挥 tnet 批量收发包的能力，可以提高 QPS。

**其他场景：**

- 作为服务端使用 tnet，客户端发送请求使用连接池模式，性能表现和 gonet 基本持平

- 作为客户端使用 tnet，开启连接池模式，性能表现和 gonet 基本持平

## 常见问题

#### Q：tnet 支持 HTTP 吗？

tnet 不支持 HTTP，在使用 HTTP 协议的服务端/客户端开启 tnet 的话，会自动降级使用 golang net 库。

#### Q：开启 tnet 之后性能为什么没有提升？

tnet 并不是万金油，在特定的场景下可以充分利用 Writev 批量发包，减少系统调用，是可以提高服务的性能的。如果在 tnet 的优势场景下服务性能仍不理想，可以按照以下步骤针对自己的服务进行优化。

开启客户端的 tnet 多路复用（multiplexed）功能，尽可能利用 Writev 批量发包；

为整个服务链路开启 tnet 和多路复用，上游使用多路复用的话，当前服务端也可以充分利用 Writev 批量发包；

如果使用了多路复用功能，可以开启多路复用监控，查看每个连接上有多少虚拟连接，如果并发量较大，导致单连接上的虚拟连接数过多，也会影响性能，添加配置开启多路复用监控上报。


```yaml
client:
  service:
    - name: trpc.test.helloworld.Greeter1
      transport: tnet
      conn_type: multiplexed
      multiplexed:
        enable_metrics: true # 开启多路复用运行状态的监控
```

每隔3s，就会打印多路复用状态的日志。在日志中可以看到当前的连接数是1个，虚拟连接总数是98个。

`DEBUG tnet multiplex status: network: tcp, address: 127.0.0.1:7002, connections number: 1, concurrent virtual connection number: 98`

同时也会上报自定义监控，监控项格式是：

并发连接数：`trpc.MuxConcurrentConnections.$network.$address`

虚拟连接总数：`trpc.MuxConcurrentVirConns.$network.$address`

假设希望设置每个连接上的最大并发虚拟连接数量为25，可以添加如下配置：

```yaml
client:
  service:
    - name: trpc.test.helloworld.Greeter1
      transport: tnet
      conn_type: multiplexed
      multiplexed:
        enable_metrics: true # 开启多路复用监控
        max_vir_conns_per_conn: 25 # 每个连接上的最大并发虚拟连接数量
```

#### Q：开启 tnet 后提示 `switch to gonet default transport, tnet server transport doesn't support network type [udp]`？

这个报错的意思是，tnet transport 暂时不支持 UDP，自动降级使用 golang net 库，不影响服务正常启动。

