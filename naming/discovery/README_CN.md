# tRPC-Go 服务发现

服务发现模块通过与服务注册中心交互，获取服务的节点信息。

## 使用
通过 client.WithDiscoveryName("xxx") 指定使用的服务发现。
```go
opts := []client.Option{
	client.WithDiscoveryName("xxxx"),
}

proxy := pb.NewGreeterProxy()
req := &pb.HelloRequest{
	Msg: "trpc-go-client",
}
proxy.SayHello(ctx, req, opts...)
```

## 服务发现接口
```go
// Discovery 服务发现接口，通过 service name 返回 node 数组
type Discovery interface {
	List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}
```
服务发现实现参考框架的默认实现。
