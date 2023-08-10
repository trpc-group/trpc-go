# tRPC-Go 业务协议打解包实现
- tRPC-Go 可以支持任意的第三方业务通信协议，只需要实现 codec 相关接口即可。
- 每个业务协议单独一个 go module，互不影响，go get 只会拉取需要的 codec 模块。
- 业务协议一般有两种典型样式：IDL 协议 如 tars，非 IDL 协议 如 oidb，具体情况可以分别参考 tars 和 oidb 的实现。

## 具体业务协议实现仓库：https://trpc.group/trpc-go/trpc-codec

# 相关概念解析
- Message：每个请求的通用消息体，为了支持任意的第三方协议，trpc 抽象出了 message 这个通用数据结构来携带框架需要的基本信息。
- Codec：业务协议打解包接口，业务协议分为 包头 和 包体，这里只需要解析出二进制包体即可，一般包头放在 msg 里面，业务不用关心。
```golang
type Codec interface {
    //server解包 从完整的二进制网络数据包解析出二进制请求包体
    Decode(message Msg, request-buffer []byte) (reqbody []byte, err error)
    //server回包 把二进制响应包体打包成一个完整的二进制网络数据
    Encode(message Msg, rspbody []byte) (response-buffer []byte, err error)
}
```
- Serializer：body 序列化接口，目前支持 protobuf json fb xml。可插拔，用户可自己定义并注册进来。
```golang
type Serializer interface {
    //server解包出二进制包体后，调用该函数解析到具体的reqbody结构体
    Unmarshal(req-body-bytes []byte, reqbody interface{}) error
    //server回包rspbody结构体，调用该函数转成二进制包体
    Marshal(rspbody interface{}) (rsp-body-bytes []byte, err error)
}
```
- Compressor：body 解压缩方式，目前支持 gzip snappy。可插拔，用户可自己定义并注册进来。
```golang
type Compressor interface {
    //server解出二进制包体，调用该函数，解压出原始二进制数据
	Decompress(in []byte) (out []byte, err error)
	//server回包二进制包体，调用该函数，压缩成小的二进制数据
	Compress(in []byte) (out []byte, err error)
}
```

# 具体实现步骤（可参考[trpc-go/codec.go](codec.go)）
- 1. 实现 tRPC-Go [FrameBuilder 拆包接口](transport/transport.go), 拆出一个完整的消息包。
- 2. 实现 tRPC-Go [Codec 打解包接口](codec/codec.go)，需要注意以下几点：
 - server codec decode 收请求包后，需要通过 msg.WithServerRPCName msg.WithRequestTimeout 告诉 trpc 如何分发路由以及指定上游服务的剩余超时时间。
 - server codec encode 回响应包前，需要通过 msg.ServerRspErr 将 handler 处理函数错误返回 error 转成具体的业务协议包头错误码。
 - client codec encode 发请求包前，需要通过 msg.ClientRPCName msg.RequestTimeout 指定请求路由及告诉下游服务剩余超时时间。
 - client codec decode 收响应包后，需要通过 errs.New 将具体业务协议错误码转换成 err 返回给用户调用函数。
- 3. init 函数将具体实现注册到 trpc 框架中。