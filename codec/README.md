English | [中文](README.zh_CN.md)

The `codec` package can support any third-party business communication protocol by simply implementing the relevant interfaces.
The following introduces the related interfaces of the `codec` package with the server-side protocol processing flow as an example.
The client-side protocol processing flow is the reverse of the server-side protocol processing flow, and is not described here.
For information on how to develop third-party business communication protocol plugins, please refer to [here](/docs/developer_guide/develop_plugins/protocol.md).

## Related Interfaces

The following diagram shows the server-side protocol processing flow, which includes the related interfaces in the `codec` package.

```ascii
                              package                     req body                                                       req struct
+-------+        +-------+    []byte     +--------------+  []byte    +-----------------------+    +----------------------+
|       +------->+ Framer +------------->| Codec-Decode +----------->| Compressor-Decompress +--->| Serializer-Unmarshal +------------+
|       |        +-------+               +--------------+            +-----------------------+    +----------------------+            |
|       |                                                                                                                        +----v----+
|network|                                                                                                                        | Handler |
|       |                                                 rsp body                                                               +----+----+
|       |                                                  []byte                                                         rsp struct  |
|       |                                +---------------+           +---------------------+       +--------------------+             |
|       <--------------------------------+  Codec-Encode +<--------- + Compressor-Compress + <-----+ Serializer-Marshal +-------------+
+-------+                                +---------------+           +---------------------+       +--------------------+
```

- `codec.Framer` reads binary data from the network.

```go
// Framer defines how to read a data frame.
type Framer interface {
    ReadFrame() ([]byte, error)
}
```

- `code.Codec`: Provides the `Decode` and `Encode` interfaces, which parse the binary request body from the complete binary network data package and package the binary response body into a complete binary network data package, respectively.

```go
// Codec defines the interface of business communication protocol,
// which contains head and body. It only parses the body in binary,
// and then the business body struct will be handled by serializer.
// In common, the body's protocol is pb, json, etc. Specially,
// we can register our own serializer to handle other body type.
type Codec interface {
    // Encode pack the body into binary buffer.
    // client: Encode(msg, reqBody)(request-buffer, err)
    // server: Encode(msg, rspBody)(response-buffer, err)
    Encode(message Msg, body []byte) (buffer []byte, err error)

    // Decode unpack the body from binary buffer
    // server: Decode(msg, request-buffer)(reqBody, err)
    // client: Decode(msg, response-buffer)(rspBody, err)
    Decode(message Msg, buffer []byte) (body []byte, err error)
}
```

- `codec.Compressor`: Provides the `Decompress` and `Compress` interfaces. 
Currently, gzip and snappy type `Compressor` are supported. 
You can define your own `Compressor` and register it to the `codec` package.

```go
// Compressor is body compress and decompress interface.
type Compressor interface {
	Compress(in []byte) (out []byte, err error)
	Decompress(in []byte) (out []byte, err error)
}
```

- `codec.Serializer`: Provides the `Unmarshal` and `Marshal` interfaces. 
Currently, protobuf, json, fb, and xml types of `Serializer` are supported. 
You can define your own `Serializer` and register it to the `codec` package.

```go
// Serializer defines body serialization interface.
type Serializer interface {
    // Unmarshal deserialize the in bytes into body
    Unmarshal(in []byte, body interface{}) error

    // Marshal returns the bytes serialized from body.
    Marshal(body interface{}) (out []byte, err error)
}
```
