# tRPC-Go 客户端连接模式

## 前言

tRPC-Go client，作为请求发起方，提供了多种连接模式以适应不同需求。这些模式包括短连接、连接池、IO 复用，以及针对 HTTP 协议的 HTTP 连接池。根据使用的协议和具体需求，用户可以灵活选择连接模式。

- 对于使用 trpc 协议的 client，支持短连接、连接池和 IO 复用连接模式，默认使用连接池模式。
- 对于使用 HTTP 协议的 client，支持短连接和 HTTP 连接池模式，默认使用 HTTP 连接池模式。
`注意：此处提到的连接池是 tRPC-Go 自身实现的 transport 里面的连接池。对于 trpc-database 等组件，它们通过插件模式使用开源库替换了原有的 transport，因此不使用 tRPC-Go 里面的连接池。`

## 原理和实现

### 短连接

client 每次请求都会新建一个连接，请求完成后连接会被销毁。请求量很大的情况下，服务的吞吐量会受到很大的影响，性能损耗也很大。

使用场景：一次性请求或者请求的被调服务是老服务，不支持在一个连接上接受多个请求的情况下使用。

### 连接池

client 针对每个下游 IP 都会维护一个连接池，每次请求先从名字服务获取一个 ip，根据 ip 获取对应连接池，再从连接池中获取一个连接，请求完成后连接会被放回连接池，在请求过程中，这个连接是独占的，不可复用的。连接池内的连接按照策略进行销毁和新建。一次调用绑定一个连接，当上下游规模很大的情况下，网络中存在的连接数以 MxN 的速度扩张，带来巨大的调度压力和计算开销。

使用场景：基本所有的场景都可以使用。
注意：因为连接池队列的策略是先进后出，如果后端是 vip 寻址方式，有可能会导致后端不同实例连接数不均衡。
trpc-go/trpc-database/redis 也是这种模式，所以使用腾讯云 redis 时，不要使用 vip 寻址，应该尽量使用北极星寻址。

### IO 复用

client 在同一个连接上同时发送多个请求，每个请求通过序列号 ID 进行区分，client 与每个下游服务的节点都会建立一个长连接，默认所有的请求都是通过这个长连接来发送给服务端，需要服务端支持连接复用模式。IO 复用能够极大的减少服务之间的连接数量，但是由于 TCP 的头部阻塞，当同一个连接上并发的请求的数量过多时，会带来一定的延时（几毫秒级别），可以通过增加 IO 复用的连接数量（IO 复用默认一个 ip 建立两个连接）来一定程度上减轻这个问题。

使用场景：对稳定性和吞吐量有极致要求的场景。需要服务端支持单连接异步并发处理，和通过序列号 ID 来区分请求的能力，对 server 能力和协议字段都有一定的要求。
注意：

- 因为 IO 复用对每个后端节点只会建立 1 个连接，如果后端是 vip 寻址方式（从 client 角度看只有一个实例），不可使用 IO 复用，必须使用 2.2 连接池模式。
- 被调 server（注意不是你当前这个服务，是被你调用的服务）必须支持 io 复用，即在一个连接上对每个请求异步处理，多发多收，否则，client 这边会出现大量超时失败。tRPC-Go server 只在 v0.5.0 以上版本才支持。

### HTTP 连接池 (tRPC-Go >= v0.19.0)

HTTP 连接池是基于 `net/http` 的连接池实现的。当 client 使用 HTTP transport 时，可以利用 HTTP 连接池来管理和复用连接。

使用场景：适用于客户端使用 HTTP transport 的场景。

client 使用 HTTP transport 有两种方式：

- 显式指定使用 HTTP transport。
- 如果 protocol 设置为 HTTP，则默认使用 HTTP transport。

## 示例

### 短连接

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    // 禁用默认的连接池，则会采用短连接模式
    client.WithDisableConnectionPool(),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error(err.Error())
    return 
}

log.Info("req: %v, rsp: %v, err: %v", req, rsp, err)
```

### 连接池

```go
// 默认采用连接池模式，不需要任何配置
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error(err.Error())
    return 
}

log.Info("req: %v, rsp: %v, err: %v", req, rsp, err)
```

#### 自定义连接池

```go
import "git.woa.com/trpc-go/trpc-go/pool/connpool"

/*
连接池参数
type Options struct {
    MinIdle             int            // 最小空闲连接数量，由连接池后台周期性补充，0 代表不做补充
    MaxIdle             int            // 最大空闲连接数量，0 代表不做限制，框架默认值 65535
    MaxActive           int            // 用户可用连接的最大并发数，0 代表不做限制
    Wait                bool           // 可用连接达到最大并发数时，是否等待，默认为 false, 不等待
    IdleTimeout         time.Duration  // 空闲连接超时时间，0 代表不做限制，框架默认值 50s
    MaxConnLifetime     time.Duration  // 连接的最大生命周期，0 代表不做限制
    DialTimeout         time.Duration  // 建立连接超时时间，框架默认值 200ms
    ForceClose          bool           // 用户使用连接后是否强制关闭，默认为 false, 放回连接池
    PushIdleConnToTail  bool           // 放回连接池时的方式，默认为 false, 采用 LIFO 获取空闲连接
}
*/

// 连接池参数可以通过 option 设置，具体请查看 trpc-go 的文档，连接池需要设置成全局变量。
var pool = connpool.NewConnectionPool(connpool.WithMaxIdle(65535))
// 默认采用连接池模式，不需要任何配置
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    // 设置自定义连接池
    client.WithPool(pool),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error(err.Error())
    return 
}

