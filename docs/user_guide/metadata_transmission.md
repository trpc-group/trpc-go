# Introduction
Metadata transparent transmission: The framework supports transparent transmission of metadata fields between the client and server, and automatically transmits them throughout the entire call chain. The fields exist in the form of key-value pairs, where the key is of string type and the value is of `[]byte` type. The value can be any data. The transparent transmission of fields is transparent to RPC requests and provides additional information about the RPC request. At the same time, the framework passes metadata fields through `ctx`.

The following document describes how to implement metadata transparent transmission in the framework.

# Principle and Implementation
To transmit metadata through the transinfo field in the tRPC protocol header, you should set the fields that need to be transmitted through the framework's API to the context. When the framework packs and unpacks, it sets the fields set by the user to the corresponding fields of the protocol, and then transmits them. The receiving party will parse the corresponding transmitted fields, and you can obtain the transmitted data through the interface.

# Example
## Client transmits data to server
When the client initiates a request, it can set the transmitted fields by adding options. Multiple fields can be added.

```go
options := []client.Option{
	client.WithMetaData("key1", []byte("val1")),
	client.WithMetaData("key2", []byte("val2")),
	client.WithMetaData("key3", []byte("val3")),
}

rsp, err := proxy.SayHello(ctx, options...) // Note: ctx passed by the framework
```

The downstream server can obtain the transmitted fields of the client through the framework's `ctx`.

```go
trpc.GetMetaData(ctx, "key1") // Note: Use the framework's ctx. The client sets the value of key1 to val1. Here, the return value will be val1. If the client does not set the corresponding value, an empty data will be returned.
```

## Server transmits data to client
When the server responds to the request, it can set the transmitted fields through `ctx` to return to the upstream caller.

```go
trpc.SetMetaData(ctx, "key1", []byte("val1")) // Note: Use the framework's ctx. Set the value of the transmitted field key1 to []byte("val1") through this API.
```

The upstream client can obtain it by setting the response head of each protocol.

```go
head := &trpc.ResponseProtocol{}
options := []client.Option{
	client.WithRspHead(head),
}

rsp, err := proxy.SayHello(ctx, options...) // Note: ctx passed by the framework
head.TransInfo // Key-value pairs of information transmitted by the framework (map[string][]byte)
```

