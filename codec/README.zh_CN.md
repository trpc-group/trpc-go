[English](README.md) | 中文
# codec

`codec` 包可以支持任意的第三方业务通信协议，只需要实现相关接口即可。
下面以服务端的协议处理流程为例介绍 `codec` 的相关接口， 客户端的协议处理流程与服务端的协议处理流程相反，这里不再赘述。
关于怎么开发第三方业务通信协议的插件， 可参考[这里](/docs/developer_guide/develop_plugins/protocol.zh_CN.md)。

## 相关接口

下图展示了服务端的协议处理流程，其中包含了`codec`包中的相关接口。

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