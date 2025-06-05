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

上边的流程图中涉及到的 `codec` 中的接口详细介绍如下，读者可以结合图进行阅读：

- `Framer`：从来自网络的二进制数据中读取完整业务包的接口。

    ```go
    type Framer interface {
        ReadFrame() ([]byte, error)
    }
    ```

- `Codec`：业务协议打解包接口，业务协议分为包头和包体。这里只需要解析出二进制包体即可，包头一般放在 `msg` 里面，业务不用关心。

    ```go
    type Codec interface {
        // server 解包 => 从完整的二进制网络数据包解析出二进制请求包体
        // client 解包 => 从完整的二进制网络数据包解析出二进制响应包体
        Decode(message Msg, buffer []byte) (body []byte, err error)

        // server 回包 => 把二进制响应包体打包成一个完整的二进制网络数据
        // client 回包 => 把二进制请求包体打包成一个完整的二进制网络数据
        Encode(message Msg, body []byte) (buffer []byte, err error)
    }
    ```

- `Serializer`：包体序列化接口，目前支持 protobuf、json、jce、flatbuffers 和 xml。用户也可以定义自己需要的 `Serializer` 并注册到 `codec` 包。

    ```go
    type Serializer interface {
        // server 解包出二进制包体 => 然后调用该函数解析到具体的请求结构体
        // client 解包出二进制包体 => 然后调用该函数解析到具体的响应结构体
        Unmarshal(in []byte, body interface{}) error

        // server 回包响应结构体 => 调用该函数转成二进制包体
        // client 回包请求结构体 => 调用该函数转成二进制包体
        Marshal(body interface{}) (out []byte, err error)
    }
    ```

- `Compressor`：包体解压缩方式，目前支持 gzip、lz4、snappy 和 zlib。用户也可以定义自己需要的 `Compressor` 并注册到 `codec` 包。

    ```go
    type Compressor interface {
        // server/client 解出二进制包体后调用该函数 => 解压出原始二进制数据
        Decompress(in []byte) (out []byte, err error)

        // server/client 回包二进制包体前调用该函数 => 压缩成小的二进制数据
        Compress(in []byte) (out []byte, err error)
    }
    ```

# 如何实现一个业务协议

## 基本步骤

要实现一个业务协议，至少需要做以下三步：

1. 实现 `Framer` 和 `FramerBuilder` 接口，从连接中读取出完整的业务包。。

2. 实现 `Codec` 业务协议打解包接口。

3. 在 `init` 函数中将具体实现注册到 tRPC 框架中。

除了这三步以外，还有可能需要实现 `Serializer` 和 `Compressor` 接口（通常来说，序列化和压缩都有现成的标准格式可供使用。读者可以阅读和直接使用 `codec` 包中已经实现的若干序列化和压缩方式）。

## 注意事项

在实现过程的第二步中还需要注意以下内容（没有的值可以不设置，关于这些接口的具体使用可以参考 [oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb) 的实现）：

- 在 Server Codec Decode 收请求包后需要调用的接口：
  - 使用 `msg.WithServerRPCName` 告诉 tRPC 如何分发 `/trpc.app.server.service/method` 路由
  - 使用 `msg.WithRequestTimeout` 指定上游服务的剩余超时时间
  - 使用 `msg.WithSerializationType` 指定序列化方式
  - 使用 `msg.WithCompressType` 指定解压缩方式
  - 使用 `msg.WithCallerServiceName` 设置 `trpc.app.server.service` 上游服务名
  - 使用 `msg.WithCalleeServiceName` 设置自身服务名
  - 使用 `msg.WithServerReqHead` 和 `msg.WithServerRspHead` 设置业务协议包头

- 在 Server Codec Encode 回响应包前需要调用的接口：
  - 使用 `msg.ServerRspHead` 取出响应包头回包给客户端
  - 使用 `msg.ServerRspErr` 将 handler 处理函数错误返回 error 转成具体的业务协议包头错误码

- 在 Client Codec Encode 发请求包前需要调用的接口：
  - 使用 `msg.ClientRPCName` 指定请求路由
  - 使用 `msg.RequestTimeout` 告诉下游服务剩余超时时间
  - 使用 `msg.WithCalleeApp` 设置下游服务

- 在 Client Codec Decode 收响应包后需要调用的接口：
  - 使用 `errs.New` 将具体业务协议错误码转换成 error 返回给用户调用函数
  - 使用 `msg.WithSerializationType` 指定序列化方式
  - 使用 `msg.WithCompressType` 指定解压缩方式

