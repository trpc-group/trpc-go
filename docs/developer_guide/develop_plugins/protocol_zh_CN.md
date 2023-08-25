[TOC]

# 前言

根据 tRPC 框架的设计原则，框架需要插件化支持其他业务常用的协议，为了满足该需求，框架设计出支持协议注册的 codec 模块。

该模块主要为了让用户只用关注 codec 的实现就可以将自己的业务协议应用到框架中，下面主要介绍了插件协议的设计原理。

无论何种协议，最终都是做为请求和回复的一种表现形式，主要是让用户能够更安全，更高效的传输自己所需要的信息，
对于C/S架构的tRPC框架来说, 其处理请求和回复的过程中，插件的调用流程如下图所示：

![ 'tRPC 插件流程图'](/.resources/developer_guide/develop_plugins/protocol/tRPC%20Plugin%20Flowchart.png)

从上图中可以看出，tRPC-Go 框架会调用 Codec(Client)
对客户端用户的请求进行编码，当请求到达服务端的时候，框架会调用服务端的 Codec(Server) 来对用户的请求进行解码，传入服务端业务处理代码，最后服务端给出相应的回复数据。

tRPC 使用该模型基本统一了各种 RPC 协议，存储组件客户端，采用统一的调用模型 + 拦截器可以很好的实现监控上报，分布式 trace 及日志功能，
对于业务做到无感。

目前 tRPC-Go 封装实现的协议有 sso, wns, oidb proto, ilive, nrpc 等协议，也封装了 mysql, redis, ckv 等客户端。
可见trpc-go/trpc-codec 及 trpc-go/trpc-database。

# 原理

## 协议设计需要实现的接口

```go
// FramerBuilder 通常每个连接 Build 一个 Framer, 用于不断的从一个连接中读取完整业务包。
type FramerBuilder interface {
New(io.Reader) Framer
}
```

```go
// Framer 读写数据桢。用于从 tcp 流中读取一个完整的业务包，并 copy 出来，交给后续的 Docode 处理。
type Framer interface {
ReadFrame() ([]byte, error)
}
```

```go
// Codec 业务协议打解包接口，业务协议分成包头 head 和包体 body
// 这里只解析出二进制 body，具体业务 body 结构体通过 serializer 来处理，
// 一般 body 都是 pb json jce 等，特殊情况可由业务自己注册 serializer
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

## 服务端协议插件原理

![ '服务端协议插件流程图'](/.resources/developer_guide/develop_plugins/protocol/Server-side%20Protocol%20Plugin%20Flowchart.png)

tRPC-Go 中的协议处理一般流程如下：

1. 自定义的协议 import 后，init 函数中注册 codec 和 FramerBuilder.

2. trpc.NewServer 根据 service 的 protocol 配置，从插件管理器中取出对应的 FramerBuilder 和 Codec, 进行设置。

3. 注册 rpc name 和对应的业务函数。

4. 启动监听

5. 服务端的 service 收到一个连接

6. 根据 FramerBuilder 构建一个 Framer

7. Framer ReadFrame, 读取一个完整的业务帧

8. 根据配置的 Codec Decode 出来 head 和业务 body(此时还是[]byte), 在此过程中一般会设置 SerializationType, 用于获取 serializer,
   同时也会设置 ServerRPCName, 用于从业务注册方法中获取处理方法。

9. 获取一条 filter.Chain, 在其中使用 serializer Unmarshal 将业务 body 反序列化成对应的结构体 (比如 pb, jce 等), 交给业务逻辑代码处理。

10. 业务逻辑返回 rsp struct.

11. 调用 serializer Marshal 将 rsp struct 序列化成[]byte, 写回客户端。

## 客户端协议插件原理

客户端的处理流程基本与服务端的类似。基本是相反的过程。

![ '客户端协议插件流程图'](/.resources/developer_guide/develop_plugins/protocol/Client-side%20Protocol%20Plugin%20Flowchart.png)

# 实现

## 设置 msg 字段

需要注意以下几点 (一些不需要的值可以不设置)：

server codec decode 收请求包后，需要调用的接口（没有的值可不设置）：

- msg.WithServerRPCName 告诉 trpc 如何分发路由 /trpc.app.server.service/method
    - msg.WithRequestTimeout 指定上游服务的剩余超时时间
    - msg.WithSerializationType 指定序列化方式
    - msg.WithCompressType 指定解压缩方式
    - msg.WithCallerServiceName 设置上游服务名 trpc.app.server.service
    - msg.WithCalleeServiceName 设置自身服务名
    - msg.WithServerReqHead msg.WithServerRspHead 设置业务协议包头


- server codec encode 回响应包前，需要调用的接口：

    - msg.ServerRspHead 取出响应包头，回包给客户端
    - msg.ServerRspErr 将 handler 处理函数错误返回 error 转成具体的业务协议包头错误码


- client codec encode 发请求包前，需要调用的接口：

    - msg.ClientRPCName 指定请求路由
    - msg.RequestTimeout 告诉下游服务剩余超时时间
    - msg.WithCalleeServiceName 设置下游服务 app server service method

- client codec decode 收响应包后，需要调用的接口：
    - errs.New 将具体业务协议错误码转换成 err 返回给用户调用函数
    - msg.WithSerializationType 指定序列化方式
    - msg.WithCompressType 指定解压缩方式

## 数值型命令字老协议如何支持 rpc 服务描述方式

一些老协议如 oidb 是通过数字命令字（command/servicetype）来分发不同方法的，不像 rpc 是用字符串来分发。

tRPC 都是 rpc 服务，对于数字命令字类型的非 rpc 协议可以通过注释别名的方式来转化成 rpc 服务，然后自己定义 service 即可，如下所示：


```go
syntax = "proto2";
package tencent.im.oidb.cmd0x110;
option go_package="git.woa.com/trpc-go/trpc-codec/oidb/examples/helloworld/cmd0x110";
message ReqBody {
optional bytes req = 1;
}
message RspBody {
optional bytes rsp = 1;
}
service Greeter {
rpc SayHello(ReqBody) returns (RspBody); // @alias=/0x110/1
}
```
- tRPC 服务默认 rpc 名字是 /packagename.Service/Method，如 /tencent.im.oidb.cmd0x110.Greeter/SayHello 这个对于数值型命令字的老协议来说无法兼容。
- 针对这种情况 trpc 工具提供了一个全新的实现方式，只需在 method 后面加上 // @alias=/0x110/1 , trpc 工具就会自动将 rpcname 替换成注释的内容。这样框架会根据 server 的 decode 方法中设置的 RPCName 来找到该方法进行处理。
- 对于 body 是 protobuf 或者 json 的所有任意协议都可以转化成 rpc 格式服务。
- 执行命令 trpc create -protofile=xxx.proto -alias 创建服务即可。

实现参考

可参考 oidb 的协议

# 示例

## oidb

https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb

## tars

https://git.woa.com/trpc-go/trpc-codec/tree/master/tars

# 总结

实现一个业务协议，需要实现一个 Framer 用于从 tcp 中解出完整业务包，实现 server codec 接口和 client codec 接口，serializer(可能需要).
同时需要注意，在 encode 和 decode 方法中，设置一些元信息，用于寻找处理方法或者 marshal, unmarshal.

# OWNER

## yifhao
