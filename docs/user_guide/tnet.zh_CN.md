## 前言

Golang 的 Net 库提供了简单的非阻塞调用接口，网络模型采用一个连接一个协程（Goroutine-per-Connection）。在多数的场景下，这个模型简单易用，但是当连接数量成千上万之后，在百万连接的级别，为每个连接分配一个协程将消耗极大的内存，并且调度大量协程也变的非常困难。为了支持百万连接的功能，必须打破一个连接一个协程模型，[高性能网络库 tnet](https://git.woa.com/trpc-go/tnet) **基于事件驱动（Reactor）的网络模型**，能够提供百万连接的能力。tRPC-Go 框架现在已经集成 tnet 网络库，从而支持百万连接功能。除此之外，tnet 还支持批量收发包功能，零拷贝 buffer，精细化内存管理等优化，因此拥有比 Golang 原生 net 库更优秀的性能。

关于 tnet 的更多实现细节见 [文章](https://km.woa.com/articles/show/542878)

## 原理

在本章中，我们通过两张图展示 Golang 中一个连接一个协程模型和基于事件驱动模型的基本原理。

### 一个连接一个协程

![one_connection_one_coroutine](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/tnet/one_connection_one_coroutine_zh_CN.png)

一个连接一个协程的模式下，服务端 Accept 一个新的连接，就为该连接起一个协程，然后在这个协程中从连接读数据、处理数据、向连接发数据。

百万连接场景通常指的是长连接场景，虽然连接总数巨大，但是活跃的连接数量只占少数，活跃的连接指的是某一时刻连接上有数据可读/写，相对的当连接上没有数据可读/写，此连接被称为空闲连接。空闲连接协程会阻塞在 Read 调用，此时协程虽然不会占用调度资源，但是依然会占用内存资源，最终导致消耗巨大的内存。按照这种模式，在百万连接场景下，为每个连接都分配一个协程成本是昂贵的。

例如上图所示，服务端 Accept 了 5 个连接，创建了 5 个协程，在这一时刻，前 3 个连接是活跃连接，可以顺利的从连接中读取得到数据，处理数据后向连接发送数据完成一次数据交互，然后进行第二轮数据读取。而后 2 个连接是空闲连接，从连接中读取数据的时候会阻塞，于是后续的流程没有触发。可以看到，这一时刻，虽然只有 3 个连接是可以成功地读取到数据，但是却分配了 5 个协程，资源浪费了 40%，空闲连接占比越大，资源浪费就越多。

### 事件驱动

![event_driven](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/tnet/event_driven.png)

事件驱动模式是指利用多路复用（epoll / kqueue）监听 FD 的可读、可写等事件，当有事件触发的时候做相应的处理。

图中 Poller 结构负责监听 FD 上的事件，每个 Poller 占用一个协程，Poller 的数量通常等于 CPU 的数量。我们采用了单独的 Poller 来监听 listener 端口的可读事件来 Accept 新的连接，然后监听每个连接的可读事件，当连接变得可读时，再分配协程从连接读数据、处理数据、向连接发数据。此时不会再有空闲连接占用协程，在百万连接场景下，只为活跃连接分配协程，可以充分利用内存资源。

例如上图所示，服务端有 5 个 Poller，其中有 1 个单独的 Poller 负责监听 Listener，接收新连接，其余 4 个 Poller 负责监听连接可读事件，在连接可读时，触发处理过程。在这一时刻，Poller 监听到有 2 个连接可读，于是为每个连接分配一个协程，从连接中读取数据、处理数据、写回数据，因为此时已经知道这两个连接可读，所以 Read 过程不会阻塞，后续的流程可以顺利执行，最终 Write 的时候，会向 Poller 注册可写事件，然后协程退出，Poller 监听连接可写，在连接可写的时候发送数据，完成一轮数据交互。

## 快速上手

### 使用方法

支持以下配置方式，用户选择其一进行配置即可，推荐使用第一种配置方法。

1. 直接使用对应的 tag 版本
2. 在 tRPC-Go 框架配置文件中启用 tnet
3. 在代码中调用 WithTransport() 方法启用 tnet

#### 方法 1：直接使用 tag（推荐）

从 v0.15.1 开始，对于不同版本的 tRPC-Go，可以使用对应版本的 tnet，如 v0.15.1-tnet-enabled（假如后缀带有数字，选取数字最高的为最新版，比如 [v0.15.1-tnet-enabled.2](https://git.woa.com/trpc-go/trpc-go/-/tags/v0.15.1-tnet-enabled.2) ）

#### 方法 2：配置文件

**注意：需要 tRPC-Go 主框架版本 v0.11.0 及以上**

在 tRPC-Go 的配置文件中的 transport 字段添加 tnet。

从 v0.11.0 版本开始，tnet 插件仅支持 tcp。

从 v0.19.0-beta 版本开始，tnet 插件同时支持 tcp 和 udp。

在 < v0.19.0-beta 版本中，net 配置 udp，transport 配置 tnet 的话，会自动 fallback 到 golang net 库。

服务端和客户端可以单独开启 tnet，二者互不影响。

##### 服务端

```yaml
server:   
  transport: tnet       # 对所有 service 全部生效
  service:                                         
  - name: trpc.app.server.service             
    network: tcp  # 此处也可以配置为 tcp,udp 旧版本时 udp 会自动 fallback 到 golang net 库
    transport: tnet   # 只对当前 service 生效
```

服务端启动服务后通过 log 确认插件启用成功：

INFO tnet/server_transport.go service:trpc.app.server.service is using tnet transport, current number of pollers: 1

##### 客户端

**注意：需要 tRPC-Go 主框架版本 v0.15.0 及以上**

从 v0.11.0 版本开始，tnet 插件仅支持 tcp。

从 v0.19.0-beta 版本开始，tnet 插件同时支持 tcp 和 udp。

在 < v0.19.0-beta 版本中，net 配置 udp，transport 配置 tnet 的话，会自动 fallback 到 golang net 库。

* 使用连接多路复用

```yaml
client:
  transport: tnet  # 对所有 service 全部生效
  service:
  - name: trpc.app.server.service
    network: tcp  
    transport: tnet  # 只对当前 service 生效
    conn_type: multiplexed  # 连接类型为多路复用
    multiplexed:
      multiplexed_dial_timeout: 1s  # dial 超时，默认为 1 秒
      max_vir_conns_per_conn: 0  # 每个实际连接的最大虚拟连接数，默认为 0（表示无限制）
      enable_metrics: true  # 是否启用 metrics, 默认为 false
```

推荐客户端开启 tnet 的同时使用多路复用连接模式，充分利用 tnet 批量收发包的能力，提高性能。

* 使用连接池

```yaml
client:
  transport: tnet  # 对所有 service 全部生效
  service:
  - name: trpc.app.server.service
    network: tcp
    transport: tnet  # 只对当前 service 生效
    conn_type: connpool  # 连接类型为连接池，以下选项都是针对连接池的
    connpool:
      # 优先级：option dial_timeout ≈ 上下文超时 > yaml dial_timeout
      # 当选项 dial_timeout 和上下文超时都存在时，真实的 dial 超时 = min(option dial_timeout, 上下文超时)
      dial_timeout: 200ms  # 连接池：dial 超时，默认 200 毫秒
      force_close: false  # 连接池：是否强制关闭连接，默认为 false
      idle_timeout: 50s  # 连接池：空闲超时，默认 50 秒
      max_active: 0  # 连接池：最大活跃连接数，默认 0（表示无限制）
      max_conn_lifetime: 0s  # 连接池：连接的最大生命周期，默认 0 秒（表示无限制）
      max_idle: 65536  # 连接池：最大空闲连接数，默认 65536
      min_idle: 0  # 连接池：最小空闲连接数，默认 0
      pool_idle_timeout: 100s  # 连接池：关闭整个池的空闲超时，默认 100 秒
      push_idle_conn_to_tail: false  # 连接池：将空闲连接回收到空闲列表的头部 / 尾部，默认为 false（头部）
      wait: false  # 连接池：当总连接数达到 max_active 时，是等待直至超时还是立即返回错误，默认为 false
```

客户端启动服务后通过 log 确认插件启用成功（Trace 级别）：

Debug tnet/client_transport.go roundtrip to:127.0.0.1:8000 is using tnet transport, current number of pollers: 1

#### 方法 3：代码配置

**注意：需要 tRPC-Go 主框架版本 v0.11.0 及以上**

##### 服务端

这种方式会对 server 的所有 service 都进行配置，如果 server 中存在 http 协议的 service，会出现报错。

```go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
    // 创建一个 serverTransport
    trans := tnet.NewServerTransport()
    // 创建一个 trpc 服务
    s := trpc.NewServer(server.WithTransport(trans))
    pb.RegisterGreeterService(s, &greeterServiceImpl{})
    s.Serve()
}
```

##### 客户端

* 使用连接多路复用

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/transport/tnet"
    tnetmultiplexed "git.code.oa.com/trpc-go/trpc-go/transport/tnet/multiplexed"
)

func main() {
    trans := tnet.NewClientTransport()
    proxy := pb.NewGreeterServiceClientProxy(
        client.WithTransport(trans),
        client.WithMultiplexedPool(
            tnet.NewMultiplexedPool(
                tnetmultiplexed.WithDialTimeout(time.Second),
                tnetmultiplexed.WithEnableMetrics(), 
                tnetmultiplexed.WithMaxConcurrentVirtualConnsPerConn(0),
            ),
        ),
    )
    rsp, err := proxy.SayHello(
        trpc.BackgroundContext(), 
        &pb.HelloRequest{Msg: "Hello"}, 
    )
}
```

* 使用连接池

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/pool/connpool"
    "git.code.oa.com/trpc-go/trpc-go/transport/tnet"
)

func main() {
    trans := tnet.NewClientTransport()
    proxy := pb.NewGreeterClientProxy(
        client.WithTransport(trans),
        client.WithPool(
            tnet.NewConnectionPool(
                connpool.WithDialTimeout(time.Second),
                // ...
            ),
        ),
    )
    rsp, err := proxy.SayHello(
        trpc.BackgroundContext(),
        &pb.HelloRequest{Msg: "Hello"},
    )
}
```

### 其他插件

1. websocket 协议同样存在其 tnet 版本：<https://git.woa.com/trpc-go/tnet/tree/master/extensions/websocket>

以及 tnet-transport 版本：<https://git.woa.com/trpc-go/trpc-tnet-transport/tree/master/websocket>

如果 trpc-go 框架的用户需要使用 websocket 协议，可以直接使用 tnet-transport 版本

2. HTTP 协议目前有对 fasthttp 的侵入修改版 <https://git.woa.com/wineguo/fasthttp/tree/tnet>

使用例子见：<https://git.woa.com/wineguo/fasthttp/blob/tnet/tnetexamples/echo/tnet/main.go>

（please use with caution）

3. 对于其他业务协议（非 tRPC 协议）的支持：

只要 codec 的实现类似于 <https://git.woa.com/trpc-go/trpc-codec> 中提供的部分，一般来说在配置中增加 `protocol: your_protocol` 以及 `transport: tnet` 即可使用 tnet 能力（具体协议可以联系 wineguo 或 leoxhyang 进行 case by case 的处理）

## 适用场景

我们使用 tnet 进行了压力测试，从[测试结果](https://km.woa.com/articles/show/586072)来看，tnet transport 相比 gonet transport 在特定场景下可以提供更好的性能，但是不是所有场景都有优势。在此总结 tnet transport 的优势场景。

### tnet 优势场景

作为服务端使用 tnet，客户端发送请求使用多路复用的模式，可以充分发挥 tnet 批量收发包的能力，可以提高 QPS，降低 CPU 占用

作为服务端使用 tnet，存在大量的不活跃连接的场景，可以通过减少协程数等逻辑降低内存占用

作为客户端使用 tnet，开启多路复用模式，可以充分发挥 tnet 批量收发包的能力，可以提高 QPS。

### 其他场景

作为服务端使用 tnet，客户端发送请求使用连接池模式，性能表现和原 gonet 基本持平

作为客户端使用 tnet，开启连接池模式，性能表现和原 gonet 基本持平

## 常见问题

**Q：tnet 支持 HTTP 吗？**

A：tnet 不支持 HTTP，在使用 HTTP 协议的服务端/客户端开启 tnet 的话，会自动降级使用 golang net 库。

---

**Q：客户端/服务端开启 tnet 后报错 "transport FramerBuilder empty"？**

A：检查 tRPC-Go 的版本是否低于 v0.15.0 且使用了 HTTP 协议，建议升级 tRPC-Go 版本，如果没有办法升级 tRPC-Go 版本，可以将 transport 的配置放到非 HTTP 协议的 service 级别。

```yaml
server:
  service:
  - name: trpc.app.server.service
    protocol: trpc
    transport: tnet # 只为当前的 service 开启 tnet
```

---

**Q：开启 tnet 之后性能为什么没有提升？**

A：tnet 并不是万金油，在特定的场景下可以充分利用 Writev 批量发包，减少系统调用，是可以提高服务的性能的。

可以通过开启客户端的 tnet 多路复用（multiplexed）功能，尽可能利用 Writev 批量发包；

为整个服务链路都开启 tnet，上游使用多路复用的话，当前服务端也可以充分利用 Writev 批量发包；

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

每隔 3s，就会打印多路复用状态的日志。在日志中可以看到当前的连接数是 1 个，虚拟连接总数是 98 个。

DEBUG tnet multiplex status: network: tcp, address: 127.0.0.1:7002, connections number: 1, concurrent virtual connection number: 98

同时也会上报自定义监控，监控项格式是：

并发连接数：trpc.MuxConcurrentConnections.$network.$address

虚拟连接总数：trpc.MuxConcurrentVirConns.$network.$address

假设现在修改每个连接上的最大并发虚拟连接数量为 25，可以这样写：

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

---

**Q：开启 tnet 后提示 "switch to gonet default transport, tnet server transport doesn't support network type [udp]"？**

A: 这个报错的意思是，tnet transport 暂时不支持 UDP，自动降级使用 golang net 库，不影响服务正常启动。

---

**Q：怎么验证我的服务是否成功使用了 tnet？**

A：正常来说只要配置文件里配上了 tnet，框架会自动识别哪些场景可以使用 tnet，对于不能使用 tnet 的场景会回降级使用 golang net 库。但是也可以通过观察日志来判断是否使用了 tnet transport。

服务端：检查服务日志，如果出现 "INFO service:trpc.app.server.service is using tnet transport, current number of pollers: 1" 表示服务端已经成功开启了 tnet。

客户端：需要开启 Trace 级别日志，发起请求的时候如果出现 "DEBUG roundtrip to:127.0.0.1:8000 is using tnet transport, current number of pollers: 1" 表示客户端已经成功开启了 tnet。

---

## 业务接入案例和效果

[tnet 现已接入的业务记录](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMiax1Z20yRUSK67eOsW?scode=AJEAIQdfAAoT0g9EAMAGkAxgZOAFM)

## 相关分享

[IEG 增值服务部 2022 年 9 月技术沙龙分享](todo)

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
