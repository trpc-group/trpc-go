# Service Discovery

Service discovery gets service node list by interacting with service registry center.

## Usages
Use `client.WithDiscoveryName("xxx")` to specify the service discovery.
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

## Service Discovery Interface
```go
// Discovery returns node list by service name.
type Discovery interface {
	List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}
```
Refer framework default implementation to how to implement service discovery.
