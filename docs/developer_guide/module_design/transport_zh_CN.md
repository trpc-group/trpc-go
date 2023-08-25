[TOC]

# tRPC-Go 模块：transport

## 背景

网络通信双方借助 socket 进行通信，通信双方借助 socket 进行数据的发送、接收。在数据发送之前、接收数据之后，有时也会做一些其他额外的操作，如数据压缩、解压缩，编码解码，连接复用等等。
通过 transport 可以对这些逻辑进行一定的封装，尽管当前 tRPC-Go 中的 transport 只用来做数据的收发。

当前，transport 即底层网络通讯层，只负责最基本的二进制数据网络通信，没有任何业务逻辑。默认全局只会有一个 ServerTransport 和一个 ClientTransport，支持业务自定义实现 Transport。

## 原理

以下是 transport 层对应的 UML 类图：

![UML](/.resources/developer_guide/module_design/transport/transport.png)

tRPC-Go 中 transport 主要负责：

- client transport 只负责完成数据的收发；
- server transport 负责建立监听套接字并 accept 新连接请求，并处理连接上到达的请求；

trpc-go 默认支持 tcp 和 udp 协议的实现。

下面我们来详细看下 client transport 和 server transport 的设计、实现。

## ClientTransport

client 通讯层通用接口

```go
type ClientTransport interface {
    RoundTrip(ctx context.Context, req interface{}, opts ...RoundTripOption) (rsp interface{}, err error)
}
```

`RoundTrip` 实现 client 请求的发送。支持只发不收、一发一收、长连接 和 流式传输 4 种模式，需要指定 client 包下 `RoundTripOptions` 的 `ReqType` ，`RoundTripOptions` 结构如下：

```go
// RoundTripOptions
type RoundTripOptions struct {
    Address   string    // IP:Port. 注意：到了 transport 层，已经过了名字服务解析，所以直接就是 IP:Port
    Network   string    // tcp/udp
    Pool      pool.Pool // client 连接池
    MsgReader MsgReader // msgReader 与 checker 两者取其一
    Checker   Checker   // msgReader 与 checker 两者取其一
    ReqType   int       // SendAndRecv, SendOnly, Stream...
}
```

设置 RoundTripOptions

```go
rsp, err := transport.RoundTrip(ctx, reqData, transport.WithDialNetwork(network),
    transport.WithDialAddress(":8888"),
    transport.WithClientMsgReader(&simpleMsgReader{}))
```

## ServerTransport

server 通讯层通用接口，`ListenAndServe` 实现服务监听和处理

```go
type ServerTransport interface {
    ListenAndServe(ctx context.Context, opts ...ListenServeOption) error
}
```

和 client 的 `RoundTripOptions` 对应，server 也可设置 `ServerTransportOptions`，如下：

```go
// ServerTransportOptions
type ServerTransportOptions struct {
    RecvMsgChannelSize      int
    SendMsgChannelSize      int
    RecvUDPPacketBufferSize int
    IdleTimeout             time.Duration
    KeepAlivePeriod         time.Duration
}
```

设置 ServerTransportOptions

```go
st := transport.NewServerTransport(transport.WithReusePort(false),
    transport.WithKeepAlivePeriod(time.Minute))
```

## 解包

### MsgReader

MsgReader 拆包，从 `io.Reader` 里面解析出包格式，并返回包数据，好处是不用预先分配好内存，前提是通过协议可以提前知道包大小

```go
type MsgReader interface {
    ReadMsg(io.Reader) ([]byte, error)
}
```

### Checker

Checker 验包，预先分配大内存 buf，校验数据并返回包大小，很通用，但是缺点是需要预先分配内存，对性能影响较大

```go
type Checker interface {
    Check([]byte) (int, error)
}
```

MsgReader 和 Checker 两种方式互斥，二者只能取其一，框架目前默认支持 MsgReader 进行解包。

## 端口复用

端口复用使用开源 `go_reuseport` 实现，具体可参考：[go_reuseport](https://github.com/kavu/go_reuseport)

## 使用

trpc-go 中 client 调用 `RoundTrip` 发送请求，得到回包

```go
rspdata, err := opts.Transport.RoundTrip(ctx, reqbuf, opts.CallOptions...)
```

server 端启动服务监听：

```go
if err = s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
    log.Errorf("service:%s ListenAndServe fail:%v", s.opts.ServiceName, err)
    return err
}
```
