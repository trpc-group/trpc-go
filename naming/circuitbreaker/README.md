# Circuit Breaker

Circuit Breaker filters out nodes that have higher error ratio by collecting response of each request.

## Usages
Use `client.WithCircuitBreakerName("xxx")` to specify a circuit breaker.
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

## Circuit Breaker Interface
```go
// CircuitBreaker defines whether a node is available and reports the result of RPC on the node.
type CircuitBreaker interface {
	Available(node *registry.Node) bool
	Report(node *registry.Node, cost time.Duration, err error) error
}
```
The default implementation is NOOP.
