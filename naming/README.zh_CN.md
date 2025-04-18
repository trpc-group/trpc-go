[English](README.md) | 中文

## 背景

名字服务模块可以将节点注册到对应的服务名下。注册信息除了 `ip:port` 外，还会包含运行环境、容器以及其他自定义的元数据信息。调用方根据服务名获取到所有节点后，路由模块再根据元数据信息对节点进行筛选，最后，负载均衡算法从满足要求的节点中选出一个节点来进行最终请求。名字提供了服务管理的统一抽象，避免了直接使用 `ip:port` 带来的运维困难。

在 tRPC-Go 中，`register` 包定义了服务端的注册规范，`discovery`、`servicerouter`、`loadbalance`、`circuitebreaker` 则一起组成 `slector` 包并定义了客户端的服务发现规范。

## 原理

先来看下naming的整体设计：

![naming design](/.resources-without-git-lfs/naming/naming.png)

结合上图，我们来简单介绍下大致的设计、实现。

## 实现

### Discovery

Discovery 定义了服务发现类的通用接口，基于给定的服务名返回服务的地址列表。

Discovery 支持业务自定义实现。框架默认提供一个基于配置文件指定返回 ip 列表的 IpDiscovery。

### Node

Node 定义了单个服务节点的数据结构。

### Registry

Registry 定义了服务注册的通用接口，支持业务自定义实现 `Register`、`Deregister`。

### LoadBalancer

LoadBalancer 定义了负载均衡类的通用接口，从一组节点中选一个节点出来。

trpc-go 默认提供了轮询和加权轮询算法的负载均衡实现。业务可以自定义实现其他负载均衡算法。

- [一致性哈希](/naming/loadbalance/consistenthash)
- [轮询](/naming/loadbalance/roundrobin)
- [加权轮询](/naming/loadbalance/weightroundrobin)

### ServiceRouter

ServiceRouter 定义了对服务Node列表做路由过滤的接口。 例如根据Set配置路由、Namespace/Env环境路由等。

### Selector

Selector 提供通过服务名获取一个服务节点的通用接口。Selector 调用了服务发现，负载均衡，熔断隔离，可以说是这些能力的一个组装。

tRPC-Go 提供了 selector 的默认实现，使用默认的服务发现、负载均衡和熔断器。详见：[./selector/trpc_selector.go](/naming/selector/trpc_selector.go)

默认 selector 逻辑: Discovery->ServiceRouter->LoadBalance->Node->业务使用->CircuitBreaker.Report

### CircuitBreaker

CircuitBreaker 提供了判断服务节点是否可用的通用接口，同时提供上报当前服务节点成功/失败的能力。

### 如何使用

tRPC-Go 支持[北极星](https://github.com/trpc-ecosystem/go-naming-polarismesh)，可以根据服务名进行服务发现。假如业务方在调用时需要设置 Target，会根据 target 的 endpoint 去进行服务发现。

```go
client.WithTarget(fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName)),
```


Target 是后端服务地址 ，格式为 `name://endpoint`。比如，`ip://127.0.0.1:80` 会直接按 `ip:port` 访问 `127.0.0.1:80`；`polaris://service_name` 会通过北极星插件对服务名 `service_name` 进行寻址。

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


具体可参考 [selector demo](/examples/features/selector)。
