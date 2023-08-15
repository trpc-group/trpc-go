# tRPC-Go Codec [中文主页](README_CN.md)
tRPC-Go codec package defines the business communication protocol of packing and unpacking.
- tRPC-Go can support any third party business communication protocol, which implements codec interface.
- Every business protocol uses one single go module, and the `go get` command only pulls the specific codec module.
- There are two typical business protocol pattern: IDL protocol, such as tars, and non-IDL protocol, such as oidb. those two examples can be refer when implements other protocol.

## Business Protocol Implementation Repository: https://trpc.group/trpc-go/trpc-codec


# Core Concept
- Message: common message body for every request. To support any third party protocol, tRPC provides the `message` structure to carry framework basic information.
- Codec: business protocol packing and unpacking interface. Business protocol contains head and body, the codec implementation only needs to parse binary body, and don't need to care about head, which is in msg.

```golang
type Codec interface {
	// Encode pack the body into binary buffer.
	Encode(message Msg, body []byte) (buffer []byte, err error)

	// Decode unpack the body from binary buffer.
	Decode(message Msg, buffer []byte) (body []byte, err error)
}
```

- Serializer: body serialization interface, now tRPC-Go supports protobuf, json, fb and xml protocol. This serializer is pluggable and users can define their own and register it. 
```golang
type Serializer interface {
	// Unmarshal deserialize the in bytes into body
	Unmarshal(in []byte, body interface{}) error

	// Marshal returns the bytes serialized from body.
	Marshal(body interface{}) (out []byte, err error)
}
```

- Compressor: body compress and decompress interface, now tRPC-Go supports gzip and snappy protocol. This compressor is also pluggable and users can define their own and register it
  
```golang
type Compressor interface {
    // Decompress returns the origin binary data decompressed by the compressor.
	Decompress(in []byte) (out []byte, err error)
	// Compress returns the compressed binary body data.
	Compress(in []byte) (out []byte, err error)
}
```

# Specific Implementation Steps (refer to [trpc-codec](codec.go))
- 1. Implements tRPC-Go [FrameBuilder unpacking interface](transport/transport.go), parses out a complete message package。
- 2. Implements tRPC-Go [Codec packing and unpacking interface](codec/codec.go), and the following points should be noted:
    - after server codec decode receives request package, it needs to tell tRPC how to dispatch route by `msg.WithServerRPCName` and set upstream left timeout by `msg.WithRequestTimeout`。
    - before server codec encode sends response package, it needs to transfer the error returned by handler function into business protocol package head error code by `msg.ServerRspErr`.
    - before client codec encode sends request package, it needs to set request route by `msg.ClientRPCName` and tell the left timeout to downstream by `msg.RequestTimeout`.
    - after client codec decode receives response package, it needs to transfer business protocol error code into error by `errs.New`, which will returned to user call function.
- 3. registers the codec implementation into tRPC framework by `init()` function.