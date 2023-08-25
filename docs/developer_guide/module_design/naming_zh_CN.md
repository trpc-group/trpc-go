# tRPC-Go 模块：naming

## 背景

名字服务用来注册服务名以及一组活跃节点，节点与远程名字服务之间通过心跳等机制保持在线。节点注册到名字服务时还需包含一些附加信息，如 ip:port、运行环境、set、容器及其他自定义规则路由参数等信息。
调用方通过服务名以及上述附加信息，用来获取注册的节点列表、过滤节点列表，并结合负载均衡算法从满足要求的节点中筛选出一个节点用来进行最终的请求。

名字服务存在的目的就是为了保证服务位置的透明，避免调用方固定 ip:port 调用。

准确地说，tRPC-Go 中的 naming 是一种建立在名字服务基础之上的负载均衡实现，trpc client 默认实现综合了上述 registry, selector, loadbalancer, servicerouter 等多种组件来实现此能力。

## 原理

先来看下 naming 的整体设计：

![naming design](/.resources/deverloper_guide/module_design/naming/naming.png)

从类图中可以看出，naming 包提供服务发现、负载均衡等框架通用能力支持，主要包括了 discovery、loadbalance、registry、circuitbreaker、selector，图中也可以看出大致的逻辑关系。

结合上图，我们来简单介绍下大致的设计、实现。

## 实现

### Discovery

Discovery 定义了服务发现类的通用接口，基于给定的服务名返回服务的地址列表。

Discovery 支持业务自定义实现。框架默认提供一个基于配置文件指定返回 ip 列表的 IpDiscovery

### Node

Node 定义了单个服务节点的数据结构

### Registry

Registry 定义了服务注册的通用接口，支持业务自定义实现 Register、Deregister

### LoadBalancer

LoadBalancer 定义了负载均衡类的通用接口，传入 Node 数组，返回某一个负载均衡节点 Node。

trpc-go 默认提供了轮询和加权轮询算法的负载均衡实现。业务可以自定义实现其他负载均衡算法。

[轮询](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/loadbalance/roundrobin)  
[加权轮询](https://git.woa.com/trpc-go/trpc-go/tree/master/naming/loadbalance/weightroundrobin)  

### ServiceRouter

ServiceRouter 定义了对服务 Node 列表做路由过滤的接口。例如根据 Set 配置路由、Namespace/Env 环境路由等。

### Selector

Selector 提供通过服务名获取一个服务节点的通用接口，在 selector 中调用了服务发现，负载均衡，熔断隔离，可以说是这些能力的一个组装。

trpc-go 提供了 selector 的默认实现，使用默认的服务发现、负载均衡和熔断器。详见：[https://git.woa.com/trpc-go/trpc-go/blob/master/naming/selector/trpc_selector.go](https://git.woa.com/trpc-go/trpc-go/blob/master/naming/selector/trpc_selector.go)

默认 selector 逻辑：Discovery->ServiceRouter->LoadBalance->Node->业务使用->CircuitBreaker.Report

### CircuitBreaker

CircuitBreaker 提供了判断服务节点是否可用的通用接口，同时提供上报当前服务节点成功/失败的能力

### 如何使用

框架默认接入北极星，根据服务名去进行服务发现。假如业务方在调用时需要设置 Target，会根据 target 的 endpoint 去进行服务发现。

```go
client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName)),
```

Target 是后端服务地址，格式为 name://endpoint，兼容老寻址方式，如 l5://modid:cmdid cmlb://appid ip://ip:port。

下面例子给出了一个业务自定义的服务发现的实现。

1、实现 Selector 接口

```go
type exampleSelector struct{}
// Select 通过 service name 获取一个后端节点
func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
    fmt.Println(serviceName)
    if serviceName == exampleServiceName {
        return &registry.Node{
            Address: "127.0.0.1:8000",
        }, nil
    }
    return nil, errors.New("no available node")
}
// Report 上报当前请求成功或失败
func (s *exampleSelector) Report(node *registry.Node, cost time.Duration, success error) error {
    return nil
}
```

2、注册自定义 selector

```go
var exampleScheme = "example"
func init() {
    selector.Register(exampleScheme, &exampleSelector{})
}
```

3、设置服务名

```go
var exampleServiceName = "selector.example.trpc.test"
client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName))
```


具体可参考 [selector demo](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/selector) 
