English | [中文](README.zh_CN.md)

## Background

Package `naming` can register nodes under the corresponding service name. In addition to `ip:port`, the registration information will also include the running environment, container and other customized metadata information. After the caller obtains all nodes based on the service name, the routing module filters the nodes based on metadata information. Finally, the load balancing algorithm selects a node from the nodes that meet the requirements to make the final request. The name provides a unified abstraction for service management and avoids the operation and maintenance difficulties caused by directly using `ip:port`.

In tRPC-Go, the `register` package defines the registration specification of the server, and `discovery`, `servicerouter`, `loadbalance`, and `circuitebreaker` together form the `slector` package and define the client's service discovery specification.

## Principle

Let's first look at the design of the `naming`:

![naming design](/.resources-without-git-lfs/naming/naming.png)

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
- [Consistent Hash](/naming/loadbalance/consistenthash)
- [Round-robin](/naming/loadbalance/roundrobin)
- [Weighted Round-robin](/naming/loadbalance/weightroundrobin)

### ServiceRouter

ServiceRouter defines the interface for routing and filtering service Nodes, such as routing based on Set configuration, Namespace/Env environment, etc.

### Selector

Selector provides a common interface for obtaining a service node by service name. Within the selector, service discovery, load balancing, and circuit breaking isolation are called, making it an assembly of these capabilities.

trpc-go provides the default implementation of the selector, which uses default service discovery, load balancing, and circuit breaker. For more information, please refer to [./selector/trpc_selector.go](/naming/selector/trpc_selector.go).

Default selector logic: Discovery -> ServiceRouter -> LoadBalance -> Node -> Business usage -> CircuitBreaker.Report.

### CircuitBreaker

CircuitBreaker provides a common interface for determining whether a service node is available, and also provides the ability to report the success or failure of the current service node.

### How to use

tRPC-Go supports [polaris mesh](https://github.com/trpc-ecosystem/go-naming-polarismesh), which can discovery nodes by service name. If the business sets the target when calling, the endpoint of the target will be used to discovery.

```go
client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName)),
```

Target is the backend service address in the format of `name://endpoint`. For example, `ip://127.0.0.1:80` will directly access `127.0.0.1:80` according to `ip:port`; `polaris://service_name` will address the service name `service_name` through the Polaris plug-in .

The following example provides an implementation of custom service discovery for business use.

1. Implement the Selector interface.
   ```go
   type exampleSelector struct{}
   // Select obtains a backend node by service name.
   func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node,    error) {
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

For more details, please refer to [selector demo](/examples/features/selector).
