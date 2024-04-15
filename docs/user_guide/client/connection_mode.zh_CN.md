[English](connection_mode.md) | 中文

# tRPC-Go 客户端连接模式


# 前言

目前 tRPC-Go client，也就是请求发起的一方支持多种连接模式，包括短连接，连接池以及连接多路复用。client 默认使用连接池模式，用户可以根据自己的需要选择不同的连接模式。
`注意：这里的连接池指的是 tRPC-Go 自己实现的 transport 里面的连接池，database 以及 http 都是使用插件模式将 transport 替换成开源库，不是使用这里的连接池。`

# 原理和实现

### 短连接

client 每次请求都会新建一个连接，请求完成后连接会被销毁。请求量很大的情况下，服务的吞吐量会受到很大的影响，性能损耗也很大。

使用场景：一次性请求或者请求的被调服务是老服务，不支持在一个连接上接受多个请求的情况下使用。

### 连接池

client 针对每个下游 ip 都会维护一个连接池，每次请求先从名字服务获取一个 ip，根据 ip 获取对应连接池，再从连接池中获取一个连接，请求完成后连接会被放回连接池，在请求过程中，这个连接是独占的，不可复用的。连接池内的连接按照策略进行销毁和新建。一次调用绑定一个连接，当上下游规模很大的情况下，网络中存在的连接数以 MxN 的速度扩张，带来巨大的调度压力和计算开销。

使用场景：基本所有的场景都可以使用。
注意：因为连接池队列的策略是先进后出，如果后端是 vip 寻址方式，有可能会导致后端不同实例连接数不均衡。此时应该尽可能基于名字服务进行寻址。

### 连接多路复用

client 在同一个连接上同时发送多个请求，每个请求通过序列号 ID 进行区分，client 与每个下游服务的节点都会建立一个长连接，默认所有的请求都是通过这个长连接来发送给服务端，需要服务端支持连接复用模式。IO 复用能够极大的减少服务之间的连接数量，但是由于 TCP 的头部阻塞，当同一个连接上并发的请求的数量过多时，会带来一定的延时（几毫秒级别），可以通过增加连接多路复用的连接数量（IO 复用默认一个 ip 建立两个连接）来一定程度上减轻这个问题。

使用场景：对稳定性和吞吐量有极致要求的场景。需要服务端支持单连接异步并发处理，和通过序列号 ID 来区分请求的能力，对 server 能力和协议字段都有一定的要求。
注意：

- 因为连接多路复用对每个后端节点只会建立 1 个连接，如果后端是 vip 寻址方式（从 client 角度看只有一个实例），不可使用连接多路复用，必须使用连接池模式。
- 被调 server（注意不是你当前这个服务，是被你调用的服务）必须支持连接多路复用，即在一个连接上对每个请求异步处理，多发多收，否则，client 这边会出现大量超时失败。

# 示例

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

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
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

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```

自定义连接池

```go
import "trpc.group/trpc-go/trpc-go/pool/connpool"

/*
连接池参数
type Options struct {
	MinIdle             int			  	// 最小空闲连接数量，由连接池后台周期性补充，0 代表不做补充
	MaxIdle             int           	// 最大空闲连接数量，0 代表不做限制，框架默认值 65535
	MaxActive           int           	// 用户可用连接的最大并发数，0 代表不做限制
	Wait                bool          	// 可用连接达到最大并发数时，是否等待，默认为 false, 不等待
	IdleTimeout         time.Duration 	// 空闲连接超时时间，0 代表不做限制，框架默认值 50s
	MaxConnLifetime     time.Duration 	// 连接的最大生命周期，0 代表不做限制
	DialTimeout         time.Duration 	// 建立连接超时时间，框架默认值 200ms
	ForceClose          bool          	// 用户使用连接后是否强制关闭，默认为 false, 放回连接池
	PushIdleConnToTail  bool			// 放回连接池时的方式，默认为 false, 采用 LIFO 获取空闲连接
}
*/

// 连接池参数可以通过 option 设置，具体请查看 trpc-go 的文档，连接池需要设置成都是全局变量。
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

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```

#### 设置空闲连接超时

对于客户端的连接池模式来说，框架会设置一个默认的 50s 的空闲超时时间。

* 对于 `go-net` 而言，连接池中会维持一个空闲连接列表，空闲超时时间只会对空闲连接列表中的连接生效，并且只会在下次获取的时候触发空闲连接触发空闲超时的关闭
* 对于 `tnet` 而言，空闲超时通过在每个连接上维护定时器来实现，即使该连接被用于客户端发起调用，假如下游未在空闲连接超时时间内返回结果的话，该连接仍然会被触发空闲超时并强制被关闭

更改空闲超时时间的方式如下：

* `go-net`

```go
import "trpc.group/trpc-go/trpc-go/pool/connpool"

func init() {
	connpool.DefaultConnectionPool = connpool.NewConnectionPool(
		connpool.WithIdleTimeout(0), // 设置为 0 是禁用
	)
}
```

* `tnet`

```go
import (
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func init() {
	tnettrans.DefaultConnPool = connpool.NewConnectionPool(
	      connpool.WithDialFunc(tnettrans.Dial),
	      connpool.WithIdleTimeout(0), // 设置为 0 是禁用
	      connpool.WithHealthChecker(tnettrans.HealthChecker),
      )
}
```

**注**：服务端默认也有一个空闲超时时间，为 60s，该时间设计得比 50s 打，从而在默认情况下是客户端主动触发空闲连接超时以主动关闭连接，而非服务端触发强制清理。服务端空闲超时的更改方法见服务端使用文档。

### 连接多路复用

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

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```

设置自定义连接多路复用

```go
/*
type PoolOptions struct {
    connectNumber int  // 设置每个地址的连接数
    queueSize     int  // 设置每个连接请求队列长度
    dropFull      bool // 队列满是否丢弃
}
*/
// 连接多路复用参数可以通过 option 设置，具体请查看 trpc-go 的文档，需要设置成都是全局变量。
var m = multiplexed.New(multiplexed.WithConnectNumber(16))

opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
		// 开启连接多路复用
		client.WithMultiplexed(true),
		client.WithMultiplexedPool(m),
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

log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
```
