English | [中文](naming.zh_CN.md)

## Background

Like most other modules in tRPC-Go, the name service module also supports plugin. This article assumes that you have read the [README](/naming/README.zh_CN.md) of the naming package.

## Pluggable Design

tRPC-Go provides the [`Selector`](/naming/selector) interface as the entrance to the name service, and provides a default implementation [`TrpcSelector`](/naming/selector/trpc_selector.go). `TrpcSelector` combines [`Discovery`](/naming/discovery), [`ServiceRouter`](/naming/servicerouter), [`Loadbalance`](/naming/loadbalance) and [`CircuitBreaker`](/naming/circuitbreaker ) combined. The framework provides a default implementation for each module.

By [plugin](/plugin), users can individually customize `Selector` or its various modules. Let's look at how this is done.

## `Selector` Plugin

The `Selector` interface is defined as follows:
```go
type Selector interface {
	Select(serviceName string, opts ...Option) (*registry.Node, error)
	Report(node *registry.Node, cost time.Duration, err error) error
}
```

The `Select` method returns the corresponding node information by service name, and some options can be passed in by `opts`. `Report` reports the call status, which may affect the results of `Selector` later, for example, circuit break nodes with too high error rates.

Here is a simple fixed-node `Selector` implementation:
```go
func init() {
    plugin.Register("my_selector", &Plugin{Nodes: make(map[string]string)})
}

type Plugin struct {
    Nodes map[string]string `yaml:"nodes"`
}

func (p *Plugin) Type() string { return "selector" }
func (p *Plugin) Setup(name string, dec plugin.Decoder) error {
    if err := dec.Decode(p); err != nil {
        return err
    }
    selector.Register(name, p)
    return nil
}

func (p *Plugin) Select(serviceName string, opts ...selector.Option) (*registry.Node, error) {
    if node, ok := p.Nodes[serviceName]; ok {
        return &registry.Node{Address: node, ServiceName: serviceName}, nil
    }
    return nil, fmt.Errorf("unknown service %s", serviceName)
}

func (p *Plugin) Report(*registry.Node, time.Duration, error) error {
    return nil
}
```
When using it, you need to anonymously import the above plugin package to ensure that the `init` function successfully registers `Plugin`, and add the following configuration to `trpc_go.yaml`:
```yaml
client:
  service:
    - name: xxx
      target: "my_selector://service1"
      # ... omit other configures
    - name: yyy
      target: "my_selector://service2"
      # ... omit other configures

plugins:
  selector:
    my_selector:
      nodes:
        service1: 127.0.0.1:8000
        service2: 127.0.0.1:8001
```
In this way, client `xxx` will access `127.0.0.1:8000`, and client `yyy` will access `127.0.0.1:8001`.

## `Discovery` Plugin

The interface definition of `Discovery` is as follows:
```go
type Discovery interface {
List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}
```
`List` lists a group of nodes based on service name for subsequent ServiceRouter and LoadBalance.

The code implementation of the `Discovery` plugin is similar to that of `Selector` and will not be described again here.

In order for Discovery to take effect, you also need to choose one of the following two options:
- If you use the default `TrpcSelector`, you need to add the following configuration to yaml:
  ```yaml
  client:
    service:
      - # Note that the name here is directly filled with service1 instead of xxx.
        # We will directly use this field to select.
        name: service1
        # Note that target cannot be used here, but the name field above must be used to select.
        # target: ...
        discovery: my_discovery
  ```
- If the default `TrpcSelector` does not meet your needs, you can customize the Selector as in the previous section. However, you must correctly handle the `Option` of the `Select` method, that is, `selector.WithDiscovery`.

## `ServiceRouter` `LoadBalance` and `CircuitBreaker` Plugins

These plugins are implemented similarly to `Discovery`. Either use `TrpcSelector` and set the corresponding fields in `yaml.client.service[i]`; Or handle `selector.WithXxx` in your own implementation of `Selector`.

## Polaris Mesh Plugin

tRPC-Go supports Polaris Mesh plugin. You can read more [here](https://github.com/trpc-ecosystem/go-naming-polarismesh).
