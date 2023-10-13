[English](README.md) | 中文

## 背景

tRPC 框架间支持多种通信方式，如 tcp、udp 等。对于 udp 协议，一个 udp 包就对应一个 RPC 请求或回包。对于 tcp 这样的流式协议，就需要框架额外做分包处理。为了隔离不同网络协议间的差异，tRPC-Go 提供了 transport 抽象。

## 原理

在 tRPC-Go 中：

- client transport 负责和对端建立连接，并提供 multiplexed 等高级特性；
- server transport 负责建立监听套接字并 accept 新连接请求，并处理连接上到达的请求。

对流式 RPC，tRPC-Go 也提供了一套 transport 抽象。

下面我们来依次介绍这些 transport 的设计与实现。

## ClientTransport

[ClientTransport](transport.go) 的接口定义如下：

```go
type ClientTransport interface {
    RoundTrip(ctx context.Context, req interface{}, opts ...RoundTripOption) (rsp interface{}, err error)
}
```

`RoundTrip` 方法实现了请求的发送与接收。它支支持多种连接模式，如连接池、多路复用。支持高性能网络库 tnet。可以通过 [`RoundTripOptions`](client_roundtrip_options.go) 设置它们，比如：

```go
rsp, err := transport.RoundTrip(ctx, req,
	transport.WithDialNetwork("tcp"),
    transport.WithDialAddress(":8888"),
	transport.WithMultiplexed(true))
```

## ServerTransport

[ServerTransport](transport.go) 的接口定义如下：

```go
type ServerTransport interface {
    ListenAndServe(ctx context.Context, opts ...ListenServeOption) error
}
```

和 client 的 `RoundTripOptions` 对应，server 也可通过 [`ServerTransportOptions`](server_listenserve_options.go)，设置异步处理、空闲超时、tls 证书等：

```go
st := transport.NewServerTransport(transport.WithServerAsync(true))
```

## ClientStreamTransport

[ClientStreamTransport](transport_stream.go) 用于发送/接收流式请求。因为 stream 是 client 发起创建的，所以，它提供了 `Init` 方法来对流进行初始化，比如与对端建立网络连接。

```go
type ClientStreamTransport interface {
    Send(ctx context.Context, req []byte, opts ...RoundTripOption) error
    Recv(ctx context.Context, opts ...RoundTripOption) ([]byte, error)
    Init(ctx context.Context, opts ...RoundTripOption) error
    Close(ctx context.Context)
}
```

client stream transport 用了与普通 RPC transport 相同的 `RoundTripOption`，它底层的连接也支持多路复用等。

## ServerStreamTransport

[ServerStreamTransport](transport_stream.go) 用于服务端处理流式请求。当 Server 端收到 client 的 Init 包之后，它会创建一个新协程运行用户业务逻辑，而原始的网络收包协程则负责将收到的包分发给新协程。

```go
type ServerStreamTransport interface {
	ServerTransport
	Send(ctx context.Context, req []byte) error
	Close(ctx context.Context)
}
```

注意，ServerStreamTransport embedding 了 `ServerTransport` 用于监听端口并创建对应的网络协程。所以，普通 RPC 的 `ListenServeOption` 对流式 server 也适用。

## 分包

tRPC 的包都由帧头、包头、包体组成。在 server 收到请求和 client 收到回包时（流式请求也适用），需要对原始数据流分割成一个个请求，然后交给对应的处理逻辑。[`codec.FramerBuild`](/codec/framer_builder.go) 和 [`codec.Framer`](/codec/framer_builder.go) 就是用来对数据流进行分包的。

在 client 端，可以通过 [`WithClientFramerBuilder`](client_roundtrip_options.go) 设置 frame builder，在 server 端，可以通过 [`WithServerFramerBuilder`](server_listenserve_options.go) 设置。
