[TOC]

# tRPC-Go module: server



## Background

A service process may listen on multiple ports and support different business protocols on each port, which is helpful when simultaneously accommodating multiple business protocols. Therefore, the design of the server layer must further abstract the concepts of process, server, and logical service.

In tRPC-Go, the concepts of process, server, and service are extracted. Typically, a process contains a server, and each server can contain one or more logical services.

This design of the server can support applications with multiple port protocols very well. It is convenient in scenarios that are compatible with frameworks that only support existing business protocols and provide additional HTTP protocols for web calls through other ports.

The multiple services in pb are actually used to logically group multiple interfaces. With the ability of tRPC-Go to support multiple services, interface isolation can be achieved on ports in the future, providing finer-grained control over the interfaces.


## Theory

Let's look at the design of the server layer and its relationship of other layers.

![server_layer_relations](/.resources/developer_guide/module_design/server/server_layer_relations.png)

tRPC-Go server includes the basic interfaces of various components such as network communication, name service, monitoring statistics, and link tracking. The specific implementation is registered by third-party middleware.

For a complete example, please refer to [helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld).


## Interface Definition

### Server

In tRPC Server, a service process has only one server, and a server can have multiple services.

``` go
type Server struct {
    services map[string]Service // k=serviceName,v=Service
}
```

### Service

Service is the implementation of specific services, and each Service needs to implement its own service registration, startup and shutdown.

``` go
type Service interface {
    // Register routing information
    Register(serviceDesc interface{}, serviceImpl interface{}) error
    // Startup service
    Serve() error
    // shutdown service
    Close(chan int) error
}
```

### ServiceDesc

ServiceDesc is a specific description of a Service, including service name, processing method and so on.

``` go
// definition of ServiceDesc of a service
type ServiceDesc struct {
    ServiceName string
    HandlerType interface{}
    Methods     []Method
}
```

### Method

Method defines a specific processing method of a Service.

``` go
// Method contains infomation of rpc method
type Method struct {
    Name string
    Func func(svr interface{}, ctx context.Context, f FilterFunc) (rspbody interface{}, err error)
}
```

## Startup Process

### NewServer initialization

The process of server initialization is very simple. Create a server through NewServer, and then call the `Serve()` method to provide services.

``` go
s := trpc.NewServer()
err := s.Serve()
if err != nil {
    panic(err)
}
```

`NewServer()` method of server supports not passing or passing in one or more `server.Option` for initialization. 

By default, the serviceMap will be generated according to the configuration of the server configuration file Service. Service is a specific service name. Each Service has its own name and multiple processor Handler map. The server provides the registration entry of the Service.

### Parameter Specification

Similar to client, server supports passing in Option for creation, and Option at this time takes effect for all Services.

``` go
opts := []server.Option{
    server.WithServiceName("helloworld"),
    server.WithNetwork("tcp"),
    server.WithProtocol("trpc"),
}
s := trpc.NewServer(opts ...)
```

### Serve Start Running

While calling the `Serve()` method of the server, it will traverse all the services of the server and call the `Serve()` method of each service separately.

``` go
for _, service := range s.services {
    go service.Serve()
}
```

Each custom service must implement the methods of Register, Serve and Close.

tRPC provides a default service implementation, which will call the transport's `ListenAndServe()` method to provide services.

``` go
if err = s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
    log.Errorf("service:%s ListenAndServe fail:%v", s.opts.ServiceName, err)
    return err
}
```


## Demo

The basic code of Server is listed below. For details, please refer to [helloworld](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld).

``` go
func main() {
    s := trpc.NewServer()
    pb.RegisterGreeterServer(s, &GreeterServerImpl{})
    if err := s.Serve(); err != nil {
        panic(err)
    }
}
```

