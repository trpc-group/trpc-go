English | [中文](metadata_transmission.zh_CN.md)

# tRPC-Go metadata transmission

## Introduction

Metadata transparent transmission: The framework supports transparent transmission of metadata between the client and server, and automatically transmits them throughout the entire call chain. The metadata is actually key-value pairs, where the key is of `string` type and the value is of `[]byte` type. The value can be any data. It provides additional information about the RPC request. The framework passes metadata through `context`.

The following document describes how to use metadata transparent transmission in the framework.

## Principle and Implementation

tRPC-Go transmits metadata through the transinfo field in the tRPC protocol header. You should set the metadata that need to be transmitted through the framework's API to the context. When the framework encodes request, it sets the metadata from user to the transinfo field of the protocol, and then transmits them. On the other side, the framework will parse metadata from the transinfo field when decods response, and you can obtain the metadata from the context.

## Example

#### Client transmits data to server

When the client sends a request, you can set the metadata by adding options. You can add multiple metadata.

```go
options := []client.Option{
    client.WithMetaData("key1", []byte("val1")),
    client.WithMetaData("key2", []byte("val2")),
    client.WithMetaData("key3", []byte("val3")),
}

rsp, err := proxy.SayHello(ctx, options...) // Note: ctx should be passed by the framework
```

The downstream server can obtain the metadata from the client through the framework's `ctx`.

```go
// Note: Use the ctx passed by the framework. 
// The client sets the value of key1 to val1, so the value will be val1. 
// If the client does not set the value for key1, an empty data will be returned.
trpc.GetMetaData(ctx, "key1") 
```

#### Server transmits data to client

When the server responds to the client, it can set the metadata through `ctx` to the upstream client.

```go
// Note: Use the ctx passed by the framework. 
// Set the value of the metadat key1 to []byte("val1") through this API.
trpc.SetMetaData(ctx, "key1", []byte("val1")) 
```

The upstream client can obtain it by setting the response head of each protocol.

```go
head := &trpc.ResponseProtocol{}
options := []client.Option{
    client.WithRspHead(head),
}

rsp, err := proxy.SayHello(ctx, options...) // Note: use ctx passed by the framework
head.TransInfo // metadata transmitted by the framework
```
