English | [中文](README.zh_CN.md)

# Overview

The `codec` module provides interfaces related to encoding and decoding, allowing the framework to extend business protocols, serialization methods, and data compression methods.

# Analysis of Core Concepts

The main concepts in the `codec` module include interfaces such as `Msg`, `Framer`, `Codec`, `Serializer`, and `Compressor`, which we will introduce in turn.

- `Msg`: The common message body for each request. To support arbitrary third-party protocols, this interface has been abstracted out in tRPC to carry the basic information needed by the framework. The `msg` struct is the sole implementation of this interface.

Before introducing the remaining interfaces, let's first use two diagrams to show the protocol processing flow on the server and client sides, so that readers can gain an overall understanding.

Server-side processing flow

```text
                       package                   req body                                                       req struct
+-------+   +--------+  []byte  +--------------+  []byte   +-----------------------+   +----------------------+
|       +-->| Framer +--------->| Codec-Decode +---------->| Compressor-Decompress +-->| Serializer-Unmarshal +------------+
|       |   +--------+          +--------------+           +-----------------------+   +----------------------+            |
|       |                                                                                                             +----v----+
|network|                                                                                                             | Handler |
|       |                                        rsp body                                                             +----+----+
|       |         package                         []byte                                                       rsp struct  |
|       |          []byte       +--------------+            +---------------------+     +--------------------+             |
|       <-----------------------+ Codec-Encode |<-----------+ Compressor-Compress |<----+ Serializer-Marshal |<------------+
+-------+                       +--------------+            +---------------------+     +--------------------+
```

Client-side processing flow

```text
req struct                                                          req body                    package  +-------+
            +--------------------+          +---------------------+  []byte  +--------------+    []byte  |       |
----------->| Serializer-Marshal +--------->| Compressor-Compress +--------->| Codec-Encode +----------->|       |
            +--------------------+          +---------------------+          +--------------+            |       |
                                                                                                         |network|
                                                                                                         |       |
                                                                     rsp body                   package  |       |
rsp struct +----------------------+        +-----------------------+  []byte +--------------+    []byte  |       |
<----------| Serializer-Unmarshal |<-------+ Compressor-Decompress |<--------+ Codec-Decode |<-----------+       |
           +----------------------+        +-----------------------+         +--------------+            +-------+
```

The interfaces involved in the above flowchart from the `codec` are described in detail below. Readers can read in conjunction with the diagrams.

- `Framer`: The interface for reading a complete business packet from binary data received from the network.

    ```go
    type Framer interface {
        ReadFrame() ([]byte, error)
    }
    ```

- `Codec`: The business protocol packing and unpacking interface. The business protocol is divided into a header and a body. Here, it is only necessary to parse out the binary body; the header is generally placed inside the `msg`, and the business does not need to worry about it.

    ```go
    type Codec interface {
        // server unpacking => Parsing the binary request packet body from the complete binary network data packet.
        // client unpacking => Parsing the binary response packet body from the complete binary network data packet.
        Decode(message Msg, buffer []byte) (body []byte, err error)

        // server packing => Packaging the binary response packet body into a complete binary network data packet.
        // client packing => Packaging the binary request packet body into a complete binary network data packet.
        Encode(message Msg, body []byte) (buffer []byte, err error)
    }
    ```

- `Serializer`: The packet body serializing and deserializing interface. Currently supported are protobuf, json, jce, flatbuffers, and xml. Users can also define their own `Serializer` and register it with the `codec` package.

    ```go
    type Serializer interface {
        // server unpacks the binary package body => Then it calls this method to parse into the specific request structure.
        // client unpacks the binary package body => Then it calls this method to parse into the specific response structure.
        Unmarshal(in []byte, body interface{}) error

        // server responds with a response structure => Then it calls this method to convert it into a binary package body.
        // client sends a request structure => Then it calls this method to convert it into a binary package body.
        Marshal(body interface{}) (out []byte, err error)
    }
    ```

- `Compressor`: The packet body compressing and decompressing interface. Currently supported are gzip, lz4, snappy, and zlib. Users can also define their own `Compressor` and register it with the `codec` package.

    ```go
    type Compressor interface {
        // server/client calls this method after unpacking the binary package body => Decompress to obtain the original binary data.
        Decompress(in []byte) (out []byte, err error)

        // server/client calls this method before packing the binary package body => Compress it into smaller binary data.
        Compress(in []byte) (out []byte, err error)
    }
    ```

# How to Implement a Business Protocol

## Basic Steps

To implement a business protocol, at least the following three steps need to be taken:

1. Implement the `Framer` and `FramerBuilder` interfaces to read complete business packets from the connection.

2. Implement the `Codec` business protocol packing and unpacking interface.

3. Register the specific implementation in the `init` function to the tRPC framework.

In addition to these three steps, it may also be necessary to implement the `Serializer` and `Compressor` interfaces(Generally speaking, serialization and compression have standard formats available for use. Readers can read and directly use several serialization and compression methods already implemented in the `codec` package).

## Precautions

In the second step of the implementation process, the following contents also need to be noted (Values that are not present do not need to be set. For specific usage of these interfaces, please refer to the implementation of [oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb)):

- The interfaces that needs to be called after the server `Codec` decodes the request packet:
  - Use `msg.WithServerRPCName` to tell tRPC how to route `/trpc.app.server.service/method`.
  - Use `msg.WithRequestTimeout` to specify the remaining timeout time for the upstream service.
  - Use `msg.WithSerializationType` to specify the serialization method.
  - Use `msg.WithCompressType` to specify the decompression method.
  - Use `msg.WithCallerServiceName` to set the upstream service name `trpc.app.server.service`.
  - Use `msg.WithCalleeServiceName` to set the name of the service itself.
  - Use `msg.WithServerReqHead` and `msg.WithServerRspHead` to set the business protocol header.

