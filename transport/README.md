English | [中文](README.zh_CN.md)

## Background

tRPC frameworks support multiple network protocols, such as tcp, udp, etc. For the udp protocol, a udp packet corresponds to an RPC request or response. For streaming protocols such as tcp, the framework requires additional package splitting mechanism. In order to isolate the differences between different network protocols, tRPC-Go provides a transport abstraction.

## Principle

In tRPC-Go:

- The client transport is responsible for establishing a connection with the peer and providing advanced features such as multiplexed;
- The server transport is responsible for listening on socket, accepting incoming connections, and processing requests arriving at the connection.

For streaming RPC, tRPC-Go also provides a set of transport abstractions.

Below we will introduce the design and implementation of these transports in turn.

## ClientTransport

The interface definition of [ClientTransport](client_transport.go) is as follows:

```go
type ClientTransport interface {
    RoundTrip(ctx context.Context, req interface{}, opts ...RoundTripOption) (rsp interface{}, err error)
}
```

The `RoundTrip` method implements the sending and receiving of requests. It supports multiple connection modes, such as connection pooling and multiplexing. It also supports high-performance network library tnet. These options can be set via [`RoundTripOptions`](client_roundtrip_options.go), for example:

```go
rsp, err := transport.RoundTrip(ctx, req,
    transport.WithDialNetwork("tcp"),
    transport.WithDialAddress(":8888"),
    transport.WithMultiplexed(true))
```

## ServerTransport

The interface of [ServerTransport](transport.go) is defined as follows:

```go
type ServerTransport interface {
    ListenAndServe(ctx context.Context, opts ...ListenServeOption) error
}
```

Just like `RoundTripOptions` of client side, the server has [`ServerTransportOptions`](server_listenserve_options.go). It can be used to set asynchronous processing, idle timeout, tls certificate, etc.

```go
st := transport.NewServerTransport(transport.WithServerAsync(true))
```

## ClientStreamTransport

[ClientStreamTransport](transport_stream.go) is used to send/receive streaming requests. Because the stream is created by the client, it provides the `Init` method to initialize the stream, such as establishing a network connection with the peer.

```go
type ClientStreamTransport interface {
    Send(ctx context.Context, req []byte, opts ...RoundTripOption) error
    Recv(ctx context.Context, opts ...RoundTripOption) ([]byte, error)
    Init(ctx context.Context, opts ...RoundTripOption) error
    Close(ctx context.Context)
}
```

The client stream transport uses the same `RoundTripOption` as the ordinary RPC transport, and its underlying connection also supports multiplexing, etc.

## ServerStreamTransport

[ServerStreamTransport](transport_stream.go) is used for server-side processing of streaming requests. When the server receives the client's Init packet, it creates a new goroutine to run the user's business logic, and the original network packet receiving goroutine is responsible for dispatching the received packets to the new goroutine.

```go
type ServerStreamTransport interface {
    ServerTransport
    Send(ctx context.Context, req []byte) error
    Close(ctx context.Context)
}
```

Note that ServerStreamTransport embeds `ServerTransport`, which is used to listen on the port and create the corresponding network goroutine. Therefore, the `ListenServeOption` of ordinary RPC is also applicable to the streaming server.

## Split Package

tRPC packets are composed of frame header, packet header, and packet body. When the server receives the request or the client receives the response packet (streaming requests are also applicable), the original data stream needs to be divided into individual requests and then handed over to the corresponding processing logic. [`codec.FramerBuild`](/codec/framer_builder.go) and [`codec.Framer`](/codec/framer_builder.go) are used to split the data stream.

On the client side, you can set the frame builder through [`WithClientFramerBuilder`](client_roundtrip_options.go). On the server side, you can set it through [`WithServerFramerBuilder`](server_listenserve_options.go).
