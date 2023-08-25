[TOC]

# tRPC-Go 模块：client





### 背景

client 用于发起网络调用，client 的职责主要包括，编解码，服务发现，负载均衡，熔断等。client 操作中的每个步骤都是可以自定义的，如自定义编解码，自定义服务发现方式，自定义负载均衡算法等。
要做到这些，并且方便后续业务方扩展，需要提供一个通用的 client，它需要支持比较完备的上述操作，但是又需要提供适当的接口来允许扩展。

这就是 tRPC-Go client 的设计初衷和目标。

### 原理

先来看下 client 的整体设计及与其他层次的关系：

![relation.png](/.resources/module_design/client/client_zh_CN.png)

接下来结合这里的类图，来进一步描述下。

全局默认使用同一个 client，并且是并发安全的。可以通过参数指定传输协议，编解码类型，路由选择器，服务发现，负载均衡，熔断器等，并且支持自定义。只要按照框架的接口实现就可以无缝和框架进行整合。

完整的示例请参考 [helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld) 。

### 接口定义

```go
// Client 客户端调用结构
type Client interface {
    //发起后端调用
    Invoke(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...Option) (err error)
}
```

Client 的定义主要通过一个参数选项 Option 去发起后端调用

```go
// Options 客户端调用参数
type Options struct {
    Namespace   string        // 服务发现需要指定 Namespace
    ServiceName string        // 后端服务 servic name
    Target      string        // 后端服务地址 默认使用北极星，可支持其他 target，name://endpoint
    Timeout     time.Duration // 后端调用超时时间
    endpoint    string        // 默认等于 service name，除非有指定 target
    checkerSet  bool          // Transport 读包有 MsgReader 和 Checker 两种方式，两种方式互斥。checkerSet 标识是否设置了读包方式
    CallOptions    []transport.RoundTripOption // client transport 需要调用的参数
    Transport      transport.ClientTransport
    Codec          codec.Codec
    Selector       selector.Selector
    LoadBalancer   loadbalance.LoadBalancer
    Discovery      discovery.Discovery
    CircuitBreaker circuitbreaker.CircuitBreaker
    Filters filter.Chain
    ReqHead interface{}
    RspHead interface{}
    Node    *registry.Node
}
// Option 调用参数工具函数
type Option func(*Options)

```

### client 初始化

使用 [trpc](https://git.woa.com/trpc-go/trpc-go-cmdline) 代码生成工具生成服务的代码包含客户端对应的代码。client 初始化

```go
opts := []client.Option{
    client.WithProtocol("trpc"),
    client.WithNetwork("tcp4"),
    client.WithTarget("ip://10.100.72.229:12367"),
}
proxy := pb.NewGreeterClientProxy()
req := &pb.HelloRequest{}
rsp, err := proxy.SayHello(ctx, req, opts...)
```

上面的代码片段通过 `Option` 参数设置编解码采用 trpc 协议，传输使用 tcp 协议，并且通过 target 指定 rpc server 地址为 `10.100.72.229:12367`。target 含义见下面的路由选择器。

### 网络传输

trpc-go 支持不同传输协议，目前支持 `tcp4`、`tcp6`、`udp4`、`udp6`、后续将会支持 `quic` `rdma`等传输协议。框架默认传输层支持 trpc 和 http 协议，可通过 `client.WithNetwork("udp")` 来确定使用 tcp 还是 udp。下面代码片段支持

```go
opts := []client.Option{
    client.WithNetwork("udp"),
    client.WithTarget("ip://10.100.72.229:12367"),
}
proxy := pb.NewGreeterClientProxy()
req := &pb.HelloRequest{}
rsp, err := clientProxy.SayHello(ctx, req, opts...)
```

### 协议

trpc-go 默认支持 trpc 和 http 协议，可以通过

```go
client.WithProtocol("trpc"),
```

进行设置。同时也可以设置自定义协议。具体第三方协议可参考[这里](https://git.woa.com/trpc-go/trpc-codec)

### 路由选择器

路由选择器，是基于 NamingService 的负载均衡实现，集成了 `服务发现，负载均衡，熔断器` 的功能，client 通过可插拔的方式支持不同类型的路由方式，并且支持自定义。
框架默认使用[北极星](https://git.woa.com/trpc-go/trpc-naming-polaris)，目前已经实现的其他插件包括 [cmlb](https://git.woa.com/trpc-go/trpc-selector-cmlb)，[cl5](https://git.woa.com/trpc-go/trpc-selector-cl5)

使用 cmlb 作为路由方式如下。

```go
opts := []client.Option{
    client.WithNetwork("tcp4"),
    client.WithTarget("cmlb://13702"),
}
proxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "client hello",
}
rsp, err := proxy.SayHello(ctx, req)
```

具体使用方式请参考，[trpc-selector-cmlb](https://git.woa.com/trpc-go/trpc-selector-cmlb)，[trpc-selector-cl5](https://git.woa.com/trpc-go/trpc-selector-cl5)。

trpc 支持整个路由的插件也支持单独的 `服务发现`，`负载均衡`，`熔断器`等的插件。
target 可以支持如下格式：

```ini
ip://ip:port
dns://domain:port
cmlb://appid
cl5://sid
ons://zkname
polaris://servicename
```

### 服务发现

用户根据自己的需要可以自定义服务发现类型，可以为 `etcd`, `zookeeper`, `dns` 等。通过访问服务发现 server 端可获取对应的服务地址列表，假设已经实现好了 trpc 的 etcd 服务发现插件，具体使用方式如下：

```go
opts := []client.Option{
    client.WithServiceName("ETCD-NAMING-TEST1"),
    client.WithDiscoveryName("etcd-discovery"),
}
clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "client hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
log.Printf("req:%v, rsp:%v, err:%v", req, rsp, err)
```

用户也已自定义实现其他类型的服务发现方式，discovery 详细[接口](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/discovery)。

### 负载均衡

服务发现返回服务器列表而不是单个地址的时候，需要使用负载均算法去决定使用哪个地址与后端进行通信。整个过程叫做负载均衡，trpc-go 目前采用客户端负载均衡方式。用户可以自定义负载均衡算法。框架提供的算法包括：

- round robin 选择服务器列表里面的下一个地址
- smooth weight round robin 平滑加权轮训算法
- random 随机选择服务器列表里面的地址

用户也可以自定义实现其他类型负载均衡算法，例如一致性 hash，动态权重等。详细[接口](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/loadbalance)。

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-go/naming/loadbalance/roundrobin"
)
opts := []client.Option{
    client.WithServiceName("ETCD-NAMING-TEST1"),
    client.WithDiscoveryName("etcd-discovery"),
    client.WithBalancerName("round_robin"),
}
clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "client hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
log.Printf("req:%v, rsp:%v, err:%v", req, rsp, err)
```