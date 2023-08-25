# tRPC-Go module: transport

## Background

In network, both parties communicate with each other using socket, and use socket to send and receive data. Sometimes, before sending or after receiving data, additional operations such as data compression, decompression, encoding, decoding, and connection reuse may be performed.  
Through transport, these logics can be encapsulated, although the transport in tRPC-Go is currently only used for data transmission.

Currently, transport is the underlying communication layer, only responsible for the most basic binary data network communication, without any business logic. By default, there is only one `ServerTransport` and one `ClientTransport` globally, and businesses can customize their own Transport implementation.

## Principle

Here is the UML class diagram corresponding to the transport layer:  

![UML](/.resources/developer_guide/module_design/transport/transport.png)

In tRPC-Go, the transport layer is mainly responsible for:

- The client transport is only responsible for completing the transmission and reception of data.
- The server transport is responsible for establishing a listening socket, accepting new connection requests, and handling requests that arrive on the connection.

trpc-go supports the implementation of the TCP and UDP protocols by default.

Next, let's take a detailed look at the design and implementation of the client transport and server transport.


## ClientTransport

Common interface for the client communication layer.


```go
type ClientTransport interface {
    RoundTrip(ctx context.Context, req interface{}, opts ...RoundTripOption) (rsp interface{}, err error)
}
```

`RoundTrip` implements the sending of client requests. It supports four modes: send-only, send-receive, long connection, and streaming transmission. You need to specify the `ReqType` in `RoundTripOptions` under the client package. The structure of `RoundTripOptions` is as follows:

```go
// RoundTripOptions
type RoundTripOptions struct {
    Address   string    // IP:Port. Note: At the transport layer, the name service resolution has already been performed, so it is directly in the format of IP:Port.
    Network   string    // tcp/udp
    Pool      pool.Pool // client connection pool
    MsgReader MsgReader // one of msgReader and checker
    Checker   Checker   // one of msgReader and checker
    ReqType   int       // SendAndRecv, SendOnly, Stream...
}
```

Set RoundTripOptions

```go
rsp, err := transport.RoundTrip(ctx, reqData, transport.WithDialNetwork(network),
    transport.WithDialAddress(":8888"),
    transport.WithClientMsgReader(&simpleMsgReader{}))
```

## ServerTransport

Common interface for the server communication layer. `ListenAndServe` is used to implement service listening and processing.

```go
type ServerTransport interface {
    ListenAndServe(ctx context.Context, opts ...ListenServeOption) error
}
```

Corresponding to `RoundTripOptions` of the client, the server can also set `ServerTransportOptions` as follows:

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

Set ServerTransportOptions

```go
st := transport.NewServerTransport(transport.WithReusePort(false),
    transport.WithKeepAlivePeriod(time.Minute))
```

## Parse Packet

### MsgReader

MsgReader is used to unpack data by parsing the packet format from the `io.Reader` and returning the packet data. The advantage is that memory does not need to be allocated in advance, provided that the packet size can be known in advance through the protocol.

```go
type MsgReader interface {
    ReadMsg(io.Reader) ([]byte, error)
}
```

### Checker

Checker is used to verify the packet by pre-allocating a large memory buffer, checking the data, and returning the packet size. It is very versatile, but the disadvantage is that it requires pre-allocation of memory, which has a greater impact on performance.

```go
type Checker interface {
    Check([]byte) (int, error)
}
```

MsgReader and Checker are mutually exclusive, and only one of them can be used. The framework currently defaults to using MsgReader for unpacking.

## Port reuse

Port reuse is implemented using the open source `go_reuseport` library. For more information, please refer to [go_reuseport](https://github.com/kavu/go_reuseport)

## How to use

In trpc-go, the client calls `RoundTrip` to send a request and receives a response.

```go
rspdata, err := opts.Transport.RoundTrip(ctx, reqbuf, opts.CallOptions...)
```

The server side starts the service listener: 

```go
if err = s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
    log.Errorf("service:%s ListenAndServe fail:%v", s.opts.ServiceName, err)
    return err
}
```

