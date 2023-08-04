# Selector Interface

Selector selects a node by service name, it internally calls service discovery, load balance and circuit breaker.

```
// Selector is the interface to select a node from service name.
type Selector interface {
	// Select selects a node from service name.
	Select(serviceName string, opt ...Option) (*registry.Node, error)
	// Report reports request status.
	Report(node *registry.Node, cost time.Duration, success error) error
}
```
