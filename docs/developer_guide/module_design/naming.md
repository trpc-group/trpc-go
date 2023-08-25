[TOC]

# tRPC-Go module: naming

## Background

The naming service is used to register service names and a set of active nodes, the nodes maintain online status with the remote naming service through mechanisms such as heartbeats. When registering with the naming service, nodes also need to additional information such as IP:Port, runtime environment, set, container, and other custom routing parameters.  
The caller uses the service name and the aforementioned additional information to obtain a list registered nodes, filter the node list, and select a node from the nodes that meet the requirements using a load balancing algorithm for the final request.  

The purpose of the naming service is to ensure the transparency of service location and avoid the caller from making calls with fixed IP:Port.

To be precise, the naming in tRPC-Go is a load balancing implementation built on top of the naming service. The tRPC client's default implementation integrates various components such as registry, selector, load balancer, and service router to achieve this capability.

## Principle

Let's first look at the design of the naming:

![naming design](/.resources/deverloper_guide/module_design/naming/naming.png)

From the class diagram, it can be seen that the naming package provides general framework support for service discovery and load balancing, including discovery, load balancing, registry, circuit breaker, and selector. The diagram also shows the approximate logical relationships.

Based on the above diagram, let's briefly introduce the approximate design and implementation.

## Implementation

### Discovery

Discovery defines the common interface for service discovery, which returns a list of service addresses based on the given service name.

Discovery supports custom implementation for business needs. The framework provides a default IpDiscovery that returns a list of IP addresses specified in the configuration file.

### Node

Node defines the data structure of a single service node.

### Registry

Registry defines the common interface for service registration and supports custom implementation for Register and Deregister operations.

### LoadBalancer

LoadBalancer defines the common interface for load balancing, which takes in an array of Nodes and returns a load-balanced Node.

trpc-go provides default implementations of load balancing algorithms such as round-robin and weighted round-robin. Businesses can also customize their own load balancing algorithms.

[Round-robin](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/loadbalance/roundrobin)  
[Weighted Round-robin](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/loadbalance/weightroundrobin)  

### ServiceRouter

ServiceRouter defines the interface for routing and filtering service Nodes, such as routing based on Set configuration, Namespace/Env environment, etc.

### Selector

Selector provides a common interface for obtaining a service node by service name. Within the selector, service discovery, load balancing, and circuit breaking isolation are called, making it an assembly of these capabilities.

trpc-go provides the default implementation of the selector, which uses default service discovery, load balancing, and circuit breaker. For more information, please refer to [https://git.woa.com/trpc-go/trpc-go/blob/master/naming/selector/trpc_selector.go](https://git.woa.com/trpc-go/trpc-go/blob/master/naming/selector/trpc_selector.go).

Default selector logic: Discovery -> ServiceRouter -> LoadBalance -> Node -> Business usage -> CircuitBreaker.Report.

### CircuitBreaker

CircuitBreaker provides a common interface for determining whether a service node is available, and also provides the ability to report the success or failure of the current service node.

### How to use

The framework defaults to integrating with the North Star for service discovery based on the service name. If the business needs to set the target when calling, service discovery will be based on the endpoint of the target.

```go
client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName)),
```

Target is the backend service address, in the format of name://endpoint, compatible with the old addressing methods, such as l5://modid:cmdid, cmlb://appid, and ip://ip:port.

The following example provides an implementation of custom service discovery for business use.

1. Implement the Selector interface.


```go
type exampleSelector struct{}
// Select obtains a backend node by service name.
func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
    fmt.Println(serviceName)
    if serviceName == exampleServiceName {
        return &registry.Node{
            Address: "127.0.0.1:8000",
        }, nil
    }
    return nil, errors.New("no available node")
}
// Report reports the success or failure of the current request.
func (s *exampleSelector) Report(node *registry.Node, cost time.Duration, success error) error {
    return nil
}
```

2. Register the custom selector

```go
var exampleScheme = "example"
func init() {
    selector.Register(exampleScheme, &exampleSelector{})
}
```

3. Set the service name

```go
var exampleServiceName = "selector.example.trpc.test"
client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName))
```


For more details, please refer to [selector demo](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/selector) 
