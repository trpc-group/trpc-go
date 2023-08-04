# tRPC-Go 熔断器

针对每个请求的请求结果都会进行上报处理，熔断器会根据上报的情况，如果触发熔断，则会对服务器节点进行熔断处理。

## 使用
通过 client.WithCircuitBreakerName("xxx") 指定使用的熔断器。
```go
opts := []client.Option{
	client.WithCircuitBreakerName("xxxx"),
}

proxy := pb.NewGreeterProxy()
req := &pb.HelloRequest{
	Msg: "trpc-go-client",
}
proxy.SayHello(ctx, req, opts...)
```

## 熔断器接口
```go
// CircuitBreaker 熔断器接口，判断 node 是否可用，上报当前 node 成功或者失败
type CircuitBreaker interface {
	Available(node *registry.Node) bool
	Report(node *registry.Node, cost time.Duration, err error) error
}
```
默认实现为不熔断处理。

