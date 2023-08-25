# 前言
框架支持可插拔设置，用户可以根据自己的需要使用不同的名字服务插件，也可以根据自己的需求自行开发名字服务插件。

# 插件化设计
tRPC-Go 框架名字服务采用插件化设计，框架只有标准接口不涉及具体实现，用户可以根据自己的需要把对应的实现注册到框架。本文将会介绍如何实现一个名字服务插件。

名字服务包括服务发现、负载均衡、服务路由、熔断器等部分，服务发现的流程可以简化为：
- 1，Discovery 通过 service name 获取对应的节点列表
- 2，ServiceRouter 通过路由规则过滤调不符合要求的节点。
- 3，LoadBalance 通过负载均衡算法选取节点。
- 4，CircuitBreaker 根据熔断条件，判断选取出的节点是否符合要求，并进行上报。

# 名字服务插件实现

框架暴露的接口分为两种。

- 整体接口：名字服务作为整体注册到框架，整体接口的优势在于注册到框架比较简单，框架不关心名字服务流程中各个模块的具体实现，插件可以整体控制名字服务寻址的整个流程，方便做性能优化和逻辑控制。

- 分模块接口：服务发现、负载均衡、服务路由、熔断器等分别注册到框架，框架组合这些模块。分模块优势在于更加的灵活，用户可以根据自己的需要对不同模块进行选择然后自由组合，但同时会增加插件的实现复杂度。

两种实现都可以实现自定义的名字服务插件。

## 整体接口
整体接口不关心名字服务的具体实现，只是通过接口传入对应的名字服务 id，返回对应的被选中的被调服务一个节点。通过 `client.WithTarget` 就可以指定具体使用的服务发现插件。

tRPC-Go 框架的接口如下：
```go
// Selector 路由组件接口
type Selector interface {
	// Select 通过 service name 获取一个后端节点
	Select(serviceName string, opt ...Option) (*registry.Node, error)
	// Report 上报当前请求成功或失败
	Report(node *registry.Node, cost time.Duration, success error) error
}
```

根据框架的接口，如何实现自定义的名字服务插件？请看下面简单的名字服务插件实现。

```go

// 存储名字服务对应的节点信息
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

// 把实现注册到框架
func init() {
	selector.Register("example", &exampleSelector{})
} 

type exampleSelector struct{} 
// Select 通过 service name 获取一个后端节点
func (s *exampleSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
   	list, ok := store[serviceName]
    if !ok || len(list) == 0 {
    	return nil, errors.New("no available node")
    }
    
    return list[rand.Intn(len(list))]
}

// Report 上报当前请求成功或失败
func (s *exampleSelector) Report(node *registry.Node, cost time.Duration, success error) error {
    return nil
}
```

根据上面能实现，可以通过 `client.WithTarget("example://service1")` 来进行寻址。

### 使用示例
假设我们已经实现了上面的 `exampleSelector` 名字服务插件，并且引入的路径为 `github.com/naming-plugin/example-selector` 则我们可以如下使用：

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

## 分模块接口
这种方式提供了更多的灵活性，能够让使用者指定各个模块的配置参数，例如负载均衡方式，服务路由规则等。通过 `client.WithServiceName("trpc.app.server.service")` 就可以使用这种方式。
如果用户不指定对应的模块，则会采用默认实现。

- 服务发现默认实现，把 service name 当做 ip:port 处理。
- 服务路由默认实现，不做任何过滤操作。
- 负载均衡默认实现，随机负载均衡算法。
- 熔断器默认实现，不熔断处理。

下面看下以自定义实现 Discovery 为例：

服务发现的接口如下：

```go
// Discovery 服务发现接口，通过 service name 返回 node 数组
type Discovery interface {
    List(serviceName string, opt ...Option) (nodes []*registry.Node, err error)
}
```

```go
func init() {
	discovery.Register("my_discovery", &MyDiscovery{})
}

// MyDiscovery  ip 列表服务发现
type MyDiscovery struct{}

// List 返回原始 ip:port
func (*MyDiscovery) List(serviceName string, opt ...Option) ([]*registry.Node, error) {
    node := registry.Node{ServiceName: serviceName, Address: "127.0.0.1:8080"}

    return []*registry.Node{node}, nil
}
```

使用时只需要指定对应的 Discovery 即可。
```go
opts := []client.Option{
	client.ServiceName("myservice"),
	client.WithDiscoveryName("my_discovery")
}
```

如果把实现设置为默认的 Discovery 则不需要指定使用的 DiscoveryName。

```go
discovery.DefaultDiscovery = &MyDiscovery{}
```

使用时只需要指定 ServiceName 即可：
```go
opts := []client.Option{
	client.WithServiceName("myservice"),
}
```

负载均衡、服务路由、熔断器模块也是同样的处理方式，都可以参考框架的默认实现。

### 使用示例
假设我们已经实现了上面的 `MyDiscovery ` 插件，并且引入的路径为 `github.com/naming-plugin/my-discovery` 则我们可以如下使用：

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

