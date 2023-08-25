# tRPC-Go 模块：codec



## 背景

网络通信数据需要遵循一定的协议进行编解码，公司内存在数量不少的协议需要兼容，这就要求 tRPC-Go 需要提供一种插件机制方便扩展编解码。
同时考虑到后续协议联调引入的额外工足量，非常不便，tRPC 项目工作组也提出了一种新的协议 trpc，它考虑了流式、序号、自定义信息、全链路超时等的各种问题，方便后续统一协议。

codec 层提供了这方面的能力。

## 原理

先来看下 codec 层的设计：

![](/.resources/developer_guide/module_design/codec/uml.png)

codec 层大致包括 serializer、compressor、codec、framer、framerbuilder 这几个核心接口，下面结合此类图进行介绍。

## 业务协议打解包通用接口

### 一、使用说明

- tRPC-Go 可以支持任意的第三方业务通信协议，只需要实现 codec 相关接口即可。
- 每个业务协议单独一个 go module，互不影响，go get 只会拉取需要的 codec 模块。
- 业务协议一般有两种典型样式：IDL 协议如 tars，非 IDL 协议如 oidb，具体情况可以分别参考 tars 和 oidb 的实现。
- 框架实现了默认的 client 打解包和 server 解包方法，详见：https://git.woa.com/trpc-go/trpc-go/blob/master/codec/serialization.go
- 业务协议可以自定义实现 Encode 和 Decode 方法

### 二、Codec

codec 定义了业务协议打解包接口，业务协议分成包头 head 和包体 body，这里只解析出二进制 body，具体业务 body 结构体通过 serializer 来处理，一般 body 都是 pb json jce 等，特殊情况可由业务自己注册 serializer。

```go
type Codec interface {
    // 打包 body 到二进制 buf 里面
    // client: Encode(msg, reqbody)(request-buffer, err)
    // server: Encode(msg, rspbody)(response-buffer, err)
    Encode(message Msg, body []byte) (buffer []byte, err error)
    // 从二进制 buf 里面解出 body
    // server: Decode(msg, request-buffer)(reqbody, err)
    // client: Decode(msg, response-buffer)(rspbody, err)
    Decode(message Msg, buffer []byte) (body []byte, err error)
}
```

trpc-go 实现了默认的 client 和 server 打解包，在框架初始化时通过 init 方法注册名字为“trpc”的 client 和 server codec 的默认实现。

```go
func init() {
    codec.Register("trpc", DefaultServerCodec, DefaultClientCodec)
    transport.RegisterFramerBuilder("trpc", DefaultFramerBuilder)
}
```

### 三、Msg

msg 定义了多协议通用的消息数据，为了支持任意的第三方协议，trpc 抽象出了 message 这个通用数据结构来携带框架需要的基本信息。一次 rpc 通信过程的消息传递统一放到 msg 中进行管理，msg 中定义的数据结构详见：https://git.woa.com/trpc-go/trpc-go/blob/master/codec/message.go

### 四、Compressor

Compressor 是数据压缩/解压通用接口，提供 Compress 和 Uncompress 两个通用方法，业务可以通过自定义实现这两个方法，来实现不同压缩和解压方式。trpc-go 目前默认支持 gzip snappy

```go
// Compressor body 解压缩接口
type Compressor interface {
    Compress(in []byte) (out []byte, err error)
    Decompress(in []byte) (out []byte, err error)
}
```

### 五、Serializer

Serializer 是数据序列化通用接口，提供 Marshal 和 Unmarshal 两个通用方法，业务可以通过自定义实现这两个方法，来实现不同序列化和反序列化操作。trpc-go 目前默认支持 protobuf json jce

```go
// Serializer body 序列化接口
type Serializer interface {
    Unmarshal(in []byte, body interface{}) error
    Marshal(body interface{}) (out []byte, err error)
}
```

### 六、实现一个第三方协议

参考 [实现一个第三方协议](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb)