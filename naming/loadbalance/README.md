# tRPC-Go Load Balance

LoadBalancer is used for each request, not connection. It's decoupled from client, and the different load balance
strategy maintains their own status. tRPC-Go provides round-robin and smooth weighted round-robin.

## Usages
Use `client.WithBalancerName("xxx")` to specify a load balance algorithm.
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

## Load Balancer Interface
```go
// LoadBalancer is the interface which returns a node from node list.
type LoadBalancer interface {
	Select(serviceName string, list []*registry.Node, opt ...Option) (node *registry.Node, err error)
}
```
The custom implementation should refer to the implementation inside that project.
