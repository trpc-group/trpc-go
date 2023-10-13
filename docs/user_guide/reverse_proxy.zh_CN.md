[English](reverse_proxy.md) | 中文

# tRPC-Go 反向代理

## 前言

在某些特殊场景中，如反向代理转发服务，需要完全透传二进制 body 数据，而不进行序列化，和反序列化请求，和响应以提升转发性能，tRPC-Go 通过提供自定义序列化方式对这些场景也提供了支持。

## 实现

### 服务端透传

服务端透传指的是，server 收到请求时直接把二进制 body 取出来交给 handle 处理函数，没有经过反序列化，回包时，也是直接把二进制 body 打包给上游，没有经过序列化。

#### 自定义桩代码文件
因为没有序列化与反序列化过程，也就是没有 pb 协议文件，所以需要用户自己编写服务桩代码和处理函数。
关键点是使用`codec.Body`（或者实现 BytesBodyIn BytesBodyOut 接口，详情看 [这里](https://github.com/trpc-group/trpc-go/blob/ed918a35b8318d59afc4363d9a2a09bfcac75ab9/codec/serialization_noop.go#L26)）来透传二进制，使用`通配符*`进行转发，并自己`执行 filter 拦截器`。

```go
// AccessServer 处理函数接口
type AccessServer interface {
    Forward(ctx context.Context, reqbody *codec.Body) (rspbody *codec.Body, err error)
}

// AccessServer_Forward_Handler 框架的消息处理回调函数
func AccessServer_Forward_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (rspbody interface{}, err error) {
    req := &codec.Body{}
    filters, err := f(req)
    if err != nil {
        return nil, err
    }
    handleFunc := func(ctx context.Context, reqbody interface{}) (rspbody interface{}, err error) {
        return svr.(AccessServer).Forward(ctx, reqbody.(*codec.Body))
    }
    var rsp interface{}
    rsp, err = filters.Filter(ctx, req, handleFunc)
    if err != nil {
        return nil, err
    }
    return rsp, nil
}

// AccessServer_ServiceDesc 自定义服务描述信息，注意使用通配符*进行转发
var AccessServer_ServiceDesc = server.ServiceDesc{ 
    ServiceName: "trpc.app.server.Access", 
    HandlerType: ((*AccessServer)(nil)), 
    Methods: []server.Method{ 
        server.Method{ 
            Name: "*", 
            Func: AccessServer_Forward_Handler, 
        }, 
    }, 
} 

// RegisterAccessService 注册服务
func RegisterAccessService(s server.Service, svr AccessServer) { 
    s.Register(&AccessServer_ServiceDesc, svr) 
} 
```

#### 指定空序列化方式

定义完桩代码以后，就可以实现处理函数并启动服务，关键点是 NewServer 时传入`WithCurrentSerializationType(codec.SerializationTypeNoop)`告诉框架当前消息只透传不序列化。

```go
type AccessServerImpl struct{}

// Forward 转发代理逻辑
func (s *AccessServerImpl) Forward(ctx context.Context, reqbody *codec.Body) (rspbody *codec.Body, err error) {
    // 你自己的内部处理逻辑
}

func main() {
    s := trpc.NewServer(
        server.WithCurrentSerializationType(codec.SerializationTypeNoop),
    )  // 不序列化
    
    RegisterAccessService(s, &AccessServerImpl{})

    if err := s.Serve(); err != nil { 
        panic(err) 
    } 
}
```

### 客户端透传

客户端透传指的是，向下游发起 rpc 请求时直接把二进制 body 打包发出去，没有经过序列化，回包后，也是直接把二进制 body 返回，没有经过反序列化。

#### 指定空序列化方式

需要注意的是，虽然当前框架没有经过序列化，但是仍然需要告诉下游当前二进制已经通过什么序列化方式打包好了，因为下游需要通过这个序列化方式来解析，所以关键是要设置`WithSerializationType` `WithCurrentSerializationType`这两个 option。

```go
ctx, msg := codec.WithCloneMessage(ctx) // 复制一个 ctx，生成 caller callee 等信息，方便框架监控上报
msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")  // 设置下游方法名
msg.WithCalleeServiceName("trpc.test.helloworld.Greeter")  // 设置下游服务名
callopts := []client.Option{
    client.WithProtocol("trpc"),
    client.WithSerializationType(codec.SerializationTypePB),          // 告诉下游当前 body 已经以 pb 序列化过了
    client.WithCurrentSerializationType(codec.SerializationTypeNoop), // 告诉框架当前 client 只透传不序列化
}

req := &codec.Body{Data: []byte("我是一个已经通过其他序列化方式打包好的二进制数据")}
rsp := &codec.Body{}  // 回包后，框架会自动把二进制数据填充到这个 rsp.Data 里面
err := client.DefaultClient.Invoke(ctx, req, rsp, callopts...) // req rsp 是用户自己已经序列化好的二进制数据
if err != nil {
    return err
}
```

## FAQ

### Q1：SerializationType 和 CurrentSerializationType 这两个 option 是什么意思，有什么区别

框架通过提供 `SerializationType` 和 `CurrentSerializationType` 这两种概念来支持代理转发这种场景。
SerializationType 主要用于网络调用的上下文传递，CurrentSerializationType 主要用于当前框架数据解析。
`SerializationType`指的是 body 的原始序列化方式，正常情况都会在协议字段里面指定，tRPC 默认序列化类型是 pb。
`CurrentSerializationType`指的是框架接收到数据时，真正用来执行序列化操作的方式，一般不用填，默认等于 SerializationType，当用户设置 CurrentSerializationType 时，则以用户设置为准，这样就可以允许用户自己设置任意的序列化方式，代理透传时指定 `NoopSerializationType` 即可。
