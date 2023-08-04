# tRPC-Go Service Router

Service Router filters nodes between service discovery and load balance to choose a specific node.

## Service Router Interface
```go
type ServiceRouter interface {
	Filter(serviceName string, nodes []*registry.Node, opt ...Option) ([]*registry.Node, error)
}
```
The custom implementation should refer to the implementation inside that project.