log.Info("req: %v, rsp: %v, err: %v", req, rsp, err)
```

#### IO 复用

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    // 开启连接多路复用
    client.WithMultiplexed(true),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error(err.Error())
    return 
}

log.Info("req: %v, rsp: %v, err: %v", req, rsp, err)
```

#### 通过 WithOption 自定义 IO 复用

```go
import "git.code.oa.com/trpc-go/trpc-go/pool/multiplexed"

// IO 复用参数可以通过 `option` 设置，具体请查看 trpc-go 的文档。
// v0.18.4 之后可以通过 `WithInitialBackoff` 和 `WithMaxReconnectCount` 设置重连策略。
// 默认重连避让策略为线性避让。
var m = multiplexed.New(
    multiplexed.WithConnectNumber(16), // 将每个地址的连接数设置为 16，默认值为 2
    multiplexed.WithQueueSize(2048), // 将发送队列的长度设置为 2048，默认值为 1024
    multiplexed.WithDropFull(true), // 使队列满时丢弃请求，默认值为 false
    multiplexed.WithDialTimeout(2*time.Second), // 设置连接超时时间为 2s，默认值为 1s
    multiplexed.WithMaxVirConnsPerConn(5), // 设置每个实际连接的最大虚拟连接数为 5，默认值为 0，0 代表无限制
    multiplexed.WithMaxIdleConnsPerHost(10), // 设置每个地址的最大空闲连接数为 10，默认值为 0，0 代表禁用
    multiplexed.WithMaxReconnectCount(20), // 设置最大重连尝试次数为 20，默认值为 10，0 代表禁止重连
    multiplexed.WithInitialBackoff(10*time.Second), // 设置初始退避时间为 10s，默认值为 5ms
    multiplexed.WithReconnectCountResetInterval(600*time.Second) // 设置重置间隔为 600s，默认是两倍的 sum(dialTimeout) + sum(backoff)
) 

opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    // 开启连接多路复用
    client.WithMultiplexed(true),
    client.WithMultiplexedPool(m),
}

```

#### 通过文件配置自定义 IO 复用

v0.18.5 之后可以配置文件设置 `InitialBackoff` 和 `MaxReconnectCount`。

```yaml
client:
  service:
    - name: trpc.test.helloworld.Greeter1
      multiplexed:
        multiplexed_dial_timeout: 1s  # multiplexed: dial timeout, default 1s.
        conns_per_host: 2  # multiplexed: number of concrete(real) connections for each host, default 2.
        max_vir_conns_per_conn: 0  # multiplexed: max number of virtual connections for each concrete(real) connection, default 0 (means no limit).
        max_idle_conns_per_host: 0  # multiplexed: max number of idle concrete(real) connections for each host, used together with max_vir_conns_per_conn, default 0 (disabled).
        queue_size: 1024  # multiplexed: size of send queue for each concrete(real) connection, default 1024.
        drop_full: false  # multiplexed: whether to drop the send package when queue is full, default false.
        max_reconnect_count: 10 # multiplexed: the maximum number of reconnection attempts, 0 means reconnect is disable, default 10.
        initial_backoff: 5ms # multiplexed: the initial backoff time during the first reconnection attempt, default 5ms.

        multiplexed_dial_timeout: 1s  # 多路复用：拨号超时时间，默认 1s
        conns_per_host: 2  # 多路复用：每个主机的具体（实际）连接数，默认 2
        max_vir_conns_per_conn: 0  # 多路复用：每个具体（实际）连接的最大虚拟连接数，默认 0（表示无限制）
        max_idle_conns_per_host: 0  # 多路复用：每个主机的最大空闲具体（实际）连接数，与 max_vir_conns_per_conn 一起使用，默认 0（禁用）
        queue_size: 1024  # 多路复用：每个具体（实际）连接的发送队列大小，默认 1024
        drop_full: false  # 多路复用：当队列满时是否丢弃发送包，默认 false
        max_reconnect_count: 10  # 多路复用：最大重连次数，0 表示禁用重连，默认 10，适用于版本 >= v0.18.5
        initial_backoff: 5ms  # 多路复用：第一次重连尝试的初始退避时间，默认 5ms，适用于版本 >= v0.18.5
        reconnect_count_reset_interval: 600s # 多路复用：重连次数重置间隔，适用于版本 >= v0.19.0
```

### HTTP 连接池 (tRPC-Go >= v0.19.0)

```go
// 使用默认 HTTP 连接池配置
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    client.WithProtocol("http"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error(err.Error())
    return 
}

log.Info("req: %v, rsp: %v, err: %v", req, rsp, err)
```

#### 自定义连接池

```go
httpOpts := transport.HTTPRoundTripOptions{
    Pool: httppool.Options{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        MaxConnsPerHost:     20,
        IdleConnTimeout:     time.Second,
    },
}
opts := []client.Option{
    client.WithNamespace("Development"),
    client.WithServiceName("trpc.app.server.service"),
    client.WithProtocol("http"),
    // 设置 HTTP 连接池参数
    client.WithHTTPRoundTripOptions(httpOpts),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}

rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error(err.Error())
    return 
}

log.Info("req: %v, rsp: %v, err: %v", req, rsp, err)
```

## FAQ

请查看客户端开发向导的 [FAQ](https://iwiki.woa.com/p/284289117#10-faq) 部分。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
