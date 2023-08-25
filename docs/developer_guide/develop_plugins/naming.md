[TOC]

# Introduction
tRPC-Go supports naming plugins, users can use different suitable naming plugins, or develop a new naming plugin for specific needs.

# Pluggable Design
Naming plugins use a pluggable design, tRPC-Go provides standard naming interface, users can register their own naming plugin.

This article will introduce how to implement a custom naming plugin.

A naming service includes service discovery, service router, load balance, and circuit breaker. For example:
- 1. Discovery: use service name to obtain corresponding node list.
- 2. ServiceRouter: use service router to filter unsatisfied nodes.
- 3. LoadBalance: choose node with load balancing algorithm.
- 4. CircuitBreaker: check whether node satisfy circuit breaker rules and report the result.

# Implementation

tRPC-Go offers two types of naming interface.

- All-in-one: Naming service registers with the framework as a single plugin. This is simpler, and a plugin can control the whole name resolve process.
- Modular: service discovery, service router, load balance, and circuit breaker register with the framework separately. This is more flexible, and users can combine different modules, however it is more complex to implement.

## All-in-one Implementation
In an all-in-one implementation, the interface passes in the naming id, and returns the chosen node. Users can specify the naming plugin with `client.WithTarget`.

The interface definition is defined as follow:
```go
// Selector route selector interface
type Selector interface {
	// Select a node in serviceName
	Select(serviceName string, opt ...Option) (*registry.Node, error)
	// Report current request's cost and error (if applicable)
	Report(node *registry.Node, cost time.Duration, success error) error
}
```

How to implement a custom naming plugin? Here's an example:

```go

// Store the correspoding service and nodes information
var store = map[string][]*registry.Node{} {
  "service1": []*registry.Node{
    	&registry.Node{
            Address: "127.0.0.1:8080",
        }, 
        &registry.Node{
            Address: "127.0.0.1:8081",
        },
    },
}

// Register the implementation
func init() {
	selector.Register("example", &exampleSelector{})
} 

type exampleSelector struct{} 
// Select a node in serviceName
func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
   	list, ok := store[serviceName]
    if !ok || len(list) == 0 {
    	return nil, errors.New("no available node")
    }
    
    return list[rand.Intn(len(list))]
}

// Report current request's cost and error (if applicable)
func (s *exampleSelector) Report(node *registry.Node, cost time.Duration, success error) error {
    return nil
}
```

With the example implementation, users can use `client.WithTarget("example://service1")` to obtain a service node.

### Usage Example
With our naming plugin `exampleSelector` and import path `github.com/naming-plugin/example-selector`, we can write a demo:

```go
package main

import (
	_ "github.com/naming-plugin/example-selector"
)

func main() {
	proxy := pb.NewGreeterClientProxy()

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := proxy.SayHello(
		ctx, 
		req,
		client.WithTarget("example://my-service-id"),
	)
	
	fmt.Println(rsp, err)
}
```

## Modular Implementation
The modular interface provides more flexibility, and allows the user to configure each module separately (e.g. load balancing algorithm, service router rules). Users can use `client.WithServiceName("trpc.app.server.service")` for modular naming plugins.

If user doesn't specify the implementation, a default implementation will be used:

- Discovery: by default, take service name as ip:port string.
- Service Router: by default, does nothing.
- Load Balancing: by default, randomly choose a node.
- Circuit Breaker: by default, does nothing.

Here's an example of a custom `Discovery` implementation:

```go
// Discovery Return node list in service name
type Discovery interface {
    List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}
```

```go
func init() {
	discovery.Register("my_discovery", &MyDiscovery{})
}

// MyDiscovery demo implementation
type MyDiscovery struct{}

// List Return IP:Port list
func (*MyDiscovery) List(serviceName string, opt ...Option) ([]*registry.Node, error) {
    node := registry.Node{ServiceName: serviceName, Address: "127.0.0.1:8080"}

    return []*registry.Node{node}, nil
}
```

Users can specify the `Discovery` plugin to use:
```go
opts := []client.Option{
	client.ServiceName("myservice"),
	client.WithDiscoveryName("my_discovery")
}
```

If the implementation uses the default `Discovery`:

```go
discovery.DefaultDiscovery = &MyDiscovery{}
```

Then users only need to specify the service name:
```go
opts := []client.Option{
	client.WithServiceName("myservice"),
}
```

Load balancing, service router, and circuit breaker follows the same rule. You may refer to tRPC-Go's default implementation.

### Usage Example
With our discovery plugin `MyDiscovery` and import path `github.com/naming-plugin/my-discovery`, we can write a demo:

```go
package main

import (
	_ "github.com/naming-plugin/my-discovery"
)

func main() {
	proxy := pb.NewGreeterClientProxy()

	req := &pb.HelloRequest{
		Msg: "trpc-go-client",
	}
	rsp, err := proxy.SayHello(
		ctx, 
		req,
		client.WithServiceName("myservice"),
		client.WithDiscoveryName("my_discovery")
	)
	
	fmt.Println(rsp, err)
}
```

# OWNER
## misakachen