- The interfaces that needs to be called before the server `Codec` encodes the response packet:
  - Use `msg.ServerRspHead` to retrieve the response header and send it back to the client.
  - Use `msg.ServerRspErr` to convert the error returned by the handler function into a specific business protocol header error code.

- The interfaces that needs to be called before the client `Codec` encodes the request packet:
  - Use `msg.ClientRPCName` to specify the request routing.
  - Use `msg.RequestTimeout` to inform downstream services of the remaining timeout time.
  - Use `msg.WithCalleeApp` to set the downstream service.

- The interfaces that needs to be called after the client `Codec` decodes the response packet:
  - Use `errs.New` to convert specific business protocol error codes into an error to be returned to the user calling function.
  - Use `msg.WithSerializationType` to specify the serialization method.
  - Use `msg.WithCompressType` to specify the decompression method.

# A Simple Implementation Example

This section uses the rawstring protocol in [trpc-codec](https://git.woa.com/trpc-go/trpc-codec) as an example to demonstrate the specific steps of implementing a business protocol. For the specific code, please refer to [here](https://git.woa.com/trpc-go/trpc-codec/tree/master/rawstring).

## Protocol Introduction

The rawstring protocol is a simple TCP-based invocation protocol characterized by using the `'\n'` character as a delimiter for packet sending and receiving.

## Implement the `Framer` and `FramerBuilder` Interfaces

```go
type FramerBuilder struct{}

func (fd *FramerBuilder) New(reader io.Reader) transport.Framer {
    return &framer{
        reader: reader,
    }
}

type framer struct {
    reader io.Reader
}

func (f *framer) ReadFrame() (msg []byte, err error) {
    reader := bufio.NewReader(f.reader)
    // Unpacking using the '\n' character as the delimiter.
    return reader.ReadBytes('\n')
}
```

## Implement the `Codec` Interface

```go
// server-side Codec
type serverCodec struct{}

func (sc *serverCodec) Decode(_ codec.Msg, req []byte) ([]byte, error) {
    return req, nil
}

func (sc *serverCodec) Encode(_ codec.Msg, rsp []byte) ([]byte, error) {
    // The server adds a '\n' character after the response as a complete binary network data.
    return []byte(string(rsp) + "\n"), nil
}

// client-side Codec
type clientCodec struct{}

func (cc *clientCodec) Encode(_ codec.Msg, req []byte) ([]byte, error) {
    // The client adds a '\n' character after the request as a complete binary network data.
    return []byte(string(reqBody) + "\n"), nil
}

func (cc *clientCodec) Decode(_ codec.Msg, rsp []byte) ([]byte, error) {
    return rspBody, nil
}
```

## Register Implementation

```go
// Register the implemented FramerBuilder to the transport package.
var DefaultFramerBuilder = &FramerBuilder{}
func init() {
    transport.RegisterFramerBuilder("rawstring", DefaultFramerBuilder)
}

// Register the implemented Codec to the codec package.
func init() {
    codec.Register("rawstring", &serverCodec{}, &clientCodec{})
}
```

# Implementation of Various tRPC-Go Business Protocols

Various specific business protocols have been implemented in the repository [trpc-codec](https://git.woa.com/trpc-go/trpc-codec). The key points to note during implementation are as follows:

- By implementing the relevant interfaces in the codec, tRPC-Go can support any third-party business communication protocol.
- Each business protocol is a separate go module, which does not affect each other. When using the `go get` command, only the required codec module will be pulled.
- There are generally two typical styles of business protocols: IDL protocols (such as tars) and non-IDL protocols (such as oidb). For specific details, you can refer to the implementations of [tars](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars) and [oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb) respectively.

# Performance Optimization Guidelines

After v0.17.0, users can provide the `optimization` build tag when running `go build` to enable performance optimization, for example:

```shell
go build -tags=optimization . 
```

## Principle

This optimization is implemented in [rpcform_optimized.go](./rpcform_optimized.go), and its principle (for advanced users, those who are not concerned with the detailed principles can skip this) is as follows: In [proposal-A15](https://git.woa.com/trpc/trpc-proposal/blob/master/A15-metrics-rules.md), it is stipulated that for the trpc protocol, the part after the last `'/'` in the rpc name should be extracted as the method name, while for other protocols, the full rpc name is used as the method name. However, this extraction process has been shown to impact performance through stress testing (see point three mentioned in this [discussion](https://git.woa.com/trpc-go/trpc-go/issues/869#note_93064132)). Although there was an attempt with [MR2059](https://git.woa.com/trpc-go/trpc-go/merge_requests/2059) to directly set the method name from the protocol codec. But due to [compatibility issues with aliases](https://git.woa.com/trpc-go/trpc-go/issues/910), this approach was briefly introduced in v0.16.0 and was [reverted](https://git.woa.com/trpc-go/trpc-go/merge_requests/2151) in v0.16.1. Therefore, we have provided a build tag named `optimization` to offer a potential performance optimization option for advanced users (those who value performance).

## Trade-off

By enabling this build tag, performance improvements can be obtained. However, the trade-off is that for the trpc protocol, the method names displayed in monitoring will be the full rpc name (for example, something like `/trpc.app.server.service/Method`, and for aliases, it will be in the form of an HTTP URI like `/v1/xxx/xxxx`). Therefore, the display items will be incompatible with previous versions. Please weigh the pros and cons to decide whether to enable it.
