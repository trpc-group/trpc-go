# tRPC-Go 模块：server



## 背景

一个服务进程可能会监听多个端口，在每个端口上支持不同的业务协议，这种在同时兼容多个业务协议时是非常有帮助的。因此 server 层的设计就必须将进程、服务、逻辑服务的概念进行进一步的抽象设计。

tRPC-Go 里面提炼出了进程、服务（server）、逻辑服务（service）的概念。通常一个进程包含一个 server，每个 server 可以包含一个或者多个逻辑 service。

server 的这种设计能够很好地支持多端口协议的应用场景，在兼容只支持存量业务协议的框架、通过其他端口提供额外的 http 协议给 web 调用等等场景下，都是很方便的。

pb 里面的多 service，其实是为了对众接口进行逻辑分组，配合 tRPC-Go 的这种多 service 能力，后续也可以实现端口上的接口隔离，对接口提供更细粒度地控制。

## 原理

先来看下 server 层的设计及与其他层次的关系：

![server 层及关系](/.resources/developer_guide/module_design/server/server_layer_relations.png)

trpc-go server 服务端，包括网络通信 名字服务 监控统计 链路跟踪等各个组件基础接口，具体实现由第三方 middleware 注册进来。
完整的示例请参考 [helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld) 。

## 接口定义

### Server

trpc server 一个服务进程只有一个 server，一个 server 可以有多个 service

``` go
type Server struct {
    services map[string]Service // k=serviceName,v=Service
}
```

### Service

Service 是具体服务的实现，每个 Service 需要实现各自的服务注册、启动和关闭

``` go
type Service interface {
    // 注册路由信息
    Register(serviceDesc interface{}, serviceImpl interface{}) error
    // 启动服务
    Serve() error
    // 关闭服务
    Close(chan int) error
}
```

### ServiceDesc

ServiceDesc 是一个服务 Service 的具体描述，包括服务名、处理方法等

``` go
// ServiceDesc 服务描述 service 定义
type ServiceDesc struct {
    ServiceName string
    HandlerType interface{}
    Methods     []Method
}
```

### Method

Method 定义了一个 Service 的具体处理方法

``` go
// Method 服务 rpc 方法信息
type Method struct {
    Name string
    Func func(svr interface{}, ctx context.Context, f FilterFunc) (rspbody interface{}, err error)
}
```

## 启动流程

### NewServer 初始化

server 初始化的过程非常简单，通过 NewServer 创建一个 server，然后调用 Serve() 方法提供服务

``` go
s := trpc.NewServer()
err := s.Serve()
if err != nil {
    panic(err)
}
```

server 的 NewServer() 方法支持不传或者传入一个或多个 server.Option 来进行初始化操作。默认情况下，会根据 server 的配置文件 service 的配置去生成 serviceMap，service 是一个具体的服务名称，每个 service 拥有自己的名称和多个处理器 Handler map，server 提供了 service 的注册入口

### 指定参数

与 client 类似，server 支持传入 Option 进行创建，此时的 Option 对所有 Service 都生效

``` go
opts := []server.Option{
    server.WithServiceName("helloworld"),
    server.WithNetwork("tcp"),
    server.WithProtocol("trpc"),
}
s := trpc.NewServer(opts ...)
```

### Serve 开始运行

调用 server 的 Serve() 方法时，会遍历 server 的所有 service，分别调用每个 service 的 Serve 方法

``` go
for _, service := range s.services {
    go service.Serve()
}

```

每个自定义的 service 在必须实现 Register、Serve 和 Close 这三个方法。

trpc 提供默认的 service 实现，会调用 transport 的 ListenAndServe 方法提供服务

``` go
if err = s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
    log.Errorf("service:%s ListenAndServe fail:%v", s.opts.ServiceName, err)
    return err
}
```

## demo

下面列出了 Server 的基本代码，具体可参考 [helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld)

``` go
func main() {
    s := trpc.NewServer()
    pb.RegisterGreeterServer(s, &GreeterServerImpl{})
    if err := s.Serve(); err != nil {
        panic(err)
    }
}
```
