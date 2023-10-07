[English](naming.md) | 中文

## 前言

像 tRPC-Go 大部分其他模块一样，名字服务模块也支持插件化。本文假定你已经阅读了 naming 包的 [README](/naming/README_zh_CN.md)。

## 插件化设计

tRPC-Go 提供了 [`Selector`](/naming/selector) interface 作为名字服务的入口，并提供了一个默认实现 [`TrpcSelector`](/naming/selector/trpc_selector.go)。`TrpcSelector` 把 [`Discovery`](/naming/discovery)、[`ServiceRouter`](/naming/servicerouter)、[`Loadbalance`](/naming/loadbalance) 和 [`CircuitBreaker`](/naming/circuitbreaker) 组合起来。对每一个小模块，框架都提供了其对应的默认实现。

通过[插件化](/plugin)方式，用户可以对 `Selector` 或它的各个小模块单独进行自定义。下面我们依次看看这是如何做到的。

## `Selector` 插件

`Selector` interface 的定义如下：
```go
type Selector interface {
	Select(serviceName string, opts ...Option) (*registry.Node, error)
	Report(node *registry.Node, cost time.Duration, err error) error
}
```

`Select` 方法通过 service name 返回对应的节点信息，可以通过 `opts` 传入一些选项。`Report` 上报调用情况，这些信息可能会影响之后 `Selector` 的结果，比如，对错误率太高的节点进行熔断。

下面是一个简单的固定节点的 `Selector` 实现：
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
使用时，需要匿名 import 上面的 plugin 包保证 `init` 函数成功注册 `Plugin`，并在 `trpc_go.yaml` 中加入下面的配置项：
```yaml
client:
  service:
    - name: xxx
      target: "my_selector://service1"
      # ... 忽略其他配置
    - name: yyy
      target: "my_selector://service2"
      # ... 忽略其他配置

plugins:
  selector:
    my_selector:
      nodes:
        service1: 127.0.0.1:8000
        service2: 127.0.0.1:8001
```
这样，client `xxx` 就会访问到 `127.0.0.1:8000`，client `yyy` 则会访问到 `127.0.0.1:8001`。

## `Discovery` 插件

`Discovery` 的接口定义如下：
```go
type Discovery interface {
    List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}
```
`List` 根据 service name 列出一组 nodes 供后续 ServiceRouter 和 LoadBalance 选择。

`Discovery` 插件的代码实现与 `Selector` 类似，这里不再赘述。

为了让 Discovery 生效，你还需要在下面两项选择其一：
- 如果你使用默认的 `TrpcSelector`，需要在 yaml 中加入下面配置：
  ```yaml
  client:
    service:
      - name: service1  # 注意，这里 name 直接填了 service1，而不是 xxx，我们将直接用该字段进行寻址
        # target: ...  # 注意，这里不能使用 target，而是要用上面的 name 字段去寻址
        discovery: my_discovery
  ```
- 如果默认的 `TrpcSelector` 不满足你的需求，可以像上节一样自定义 Selector，但是，你必须正确处理 `Select` 方法的 `Option`，即 `selector.WithDiscovery`。

## `ServiceRouter` `LoadBalance` 和 `CircuitBreaker` 插件

其他这些插件的实现方式与 `Discovery` 类似。要么使用 `TrpcSelector` 并在 `yaml.client.service[i]` 中设置对应的字段；要么在你自己实现的 `Selector` 中处理 `selector.WithXxx`。

## Polaris Mesh 插件

tRPC-Go 支持 Polaris Mesh 插件，你可以在[这里](https://github.com/trpc-ecosystem/go-naming-polarismesh)了解更多。
