# tRPC-Go 负载均衡

针对每个请求进行负载均衡，而不是针对每个连接进行负载均衡，负载均衡与服务发现以及客户端完全解耦，负载均衡在内部根据不同的负载均衡策略维护自身的状态。trpc-go 提供轮训、平滑加权轮训等负载均衡算法。

## 使用
通过 client.WithBalancerName("xxx") 指定使用的负载均衡算法。
```go
opts := []client.Option{
	client.WithBalancerName("round_robin"),
}

proxy := pb.NewGreeterProxy()
req := &pb.HelloRequest{
	Msg: "trpc-go-client",
}
proxy.SayHello(ctx, req, opts...)
```

## 负载均衡接口
```go
// LoadBalancer 负载均衡接口，通过 node 数组返回一个 node
type LoadBalancer interface {
	Select(serviceName string, list []*registry.Node, opt ...Option) (node *registry.Node, err error)
}
```
自定义实现参考项目内部的实现。

