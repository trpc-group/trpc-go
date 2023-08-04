# Service Registry

Service Registry registers service nodes and reports healthy by interacting with service registry center.

## Service Registry Interface
```go
// Registry is the service registry interface.
type Registry interface {
	Register(service string, opt ...Option) error
	Deregister(service string) error
}
```
The custom implementation should refer to the implementation inside that project.

