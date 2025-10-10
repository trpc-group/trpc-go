# 概述

模块 `codec` 提供了编解码相关的接口，允许框架扩展业务协议、序列化方式和数据压缩方式。

# 核心概念解析

模块 `codec` 中的主要概念包括 `Msg`、`Framer`、`Codec`、`Serializer` 和 `Compressor` 等接口，我们将依次介绍它们。

- `Msg`：每个请求的通用消息体。为了支持任意的第三方协议，在 tRPC 中抽象出了这个接口来携带框架需要的基本信息。结构体 `msg` 是该接口的唯一实现。

在介绍剩下的接口之前，我们先用两张图展示出服务端和客户端的协议处理流程，以便读者能够获得一个整体上的认知。

服务端处理流程

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

客户端处理流程

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

- `codec.Framer` 读取来自网络的的二进制数据。

```go
// Framer defines how to read a data frame.
type Framer interface {
    ReadFrame() ([]byte, error)
}
```

- `code.Codec`：提供 `Decode` 和 `Encode` 接口， 分别从完整的二进制网络数据包解析出二进制请求包体，和把二进制响应包体打包成一个完整的二进制网络数据。

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

- `codec.Compressor`：提供 `Decompress` 和 `Compress` 接口，目前支持 gzip 和 snappy 类型的 `Compressor`，你可以定义自己需要的 `Compressor` 注册到 `codec` 包

```go
// Compressor is body compress and decompress interface.
type Compressor interface {
	Compress(in []byte) (out []byte, err error)
	Decompress(in []byte) (out []byte, err error)
}
```

- `codec.Serializer`：提供 `Unmarshal` 和 `Marshal` 接口，目前支持 protobuf、json、fb 和 xml 类型的 `Serializer`，你可以定义自己需要的 `Serializer` 注册到 `codec` 包。

```go
// Serializer defines body serialization interface.
type Serializer interface {
    // Unmarshal deserialize the in bytes into body
    Unmarshal(in []byte, body interface{}) error

    // Marshal returns the bytes serialized from body.
    Marshal(body interface{}) (out []byte, err error)
}
```