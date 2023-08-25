[TOC]

# tRPC-Go module: codec



## Background

Network communication data needs to follow certain protocols for encoding and decoding. There are a considerable number of protocols that need to be compatible within the company, which requires tRPC-Go to provide a plugin mechanism for easy extension of encoding and decoding.

Considering the additional workload introduced by subsequent protocol integration and debugging, tRPC project team has proposed a new protocol called trpc, which addresses various issues such as streaming, sequencing, custom information, full-link timeout, etc., making it easier to unify protocols in the future.

The codec layer provides this capability.

## Principle

Let's first look at the design of the codec layer:

![trpc-go_codec](/.resources/developer_guide/module_design/codec/uml.png)

The codec layer mainly includes several core interfaces such as serializer, compressor, codec, framer, and framerbuilder. Below is an introduction to these interfaces in conjunction with a schematic diagram.

## Common interface for packing and unpacking business protocols

### Instructions for use

- tRPC-Go can support any third-party business communication protocol, as long as the codec-related interfaces are implemented.
- Each business protocol has its own Go module, which does not affect each other. `go get` command only downloads the required codec modules.
- There are two typical styles of business protocols: IDL protocols such as Tars, and non-IDL protocols such as OIDB. You can refer to the implementation of Tars and OIDB for specific details.
- The framework has implemented default client packing and unpacking methods and server unpacking method, see details at: https://git.woa.com/trpc-go/trpc-go/blob/master/codec/serialization.go
- Business protocols can customize the implementation of Encode and Decode methods.

### Codec

Codec defines the interface for packing and unpacking business protocols. Business protocols are divided into a header and a body, here only the binary body is parsed, and the specific business body structure is processed through a serializer. Generally, the body is in formats such as pb, json, jce, etc. In special cases, the business can register their own serializer.

```go
type Codec interface {
    // pack body into binary buf
    // client: Encode(msg, reqbody)(request-buffer, err)
    // server: Encode(msg, rspbody)(response-buffer, err)
    Encode(message Msg, body []byte) (buffer []byte, err error)
    // extract body from binary buf
    // server: Decode(msg, request-buffer)(reqbody, err)
    // client: Decode(msg, response-buffer)(rspbody, err)
    Decode(message Msg, buffer []byte) (body []byte, err error)
}
```

Trpc-go implements  the packing and unpacking of default client and default server. During framework initialization, the default implementation for the client and server codec with the name "trpc" is registered through the init method.

```go
func init() {
    codec.Register("trpc", DefaultServerCodec, DefaultClientCodec)
    transport.RegisterFramerBuilder("trpc", DefaultFramerBuilder)
}
```

### Msg

`Msg` defines the message data that is commonly used for multiple protocols. In order to support any third-party protocol, trpc abstracts a common data structure called message to carry the basic information required by the framework. The message delivery of an rpc communication process is unified into msg for management. The data structures defined in msg are detailed at: https://git.woa.com/trpc-go/trpc-go/blob/master/codec/message.go

### Compressor

`Compressor` is a common interface for data compression and decompression, providing two common methods: `Compress` and `Uncompress`. The business can customize the implementation of these two methods to achieve different compression and decompression methods. trpc-go currently supports gzip and snappy by default.

```go
// Compressor body decompression interface
type Compressor interface {
    Compress(in []byte) (out []byte, err error)
    Decompress(in []byte) (out []byte, err error)
}
```

### Serializer

`Serializer` is a common interface for data serialization, providing two common methods: `Marshal` and `Unmarshal`. The business can customize the implementation of these two methods to achieve different serialization and deserialization operations. Trpc-go currently supports protobuf, json, and jce by default.

```go
// Serializer body serialization interface
type Serializer interface {
    Unmarshal(in []byte, body interface{}) error
    Marshal(body interface{}) (out []byte, err error)
}
```

## Implement a third-party protocol

Refer to [Implement a third-party protocol](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb)