[TOC]

tRPC-Go 接入高性能网络库



# 前言

Golang 的 Net 库提供了简单的非阻塞调用接口，网络模型采用**一个连接一个协程（Goroutine-per-Connection）**。在多数的场景下，这个模型简单易用，但是当连接数量成千上万之后，在百万连接的级别，为每个连接分配一个协程将消耗极大的内存，并且调度大量协程也变的非常困难。为了支持百万连接的功能，必须打破一个连接一个协程模型，[高性能网络库 TNET](https://git.woa.com/trpc-go/tnet) 基于**事件驱动（Reactor）的网络模型**，能够提供百万连接的能力。tRPC-Go 框架现在可以通过插件的形式使用 TNET 网络库，从而支持百万连接功能。除了支持百万连接外，TNET 拥有比 Golang 原生 Net 库更优秀的性能，通过插件同时可以用到 TNET 的高性能网络能力。

TNET 的主要优点：

- 支持更多的连接数（百万级别）
- 更高的性能（QPS ↑, 时延↓）
- 更少的内存（相较于 go/net 只需要 ~10% 的内存）
- 易用性（和 Golang Net 库提供的接口保持一致）

**关于 TNET 的更多实现细节见 [文章](https://km.woa.com/articles/show/542878)**

# 原理

在本章中，我们通过两张图展示 Golang 中一个连接一个协程模型和基于事件驱动模型的基本原理。

## 一个连接一个协程

![one_connection_one_coroutine](/.resources/user_guide/tnet/one_connection_one_coroutine_zh_CN.png)

一个连接一个协程的模式下，服务端 Accept 一个新的连接，就为该连接起一个协程，然后在这个协程中从连接读数据、处理数据、向连接发数据。

百万连接场景通常指的是长连接场景，虽然连接总数巨大，但是活跃的连接数量只占少数，活跃的连接指的是某一时刻连接上有数据可读/写，相对的当连接上没有数据可读/写，此连接被称为空闲连接。空闲连接协程会阻塞在 Read 调用，此时协程虽然不会占用调度资源，但是依然会占用内存资源，最终导致消耗巨大的内存。按照这种模式，在百万连接场景下，为每个连接都分配一个协程成本是昂贵的。

例如上图所示，服务端 Accept 了 5 个连接，创建了 5 个协程，在这一时刻，前 3 个连接是活跃连接，可以顺利的从连接中读取得到数据，处理数据后向连接发送数据完成一次数据交互，然后进行第二轮数据读取。而后 2 个连接是空闲连接，从连接中读取数据的时候会阻塞，于是后续的流程没有触发。可以看到，这一时刻，虽然只有 3 个连接是可以成功地读取到数据，但是却分配了 5 个协程，资源浪费了 40%，空闲连接占比越大，资源浪费就越多。

## 事件驱动

![event_driven](/.resources/user_guide/tnet/event_driven.png)

事件驱动模式是指利用多路复用（epoll / kqueue）监听 FD 的可读、可写等事件，当有事件触发的时候做相应的处理。

图中 Poller 结构负责监听 FD 上的事件，每个 Poller 占用一个协程，Poller 的数量通常等于 CPU 的数量。我们采用了单独的 Poller 来监听 listener 端口的可读事件来 Accept 新的连接，然后监听每个连接的可读事件，当连接变得可读时，再分配协程从连接读数据、处理数据、向连接发数据。此时不会再有空闲连接占用协程，在百万连接场景下，只为活跃连接分配协程，可以充分利用内存资源。

例如上图所示，服务端有 5 个 Poller，其中有 1 个单独的 Poller 负责监听 Listener，接收新连接，其余 4 个 Poller 负责监听连接可读事件，在连接可读时，触发处理过程。在这一时刻，Poller 监听到有 2 个连接可读，于是为每个连接分配一个协程，从连接中读取数据、处理数据、写回数据，因为此时已经知道这两个连接可读，所以 Read 过程不会阻塞，后续的流程可以顺利执行，最终 Write 的时候，会向 Poller 注册可写事件，然后协程退出，Poller 监听连接可写，在连接可写的时候发送数据，完成一轮数据交互。

# 性能提升效果

TNET 的性能优势不仅体现在百万连接上，在少量连接的时候，得益于 TNET 中的批量收发包、内存的精细化管理等优化，tRPC-Go 的性能也有了不小的提升。

为了展示 tRPC-Go 使用 TNET 后的效果，我们在 48 核 2.3GHz 物理机上对 tRPC-Go 分别使用 TNET、Go/net 作为网络传输层进行压测对比，设置如下：

- 客户端 25 核，服务端 8 核
- 连接数：100，500，1000
- 服务端配置：同步模式，异步模式（通过 tRPC-Go 的 Server.WithServerAsync 来配置）
- 服务端程序：echo 服务，收到包长：122 Byte
- 压测工具：eab
- 固定 P99 <= 10ms 的情况下看能够达到的最大 QPS

测试结果表现 TNET 在 100, 500, 1000 连接数下的表现均好于 Go/net。

![tnet](/.resources/user_guide/tnet/tnet_zh_CN.png)

# 快速上手

![transport_module](/.resources/user_guide/tnet/transport_module_zh_CN.png)

tRPC-Go 的 transport 模块采用[可插拔](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/overview_zh_CN.md)化设计，我们定制了它的 transport 模块，将 tnet 网络库 作为 tRPC-Go 的底层网络传输层。tRPC-Go 框架（v0.11.0 版本以上）已经集成了 tnet 网络库。

（1）websocket 协议同样存在其 tnet 版本：https://git.woa.com/trpc-go/tnet/tree/master/extensions/websocket

以及 tnet-transport 版本：https://git.woa.com/trpc-go/trpc-tnet-transport/tree/master/websocket

如果 trpc-go 框架的用户需要使用 websocket 协议，可以直接使用 tnet-transport 版本

（2）HTTP 协议目前有对 fasthttp 的侵入修改版（https://git.woa.com/wineguo/fasthttp/tree/tnet）

使用例子见：https://git.woa.com/wineguo/fasthttp/blob/tnet/tnetexamples/echo/tnet/main.go

（please use with caution）

（3）对于其他业务协议（非 tRPC 协议）的支持：

只要 codec 的实现类似于 https://git.woa.com/trpc-go/trpc-codec 中提供的部分，一般来说在配置中增加 `protocol: your_protocol` 以及 `transport: tnet` 即可使用 tnet 能力（具体协议可以联系 wineguo 或 leoxhyang 进行 case by case 的处理）

## 使用方法

支持两种配置方式，用户选择其一进行配置即可，推荐使用第一种配置方法。

（1）在 tRPC-Go 框架配置文件中添加插件

（2）在代码中调用 WithTransport() 方法添加插件

### 方法一：配置文件配置（推荐）

**注意：需要 tRPC-Go 主框架版本 v0.11.0 及以上**

在 tRPC-Go 的配置文件中的 transport 字段添加 tnet。因为插件现阶段只支持 TCP，所以 UDP 服务请不要配置 tnet 插件。

**服务端**：

``` yaml
server:   
  transport: tnet       # 对所有 service 全部生效
  service:                                         
    - name: trpc.app.server.service             
      network: tcp
      transport: tnet   # 只对当前 service 生效  
```

服务端启动服务后通过 log 确认插件启用成功：

INFO tnet/server_transport.go service:trpc.app.server.service is using tnet transport, current number of pollers: 1

**客户端**：

``` yaml
client:   
  transport: tnet       # 对所有 service 全部生效
  service:                                         
    - name: trpc.app.server.service             
      network: tcp
      transport: tnet   # 只对当前 service 生效 
```

客户端启动服务后通过 log 确认插件启用成功（Trace 级别）：

Debug tnet/client_transport.go roundtrip to:127.0.0.1:8000 is using tnet transport, current number of pollers: 1

### 方法二：代码配置

**注意：需要 tRPC-Go 主框架版本 v0.11.0 及以上**

**服务端**：

这种方式会对 server 的所有 service 都进行配置，如果 server 中存在 http 协议的 service，会出现报错。

``` go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
  // 创建一个 serverTransport
  trans := tnet.NewServerTransport()
  // 创建一个trpc服务
  s := trpc.NewServer(server.WithTransport(trans))
  pb.RegisterGreeterService(s, &greeterServiceImpl{})
  s.Serve()
}
```

**客户端**：

``` go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
	proxy := pb.NewGreeterServiceClientProxy()
	trans := tnet.NewClientTransport()
	rsp, err := proxy.SayHello(trpc.BackgroundContext(), &pb.HelloRequest{Msg: "Hello"}, client.WithTransport(trans))
}
```

## 支持选项

涉及性能调优的选项主要有以下两个：

1. `tnet.SetNumPollers` 用来设置 pollers 的个数，其默认值为 1，根据业务场景的不同，这个数量需要相应地调大（可在业务自身压测时依次尝试 2 的幂次直至 CPU 核数，比如 2, 4, 8, 16...），这种设置可以通过自定义 flag 或者从环境变量中读取，以避免反复编译二进制
2. `server.WithServerAsync` 用来设置同步异步模式，其默认值为 true（异步），根据业务场景的不同，用户在压测自身业务时可以尝试通过 `server.WithServerAsync(false)` 来设置为同步以进行对比

以上两个选项的设置示例如下：

设置 poller 个数：

``` go
import "git.woa.com/trpc-go/tnet"

var num uint

func main() {
    flag.UintVar(&num, "n", 4, "设置 tnet poller 个数")
    tnet.SetNumPollers(num)
    // ..
}
```

设置同步：

``` go
import (
    "git.code.oa.com/trpc-go/trpc-go/server"
    "git.code.oa.com/trpc-go/trpc-go"
)

func main() {
    // 这里是全局所有 service 进行配置
    // 更推荐下面的配置写法, 可以为某个 server service 单独配置
    s := trpc.NewServer(server.WithServerAsync(false))
    // ..
}
```

在配置文件中为某个 service 配置同步：

``` yaml
server:  # 服务端配置
  app: yourAppName  # 业务的应用名
  server: helloworld_svr  # 进程服务名
  service:  # 业务服务提供的 service，可以有多个
    - name: helloworld.helloworld_svr  # service 的路由名称
      ip: 127.0.0.1  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      # nic: eth0
      port: 8000  # 服务监听端口 可使用占位符 ${port}
      network: tcp  # 网络监听类型 tcp udp
      protocol: trpc  # 应用层协议 trpc http
      transport: tnet # 使用 tnet transport 网络库
      server_async: false # 设置为同步
      timeout: 1000  # 请求最长处理时间 单位 毫秒
```

此外，插件支持 KeepAlive 选项，在 tnet 网络库中 KeepAlive 默认开启。

``` go
import "git.code.oa.com/trpc-go/trpc-go/transport/tnet"

func main() {
    t := tnet.NewServerTransport(tnet.WithKeepAlivePeriod(20 * time.Second))
    //...
}
```

# 业务接入案例和效果

[tnet 现已接入的业务记录](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMiax1Z20yRUSK67eOsW?scode=AJEAIQdfAAoT0g9EAMAGkAxgZOAFM)

# 待实现功能

1. UDP 服务

# 相关分享

IEG 增值服务部 2022 年 9 月技术沙龙分享

[tnet 高性能网络库设计实现与性能优化](todo)