# 简单的实现示例

本节以 [trpc-codec](https://git.woa.com/trpc-go/trpc-codec) 中的 rawstring 协议为例来演示实现业务协议的具体步骤，具体的代码请参考[这里](https://git.woa.com/trpc-go/trpc-codec/tree/master/rawstring)。

## 协议介绍

rawstring 协议是一种简单的基于 TCP 的调用协议，其特点是以 `'\n'` 字符为分隔符进行收发包。

## 实现 `Framer` 和 `FramerBuilder` 接口

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
    // 以 '\n' 字符为分隔符进行解包
    return reader.ReadBytes('\n')
}
```

## 实现 `Codec` 接口

```go
// 服务端 Codec
type serverCodec struct{}

func (sc *serverCodec) Decode(_ codec.Msg, req []byte) ([]byte, error) {
    return req, nil
}

func (sc *serverCodec) Encode(_ codec.Msg, rsp []byte) ([]byte, error) {
    // 服务端在响应后边添加一个 '\n' 字符作为完整的二进制网络数据
    return []byte(string(rsp) + "\n"), nil
}

// 客户端 Codec
type clientCodec struct{}

func (cc *clientCodec) Encode(_ codec.Msg, req []byte) ([]byte, error) {
    // 客户端在请求后边添加一个 '\n' 字符作为完整的二进制网络数据
    return []byte(string(reqBody) + "\n"), nil
}

func (cc *clientCodec) Decode(_ codec.Msg, rsp []byte) ([]byte, error) {
    return rspBody, nil
}
```

## 注册实现

```go
// 将实现好的 FramerBuilder 注册到 transport 包
var DefaultFramerBuilder = &FramerBuilder{}
func init() {
    transport.RegisterFramerBuilder("rawstring", DefaultFramerBuilder)
}

// 将实现好的 Codec 注册到 codec 包
func init() {
    codec.Register("rawstring", &serverCodec{}, &clientCodec{})
}
```

# 各种 tRPC-Go 业务协议的实现

在仓库 [trpc-codec](https://git.woa.com/trpc-go/trpc-codec) 中实现了各种具体的业务协议。实现时需要注意的要点如下：

- 只需要实现 codec 中的相关接口就可以让 tRPC-Go 支持任意的第三方业务通信协议。
- 每个业务协议单独一个 go module 互不影响，使用 `go get` 命令时只会拉取需要的 codec 模块。
- 业务协议一般有两种典型样式：IDL 协议（比如 tars）和非 IDL 协议（比如 oidb），具体情况可以分别参考 [tars](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars) 和 [oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb) 的实现。

# 性能优化指引

在 v0.17.0 以后，用户可以在 `go build` 时提供 `optimization` 的 build tag 以进行性能优化，例如：

```shell
go build -tags=optimization . 
```

## 原理

这一优化在 [rpcform_optimized.go](./rpcform_optimized.go) 中实现，其原理（针对高阶用户，不关注原理细节的可以略去）如下：在 [proposal-A15](https://git.woa.com/trpc/trpc-proposal/blob/master/A15-metrics-rules.md) 规定了对于 trpc 协议，需要提取 rpc name 最后一个 `'/'` 之后的部分作为方法名，对于其他协议则是以完整的 rpc name 作为方法名。但是这个提取的过程经过压测显示对性能有影响（见该 [讨论](https://git.woa.com/trpc-go/trpc-go/issues/869#note_93064132) 中提到的第三点），虽然有 [MR2059](https://git.woa.com/trpc-go/trpc-go/merge_requests/2059) 尝试从协议 codec 处直接设置方法名，但是由于[alias 的兼容性问题](https://git.woa.com/trpc-go/trpc-go/issues/910)，这种方案在 v0.16.0 中被短暂引入后，又在 v0.16.1 中被[回滚](https://git.woa.com/trpc-go/trpc-go/merge_requests/2151)。因此我们提供了一个名为 `optimization` 的 build tag 来为高阶用户（看重性能的用户）提供一个可能的性能优化选项。

## 权衡

开启了这个 build tag 之后可以获得性能上的提升，带来的代价则是对于 trpc 协议而言，监控上展示的方法名将为完整的 rpc name（比如类似 `/trpc.app.server.service/Method`，对于 alias 则是形如 HTTP 的 URI 形式 `/v1/xxx/xxxx`），所以显示项会与之前的不相兼容。请业务方自己权衡利弊以考虑是否开启。
