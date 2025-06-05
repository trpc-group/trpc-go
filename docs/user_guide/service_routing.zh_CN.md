本文内容基于 [xiaobaihe](https://km.woa.com/user/xiaobaihe) 的原始 km 文章 [如何理解 trpc 框架中的服务路由](https://km.woa.com/articles/show/553801) ，已获得修改授权。

# 1. `WithServiceName` 和 `WithTarget` 带来的困惑

假如主调服务和被调服务都在北极星注册了，那么这两种路由方式对应服务路由规则有什么区别？

在看下文前，建议大致阅读 [北极星的路由规则](https://iwiki.woa.com/pages/viewpage.action?pageId=102467866)，理解北极星的出流量规则和入流量规则，因为两种寻址方式在规则上有些许交集。

## 1.1 `WithServiceName` 的服务路由规则

这里用 123 平台部署的服务为例，先描述一下 123 平台的在服务首次创建时：

每一个在 123 平台 Development 环境启动的服务（这里用 trpc.xxx.yyy.AAA 指代此服务），123 平台都会自动通过 123 名字服务插件，在北极星平台对应的 Development 环境，创建一个出流量规则。其中这个出流量路由规则的设计如下：

![outbound_traffic_routing_rules](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/outbound_traffic_routing_rules.png)

此条路由规则的含义表示为：当请求从服务 trpc.xxx.yyy.AAA 发出（即主调为 trpc.xxx.yyy.AAA），并且请求携带的 namespace 为 Development，环境名为 123abc 时，对于这种请求，路由选择器会从此条路由规则对应的实例匹配规则中找出可供选择的节点。以上图为例，路由选择器会优先选择被调服务环境名为 123abc 的节点作为被请求的节点，如果环境名为包含 123abc 的被调服务节点不存在，则会退而求其次查找环境名为 test 的被调服务节点进行路由。

![explanation_outbound_traffic_routing_rules](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/explanation_outbound_traffic_routing_rules.png)

> 注意：以下的 A 服务/B 服务/C 服务假设均为 trpc-go 服务。

### 1.1.1 如果主调服务为此次请求的首个服务

即请求从这个服务 A 接收服务开始：请求 --> A 服务 --> B 服务 --> C 服务。

#### a. 默认开启服务路由

**注意：**在开启服务路由时，要求主调提供 `Namespace` 和 `ServiceName` 信息，而用户调用的时候可能是纯客户端（没有完整加载框架的配置），或非纯客户端（有完整加载框架的配置），这两种情况需要分别考虑：

* 在非纯客户端的情况下，`Namespace` 和 `ServiceName` 信息会自动在 `client.Invoke` 中自动填充，用户不需要有额外操作：

```go
func (c *client) getServiceInfoOptions(msg codec.Msg) []selector.Option {
    if msg.Namespace() != "" {
        return []selector.Option{
            selector.WithSourceNamespace(msg.Namespace()),  // 填充 Namespace
            selector.WithSourceServiceName(msg.CallerServiceName()), // 填充 ServiceName
            selector.WithSourceEnvName(msg.EnvName()), 
            selector.WithEnvTransfer(msg.EnvTransfer()), 
            selector.WithSourceSetName(msg.SetName()), 
        }
    }
    return nil
}
```

* 在纯客户端场景下，用户的 `ctx` 中没有 `Namespace` 和 `ServiceName` 信息，需要显式设置，类似于：

```go
// 在 proxy 调用前设置：
msg := trpc.Message(ctx)
msg.WithCalleeServiceName("trpc.xxx.yyy.AAA")  // 注意这里是 Callee 
msg.WithNamespace("Development")
proxy := pb.NewHelloClientProxy(ctx)
proxy.SayHello(ctx, req)

// 或者在 client filter 中设置：
proxy := pb.NewGreeterClientProxy(client.WithFilter(
    func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
        msg := trpc.Message(ctx)
        msg.WithCallerServiceName("trpc.xxx.yyy.AAA")  // 注意这里是 Caller 
        msg.WithNamespace("Development")
        return next(ctx, req, rsp)
    }))
proxy.SayHello(ctx, req)
```

为什么在 proxy 调用前需要设置 Callee 的信息，而在 client filter 中需要设置则是 Caller 的信息？

因为 [桩代码](https://git.woa.com/trpc-go/trpc-go/blob/v0.18.6/testdata/helloworld.trpc.go#L98) 中会调用

```go
ctx, msg := codec.WithCloneMessage(ctx)
```

里面会将 msg 中的 Callee 的信息复制到 Caller 上，因此在桩代码调用前需要设置的是 Callee 的信息（即 `msg.WithCalleeServiceName`）， 桩代码内部走 client filter 时则需要设置 Caller 的信息（即 `msg.WithCallerServiceName`）。

**i）不显式设置透传环境**

主调服务 A 接收客户端的请求，因为是首个 trpc 服务，则因为没有上游的任何透传环境（暂不需要理解什么叫透传环境，后面会解释），则会根据 CallerSerivceName（主调服务名）/ CallerNamespace（主调 Namespace）/ CallerEnvName（主调环境名）在北极星上找到对应的出流量请求匹配规则，这里仍然用上文中 trpc.xxx.yyy.AAA 的出路由规则为例，会找到包含 123abc 和 test 这两个环境名的实例匹配规则。并且会根据 123abc 和 test 这两个实例匹配规则优先级顺序，先匹配环境名为 123abc 的节点，如果找不到才会再与环境名为 test 的节点匹配，如果找不到满足的节点则会报 "filter instances without tranfer env err" 错误。

如果存在满足对应实例匹配规则的节点，则 trpc.xxx.yyy.AAA 服务会与对应的满足规则下游节点建立链接发起请求，并且将 "123abc" 和 "test" 这两个环境变量按照在北极星实例匹配规则的优先级顺序，连接成以逗号分割的字符串 "123abc,test"，放入 trpc 协议的透传字段 "trpc-env" 这个透传字段中，然后像下游服务请求，整个逻辑全在 `client.Invoke` 中通过 trpc 框架完成，大致流程图如下：

![without_enabling_transparent_transmission_environment](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/without_enabling_transparent_transmission_environment.png)

代码参数示例如下：

```go
opts := []client.Option{
    client.WithCallerNamespace("Development"),       // 设置主调的 Namespace, 默认 trpc 框架会自动填写
    client.WithCallerEnvName("123abc"),              // 设置主调的环境名，默认 trpc 框架会自动填写
    client.WithCallerServiceName("trpc.xxx.yyy.AAA"),// 设置主调的服务名（对应服务在北极星的服务名）默认 trpc 框架会自动填写
    client.WithServiceName("trpc.xxx.yyy.BBB"),      // 设置被调的服务名（对应服务在北极星的服务名）
}
```

**ii）显式设置透传环境**

上述的规则，只在 trpc.xxx.yyy.AAA 服务没有显式指定 trpc-env（即没有在代码中显式使用设置 `WithEnvTransfer` 方式设置透传环境值）生效。

```go
msg := trpc.Message(ctx)
msg.WithEnvTransfer("123abc,456def,test")
```

如上面的实例，假如 trpc.xxx.yyy.AAA 在向下游请求的时候，主动设置了透传的环境参数为上述 "abc123,def456,test"，则会直接不再使用任何北极星上由 123 插件生成的实例匹配规则，框架会自动构造一个新的路由规则。规则即：依次匹配满足环境名为 abc123/def456/test 的节点，并且 abc123 优先级最高，后续 def456/test 优先级依次递减。如果 abc123 匹配到则跳过后续匹配，否则继续向下匹配，直到匹配到一个满足规则的节点。如果找不到满足的节点，则会报 "filter instance with env err" 错误。

![enable_transparent_transmission_environment](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/enable_transparent_transmission_environment.png)

#### b. 关闭服务路由

首先明确一个问题是，什么叫做关闭服务路由？这个名词其实在很多文档中都没有解释清楚。这里简单的理解，就是不再根据主调的 CallerNamespace/CallerEnvName/CallerServiceName 来找对应请求匹配规则，而是直接根据被调的 Namespace/CalleeEnvName/ServiceName，过滤出满足这些变量可用的下游节点，代码参数示例如下：

```go
opts := []client.Option{
    client.WithNamespace("Development"),        // 设置被调的 Namespace
    client.WithCalleeEnvName("123abc"),         // 设置被调服务的环境名
    client.WithServiceName("trpc.xxx.yyy.BBB"), // 设置被调的服务名（对应服务在北极星的服务名）
    client.WithDisableServiceRouter(),
}
```

上面的代码则会先通过北极星接口查询到满足 Namespace=Development/env=123abc/北极星服务名=trpc.xxx.yyy.BBB 的节点列表，然后再根据负载均衡策略与其中的一个节点建立链接。

如果代码中去掉 `client.WithCalleeEnvName("123abc")`，则表示查询满足 Namespace=Development/北极星服务名=trpc.xxx.yyy.BBB 的节点列表，这个时候就会发现，主调会随机与满足 Namespace=Development/北极星服务名=trpc.xxx.yyy.BBB 的节点建立连接，即建立连接的节点对应的 env 环境名随机不确定。

另一个关闭服务路由的场景就是在 Development 需要调用 Production 的服务时，可以通过指定下游 Namespace 为 Production 并一定要关闭服务路由完成。

```go
opts := []client.Option{
    client.WithNamespace("Production"), // 设置被调的 Namespace 为 Production
    client.WithDisableServiceRouter(),
    client.WithServiceName("trpc.xxx.yyy.BBB"),
}
```

### 1.1.2 如果主调服务非此次请求的首个服务

即请求在被服务 B 接收前，已被大于 1 个 trpc 服务接收并转发：请求 --> A 服务 --> B 服务 --> C 服务。

#### a. 默认开启服务路由

同样以 trpc.xxx.yyy.AAA 服务为例，假设此服务如上述 1.1.1-a-i 中描述使用 ServiceName 方式请求下游，并且开启了服务路由，则 "123abc,test" 这个透传环境字段会通过 trpc 协议传递到下游的 trpc.xxx.yyy.BBB 服务。trpc.xxx.yyy.BBB 服务在接收到带有透传环境的请求时，会直接使用透传环境信息构造路由规则（不会使用在任何北极星的路由规则）。

![enable_service_routing](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/enable_service_routing.png)

例如，假设 trpc.xxx.yyy.AAA 服务透传下来的透传环境为 "123abc,test"，则 trpc.xxx.yyy.BBB 再向下游请求时（使用 `WithServiceName` 并使用默认开启服务路由配置），会优先选择匹配环境名为 123abc 的服务，如果不存在则会再匹配环境名为 test 的服务，假设透传环境包含了多个逗号分割的环境，则优先级按照逗号分割后的数组索引，依次降低（降低是说匹配的优先级，不是指北极星中优先级的数字大小）。

当然 trpc.xxx.yyy.BBB 也可以和 1.1.1-a 中的 trpc.xxx.yyy.AAA 一样，显示设置透传环境，显式设置透传环境则会覆盖掉原有的从 trpc.xxx.yyy.AAA 服务透传下来的环境。

注意这里覆盖掉上游透传下来的透传环境，有两种覆盖，一个是覆盖为空，一个是覆盖为新的非空值。而两个覆盖的效果会完全的不一样：

1. 覆盖为空：`msg.WithEnvTransfer("")`，则会变成和 1.1.1-a-i 中描述的默认情况下 trpc.xxx.yyy.AAA 路由规则一致，即会根据 CallerSerivceName（trpc.xxx.yyy.BBB）/ CallerNamespace / CallerEnvName 在北极星上找到对应的出流量请求匹配规则，根据此请求匹配规则，获取到请求匹配对应的实例匹配规则。
2. 覆盖为非空值：`msg.WithEnvTransfer("123abc,456def,test")`，则会变成和 1.1.1-a-ii 中描述的 trpc.xxx.yyy.AAA 路由规则一致，会根据透传的环境依次按照优先级匹配。

#### b. 关闭服务路由

对于这种情况，和 1.1.1-b 一样，不再赘述。

### 1.1.3 如何理解 trpc 主张的多环境路由理念？

从上面关于 `WithServiceName` 开启服务路由规则的描述可以总结一个规律。假设一个调用链为 A->B->C->D，如果 ABCD 四个服务均使用 `WithServiceName` 寻址，且为 Development 环境开启默认的服务路由。如果服务 A 处于特性环境 env:123456，并且对应存在基线 env:test 环境，且在北极星上存在一个请求匹配规则为 env:123456，对应实例匹配规则为 env:123456;priority:0 和 env:test:priority:1。

那么请求在经过服务 A 后，透传 trpc-env 后（”123456,test“）会一直将此透传环境传递到 D, 即从 A 到 D 的路由，只会在 env:123456 和 env:test 这两个之间选择，即从 B->C->D 的路由规则不再和北极星相关，只会根据 trpc-env 透传过来的环境，构造出 trpc 自己的路由规则。我们用一张图可以更好的表示这个概念。

![multi-environment_routing](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/multi-environment_routing.png)

可以看到请求总会优先路由到环境为 123456 的服务中，这样如果我们以 test 作为基线测试环境，以 123456 作为某一个需求的特性环境，则如果请求从头到尾都是用 `WithServiceName` 方式进行路由，则总可以保证请求优先被特性环境的服务处理，在特性环境没有对应服务时，才会被基线测试环境的服务处理。

除了可以用 `WithServiceName` 方式进行上述的场景应用，当然还可以通过显示设置 trpc-env 进行一些 mock 服务的路由，使请求优先路由到一些集成测试的 mock 服务上，例如下图，我们单独在 mock 这个环境名中设置 B/D 两个服务为一个 mock 服务。通过在客户端请求中显示设置 trpc-env 透传环境为 "mock,123456,test"，则可以使请求优先请求到 mock 环境对应的服务，进而做到在集成测试等场景优先请求到 mock 服务来进行测试。

```go
msg := trpc.Message(ctx)
msg.WithEnvTransfer("mock,123456,test")
```

![multi-environment_routing_with_mock](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/multi-environment_routing_with_mock.png)

所以我觉得使用 `WithServiceName` 方式是需要遵守一定的约定，比如一个需求的开发，主张在一个特性环境完成，这样可以最少的改变路由规则。

#### 一些跨特性环境场景遇到的问题

但是往往在跨团队请求的时候，或者多人用自己的特性环境同时开发不同的服务时，让被调服务（例如下图的 E 服务）部署在同一个环境下可能不太实际，所以只能通过两种方法来处理这种问题。以下图为例，假如要在 123456 环境的服务 C 中调用 abcdef 的服务 E，可以用两种方法完成。

![cross_feature_environment](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/cross_feature_environment.png)

1. 在 C 服务中显示设置 trpc-env 为 abcdef，即 `msg.WithEnvTransfer("abcdef")`。
2. 在 C 服务中通过代码或者配置方式，关闭服务路由并设置下游的环境名。

    ```yaml
    # 代码方式请见 1.1.1-a-ii 节
    # trpc-go.yaml 配置
    client:
      service:
        - name: "trpc.xxx.yyy.EEE"   # name 是被调服务在北极星的名字（name 和 callee 可以不一样）
          callee: "trpc.xxx.yyy.EEE" # 注意：callee 是被调服务 proto 中设置的服务名，并不是对应服务在北极星的名）如果 callee 不填写，默认会用上面的 name 对应的值作为 callee, 框架会将 callee 对应的调用下游的请求，自动填充上配置：下游的环境名并禁用服务路由
          env_name: abcdef
          disable_servicerouter: true
    ```

    > 假如配置了 `disable_servicerouter` 之后还是有问题？很有可能是这个配置根本没生效，因为 callee 配置的不对，callee 到底如何确定？阅读：[client 配置中的 callee 和 name 的区别是什么？](https://iwiki.woa.com/p/99485621#q7-client-%E9%85%8D%E7%BD%AE%E4%B8%AD%E7%9A%84-codeab21e55869c55bd8637c3732df94508c-%E5%92%8C-code4c7d8e8ca318a9863d99e9737c57bdfa-%E7%9A%84%E5%8C%BA%E5%88%AB%E6%98%AF%E4%BB%80%E4%B9%88%EF%BC%9F)

## 1.2 `WithTarget` 的服务路由规则

`WithTarget` 主要是提供了更多路由选择器，比如可以利用 `ip://<ip>:<port1>,<ip2>:<port2>` 语法在多个 ip 列表中随机选择一个进行服务调用，或者使用 `dns://<domain>:<port>` 进行 dns 域名寻址。本节主要重点讨论，使用 `WithTarget` 方式进行北极星寻址，即显示指定使用 `WithTarget("polaris://trpc.xxx.yyy.AAA")` 方式寻址与上文中 `WithServiceName` 的区别。

官方文档对于使用 `client.WithTarget("polaris://trpc.xxx.yyy.AAA")` 寻址是这么解释的：“使用 `client.WithTarget` 寻址，则会整个使用北极星的 `GetOneInstance` 接口，不会关心内部的各个组件的配合”。本节主要探讨的是在使用 `WithTarget` 北极星寻址的时候，具体服务路由是怎么完成的（负载均衡，熔断器这些不会涉及）。

在使用 `WithTarget` 方式进行北极星路由时，事实上就是不再会使用任何 trpc-env 中传递的环境信息进行路由，而完全使用北极星上配置的路由规则进行路由。上文中在描述 `WithServiceName` 方式寻址可以看出，在涉及到查询北极星路由规则的时候，只是使用了北极星的出流量规则，而没有利用任何的入流量规则，所以一种比较大的区别就是使用 `WithTarget` 方式，可以利用到北极星的入流量规则。

根据 [北极星的动态路由的原则](https://iwiki.woa.com/pages/viewpage.action?pageId=102467866)，A->B 在主调 A 配置了出流量规则，对应的被调 B 配置了入流量规则，则只会应用 B 的入流量规则，如果有需要利用入流量规则的场景，则必须要使用 `WithTarget` 方式指定北极星路由。比如下面的 AAA->BBB, 只有当请求为来自于 Development，并且请求携带的 env 为 abc123 时，才能被处于 Development 且 env 为 test 的 BBB 请求接收，并且完全忽视 AAA 的出流量规则。

![with_target_outbound](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/with_target_outbound.png)

![with_target_inbound](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/service_routing/with_target_inbound.png)

# 2. 关于关闭服务路由的问题

在使用 `WithServiceName` 时，可以使用 disable_servicerouter 方式，通过关闭服务路由能力，指定下游的 Namespace 为 Development 或者 Production, 来选择请求到测试环境还是正式环境。

如果在 `WithTarget` 方式下，想实现 Development 调用 Prodcution，则有两种方法可以完成：

1. 同 1.1.1-b 中相似：

    ```go
    opts := []client.Option{
        client.WithNamespace("Production"), // 设置被调的 Namespace 为 Production
        client.WithDisableServiceRouter(),
        client.WithTarget("polaris://trpc.xxx.yyy.BBB"),
    }
    ```

2. 在不关闭服务路由的情况下，可以在北极星被调服务的 Production 环境入流量方向设置规则，由于入流量规则优先级高，可以在入流量规则中设置请求匹配规则 `Namespace:Development, env: <测试环境名>`，实例匹配规则为默认即可。通过这种方式可以将主调测试环境流量引入被调正式环境。
3. 另外如果全局均不需要服务路由功能，一个一劳永逸的方法则可以直接在 trpc 框架配置中设置 selector 北极星插件参数：

    ```yaml
    plugins:
      selector:
        polaris:
          address_list: ${polaris_address_grpc_list}
          protocol: grpc
          enable_servicerouter: false # 默认不配置是 true, 表示打开服务路由能力
          discovery:
            refresh_interval: ${polaris_refresh_interval}  
    ```

# 3. 再检查一下 proto 中服务名和对应的北极星名字是否一样

从我的经验上来看，其实多数情况下，都不会遇到路由设置不对导致服务请求异常，而更多的时候是因为在拼写错误导致 proto 中定义的服务名（即 package.service）与注册在北极星上的名字服务不一致。我们往往使用会使用配置文件的方式来加载下游服务的路由规则等配置。如下：

```yaml
client:
  service:
    - name: trpc.xxx.yyy.AAA
      namespace: Development
      disable_servicerouter: true
      # ...
```

trpc 框架是如何找到我调用下游时的路由配置呢？本质上 trpc 框架在加载此配置的时候，会默认对配置进行补全，如果对应的 service 中没有没有显示设置 callee 字段，则会默认用 name 对应的字段填充 callee，对于上面的例子就是用 trpc.xxx.yyy.AAA 填充 callee。然后框架会根据每个 client 中下游的 serivce 的配置，按照 callee 为 key, 对应的配置为 value，生成 map。代码中在调用下游的时，会自动根据调用下游时，下游服务 proto 中定义的 package.service 在这个 map 中查找对应的配置，并填充。

那么假如注册在北极星的服务名与定义在 proto 中的 pacakge.service 不相同时，如果不显示指定 callee 字段为下游 proto 中定义的 package.service 时，就会出现在上述的 map 重找不到对应路由配置，这个时候框架会根据当前主调情况采用默认的路由行为，而这个时候由于就可能出现请求失败或者请求到的服务和设置的下游服务不相同等异常情况。

> callee 到底如何确定？阅读：[client 配置中的 callee 和 name 的区别是什么？](https://iwiki.woa.com/p/99485621#q7-client-%E9%85%8D%E7%BD%AE%E4%B8%AD%E7%9A%84-codeab21e55869c55bd8637c3732df94508c-%E5%92%8C-code4c7d8e8ca318a9863d99e9737c57bdfa-%E7%9A%84%E5%8C%BA%E5%88%AB%E6%98%AF%E4%BB%80%E4%B9%88%EF%BC%9F)

```yaml
client:
  service:
    - name: trpc.xxx.yyy.AAA
      callee: trpc.xxx.yyy.AAAA # 显示填充 callee 为下游真实请求对应的 proto 中定义 <package>.<service>
      namespace: Development
      disable_servicerouter: true
      # ...

# 定义的 proto
# package trpc.xxx.yyy;
# service AAAA {
#     // ...
# }
```

所以我们在很多地方如果请求的被调服务为 trpc 服务，则建议被调服务开发的时候，proto 中的 pacakge.service 与对应在北极星上的名字服务一致，这样可以减少很多异常。

# 4. 小结

1. 建议：如果服务的上下游均是使用 trpc 框架构建的服务，则使用 `WithServiceName` 进行寻址，使用 target 寻址多见于纯客户端方式，即不通过 trpc 框架插件注册北极星 selector 时使用。纯客户端示例代码如下：

    ```go
    import (
        "git.code.oa.com/trpc-go/trpc-naming-polaris/selector"
        "git.code.oa.com/trpc-go/trpc-go/client"
    )

    func init() {
        selector.RegisterDefault()
    }

    func main() {
        // ...
        opts := []client.Option{
            client.WithNamespace("Development"),
            client.WithTarget("polaris://trpc.xxx.yyy.AAAA"),
        }
        // ...
    }
    ```

2. `WithServiceName` 在跨多个特性环境的时候，建议使用关闭服务路由能力，通过指定下游环境名方式将请求定向到指定的节点中。
3. 在路由异常的时候，首先检查 proto 中定义的 package.service 是否与注册在北极星对应服务的名字一样，然后再进行分析。

# 5. Set 路由

在了解了前文 `WithTarget` 和 `WithServiceName` 的区别之后，我们来讲一下 set 路由的使用。

首先，最重要的一点是：set 路由的使用需要在 `WithServiceName` 的用法下进行，即：`WithTarget` 不支持按 set 调用。

其次，要明确 set 使用的两种方式：

1. 通过自动判断是否开启 set 从而使用 set 功能。
  * 这种方式下一定不能使用 disable_service_router=true 的配置，否则 set 规则会失效。
2. 使用 `client.WithCalleeSetName` （或 yaml 中的 client 配置）来指定 set 调用。
  * 这种情况下可以使用 disable_service_router=true 的配置，相当于仅仅是筛选被调的节点（带有哪个 set 标识），不走 set 路由规则。

一般来说，第一种方式配合 disable_service_router=true 的配置，第二种方式配合 disable_service_router=false 的配置。

更加详细的逻辑可以参考[实现](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/master/servicerouter/servicerouter.go)。

下面详细解释一下这两种使用方式：

## 5.1 自动判断

这种方式为最简单的场景。

* 在 123 平台上，用户只需要在主调服务 serviceA 创建一个 set 名为 xx.sz.1 的节点 A（创建带 set 节点的具体操作可以参考文档 [tRPC-Go Set 路由](https://iwiki.woa.com/pages/viewpage.action?pageId=118669392) ），然后再在被调服务 serviceB 创建一个 set 名为 xx.sz.1 的节点 B，那么 A 节点对 serviceB 调用时自动就会调用到节点 B，注意：此时不需要给定任何的 option（不需要指定 target，不需要带什么 callee metadata 之类的东西，就是简单的调用）。
* 其背后的原理在于 123 平台在创建带 set 节点时，会执行以下两个操作：

  1. 将节点注册到北极星上时，带上两个实例标签：`internal-enable-set: Y` 和 `internal-set-name: xx.sz.1`。
  2. 将 `trpc_go.yaml` 中全局配置里的两个 set 字段进行填充，这两个字段填充之后，这个节点所有发往下游的请求都会自动带上一个 `client.WithCallerSetName("xx.sz.1")` 的 `option`，从而用于筛选出下游的对应 set 节点，并自动走 set 规则（即文档 [tRPC-Go Set 路由](https://iwiki.woa.com/pages/viewpage.action?pageId=118669392) 里提到的 set 调用规则）。

      ```yaml
      global:                  # 全局配置
        enable_set: Y          # 是否启用 set
        full_set_name: xx.sz.1 # set 名
      ```

* 在其他平台上，用户想要创建一个带 set 的节点的话，则需要手动把 123 平台自动做的事情手动做一遍，即：1. 在注册北极星时带两个实例标签，2. 将 `trpc_go.yaml` 的全局配置中的两个字段进行相应的填充。

在这种模式下，通配符可以生效。比如节点 A 的 set 为 `xx.sz.*` 的话，他可以调用到处于 `xx.sz.1`、`xx.sz.2`、`xx.sz.*` set 中的节点。

## 5.2 指定 set 调用

指定 set 调用指的是使用 `client.WithCalleeSetName` （或 yaml 中的 client 配置）来进行调用，注意这里是 `Callee`，有别于方式一里的 `Caller`，在这种方式下，方式一中的 set 规则会失效，调用会严格筛选出符合给定的 `CalleeSetName` 的节点（这个 set name 的第一段不一定和主调的 set name 的第一段相同）。并且，在这种模式下，通配符会失效，这意味着假如节点 A 的 set 为 `xx.sz.*` 的话，他只能调用到处于 `xx.sz.*` set 中的节点。

# 6.FAQ

**注意：PCG 123 平台的北极星名字服务配置都是自动化的，不要到北极星控制平台乱操作。北极星插件老版本有 bug，先尝试升级到最新版看是否能解决问题**

## 6.1 北极星寻址失败相关问题

### Q1 - not found service?

* 被调服务不存在：请到 <http://polaris.woa.com/#/polaris/services> 查看被调服务是否存在。
* 主调服务没上 123，但是却开启服务路由（默认是开启的，可配置成关闭）：

  ```text
  Polaris-1006(ErrCodeServerError): multierrs received for GetInstances request, namespace: Production, service trpc.app.server.service1, cause: 1 error occurred:
  SDKError for {ServiceKey: {namespace: "Development", service: "trpc.app.server.service2"}, Operation: `sourceRoute`}, detail is Polaris-1006(ErrCodeServerError): Response from {ID: 3556264478, Service: {ServiceKey: {namespace: "Polaris", service: "polaris.discover"}, ClusterType: discover}, Address: 9.157.132.141:8090}: not found service
  ```

  解决方案：
  1. 关闭服务路由

  * 全局关闭

    ```yaml
    plugins:                                         # 插件配置
    selector:                                        # 针对 trpc 框架服务发现置
    polaris:                                         # 北极星服务发现的配置
      protocol: grpc                                 # 名字服务远程交互协议类型
      enable_servicerouter: false                    # 是否开启服务路由，默认开启
    ```

  * 单次请求关闭

    ```go
    opts := []client.Option{
        client.WithNamespace("Production"),
        client.WithTarget("trpc.app.server.service"),
        client.WithDisableServiceRouter(),
    }
    ```

  2. 主调服务上线

* 此外，naming-polaris 从 v0.3.1 => v0.3.2 添加了额外的逻辑导致主调只要存在 service name 或 namespace 就会拉主调的北极星规则，从而要求主调在北极星上有注册，见问题 [fail:type:framework, code:131, msg:client Select: get one instance err: polaris-go version: 0.12.6, Polaris-1006? - wineguo 的回答](http://mk.woa.com/q/293838/answer/120608)。按照上面所说，关闭服务路由或者主调服务在北极星上进行注册即可。
* 如果是因为升级 v0.8.2 => v0.8.4 引起的，之前没有问题，那把框架和七彩石插件升级到最新版即可：框架 >= v0.8.5；七彩石 >= v0.1.22。

### Q2 - missing port in address?

1. 检查下有没有这句

    ```go
    import (
        _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
    )
    ```

2. 确保 trpc_go.yaml 框架配置有配置北极星，在 123 平台都会自动生成可直接使用，不需要自己管配置文件，请直接发布到 123 平台，不要自己在本地测试。
3. 纯客户端模式需要自己注册，具体看 [这里](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/master/selector)。
4. client 的 proto name 和 service name 不一致，需要在 trpc_go.yaml 中明确指定 callee name `callee: xxxxxx`，其中 `xxxxxx` 可以在 x.trpc.go 桩代码中的目标 ServiceDescriptor 中的 `ServiceName` 字段上找到（可以参考 [这里](https://mk.woa.com/q/287524) ）。

### Q3 - route rule not match?

北极星支持服务路由功能，支持配置规则。123 平台上的服务默认生成路由规则，具体请查看 [这里](https://iwiki.woa.com/pages/viewpage.action?pageId=99485673&src=contextnavpagetreemode)。

具体解决方案：

1. 服务部署到同一个环境下。
2. 全局关闭，修改插件配置：`enable_servicerouter: false`。
3. 单次请求关闭服务路由：设置 `client.WithDisableServiceRouter` 选项：

    ```go
    opts := []client.Option{
        client.WithNamespace("Production"),
        client.WithTarget("trpc.app.server.service"),
        client.WithDisableServiceRouter(),
    }
    ```

4. 如果是使用指定环境请求，需要保证关闭服务路由和指定对方的环境：

    ```go
    opts := []client.Option{
        client.WithNamespace("Development"),
        client.WithServiceName("trpc.app.server.service"),
        client.WithCalleeEnvName("62a30eec"),
        client.WithDisableServiceRouter()
    }
    ```

5. type:framework, code:131, msg:client Select: filter instance with env err

    ```text
    type:framework, code:131, msg:client Select: filter instance with env err: Polaris-1012(ErrCodeRouteRuleNotMatch): route rule not match, rule status: sourceRuleFail, sourceService {service: trpc.co_game.task_manager.task_queue, namespace: Development, metadata: map[env:pre]}, dstService {service: trpc.co_game.hpjy_story_task.story_task_http, namespace: Development}, notMatchedSource is {}, notMatchedDestination is {"destinations":[{"service":"*","namespace":"Development","metadata":{"env":{"value":"08decd85"}},"priority":0,"weight":100}]}, zeroWeightDestination is {}, please check your route rule of service Development:trpc.co_game.task_manager.task_queue
    ```

    这个错误表示上游环境信息透传到了下游，08decd85 环境的请求到了 pre 环境，把上游的环境变量 08decd85 透传到了 pre，pre 将会优先使用 08decd85 去匹配下游。可以通过以下方法清空上游带来的环境变量：

    ```go
    // 框架的 ctx
    msg := trpc.Message(ctx)
    msg.WithEnvTransfer("")
    ```

### Q4 - service or namespace is empty?

* 确保 service 和 namespace 设置了。
* 如果是在配置中设置了 callee 确保 callee 和被调服务名保持一致。
* 确保客户端调用是在 `trpc.NewServer` （也就是框架配置加载之后）进行的，参考 [trpc-go 调用 redis 报错 service or namespace is empty, namespace: , service: xxxxxx？ - wineguo 的回答](http://mk.woa.com/q/294409/answer/121627)。

### Q5 - fail to get instances, err is Polaris-1004(ErrCodeAPITimeoutError)?

连不上北极星后台服务器，导致连接超时。

解决方案：

1. 开通网络策略，确保所在机器能够连上 idc 上的北极星后台服务器，环境问题可以咨询 polaris helper。
2. 老版本使用 dns 寻址北极星后台服务器，存在有些机器 dns 没法使用的问题，可以直接升级到最新版本。

### Q6 - filter instances no instances available?

北极星默认开启服务路由，不同环境之间隔离。
新建了特性环境，但是被调服务在所在环境中没有节点导致的报错，确保新环境中被调服务存在节点。

### Q7 - selector: polaris not exist?

trpc-go 框架的名字服务插件都是可插拔的，需要用户自己 import 对应插件注册到框架中才能使用。
确保在 main.go 中添加了以下代码：

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)
```

### Q8 - fail to Register instance, err is Polaris-1001(ErrCodeAPIInvalidArgument)?

```text
register fail:fail to Register instance, err is Polaris-1001(ErrCodeAPIInvalidArgument): fail to validate InstanceRegisterRequest: , cause: 1 error occurred:\n\t* InstanceRegisterRequest: host should not be empty
```

配置有问题，仔细按 [文档](https://git.woa.com/trpc-go/trpc-naming-polaris) 操作。

## 6.2 北极星平台使用相关问题

### Q1 - 北极星如何使用老的寻址系统，如 cl5 ons cmlb 等等？

公司内部绝大多数老寻址系统都已经将数据同步到北极星了，可以直接使用北极星 api 进行寻址，具体看 [这里](https://git.woa.com/trpc-go/trpc-naming-polaris)。

### Q2 - 北极星如何兼容 l5？

有些老的调用方只能支持 l5 来调用，需要被调方服务同时对外提供北极星和 l5。
[北极星管理页面](http://polaris.woa.com/#/polaris/aliases) 已经支持 l5 别名，可以设置别名对外提供 l5 访问方式。

![polaris-admin-ui](../../.resources/user_guide/service_routing/polaris-admin-ui.png)

### Q3 - 如何使用一致性哈希路由？

请参考 [这里](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/master/loadbalance)。

### Q4 - 服务为什么熔断了？

每次发起后端 rpc 请求结束时，trpc-go client 都会上报成功或者失败到北极星的熔断器插件，trpc-go client 只有当请求 `connect 失败和超时` 两种情况才会上报失败，其他全部是成功。如果失败次数达到北极星熔断器阈值（如 1min 连续 10 次失败，具体看 [这里](https://git.woa.com/trpc-go/trpc-naming-polaris)），则开始触发熔断。
当出现熔断时，首先应当 `自己定位为什么失败`，而不是拉北极星或者 trpc 的人来处理。
另外，因为北极星这边只有 connect 失败和超时才算失败，所以在北极星统计平台的错误率和其他监控系统的错误率肯定是不一致的，定位问题时，不要只看监控，还需要结合日志，调用链一起定位。

### Q5 - 如何使用北极星的元数据路由能力？

目前，trpc 插件默认的 selector 只支持规则路由，不支持元数据路由。如果要用这个功能，可以先用 `WithTarget` 来实现。示例如下：

```go
proxy := pb.NewClientProxy(
    client.WithTarget("polaris://xxxx"),
    client.WithCalleeMetadata("key", "val"),
)
```

也可以在配置中进行设置：

```yaml
client:
  service:
    - # 被调服务名
      # 如果使用 pb，callee 必须与 pb 中定义的服务名保持一致
      # callee 和 name 至少填写一个，为空时，使用 name 字段
      callee: "some-callee"
      # 被调服务名，常用于服务发现
      # 注意区分 [naming service 和 proto service](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)
      # name 和 callee 至少填写一个，为空时，使用 callee 字段
      name: "some-name"
      # 选填，指定被调用方元数据，默认为空
      callee_metadata: 
        key: val
      # 选填，目标服务，非空时，selector 将以 target 中的信息为准
      target: "polaris://xxxx"
```

注意：为了避免配置不生效，请仔细阅读 [client 配置的 callee 和 name 的区别是什么](https://iwiki.woa.com/p/99485621#q7-client-配置中的-codeab21e55869c55bd8637c3732df94508c-和-code4c7d8e8ca318a9863d99e9737c57bdfa-的区别是什么？)？

### Q6 - 非 123 平台如何实现北极星名字服务的自注册与反注册？

PCG 123 平台在服务发布时，首先会提前到北极星后台上反注册实例，剔除 ip，然后等一会儿才开始销毁容器，等新容器部署成功以后，再把新 ipport 注册到北极星上。
其他平台没有这个功能，可以使用框架的自注册功能，开启自注册开关，具体见 [这里](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/master/registry)。

注意：

* 服务注册所需要的 token 可以从 [这里](http://polaris.woa.com/) 获取。这个之前是 123 平台在创建新服务时自动调用北极星接口获取的，没有 123 平台的话，那只能自己提前手动到北极星管理后台创建新服务，或者其他发布平台来做这个事。
* instance_id 不用配置，删除即可，自注册会自动生成。
* `plugins.registry.polaris.service.name` 必须与 `server.service.name` 配置一致，否则注册失败。address_list 填北极星 server 远端地址。
* 销毁时，不要使用 `kill -9` 杀进程，可以使用 [这些信号](https://git.woa.com/trpc-go/trpc-go/blob/master/server/serve_unix.go#L19) 停止进程。
* 框架接收到停止进程时，会开始执行反注册，可以自己配置 [`server.close_wait_time`](https://git.woa.com/trpc-go/trpc-go/blob/master/config.go#L104) 决定反注册到真正停止服务之间的等待时间。

### Q7 - 北极星就近访问没有生效？

1. 有可能是下游出错熔断了，只能广州可用。需要排查一下有没有熔断，为什么熔断。
2. 刚启动时，未加载完全地域信息，可以等一会儿再观察一下。

仍然有问题可以联系 noahzeng 定位。

### Q8 - 北极星如何海外部署？

北极星除了部署了国内的服务节点外，还部署了多个海外节点，对框架来说只需要配置下海外节点地址即可：

```yaml
plugins:  # 插件配置
  registry:
    polaris:  # 北极星名字注册服务的配置
      join_point: default  # 名字服务使用的接入点，该选项会覆盖 address_list 和 cluster_service
selector:  # 针对 trpc 框架服务发现的配置
  polaris:  # 北极星服务发现的配置
    join_point: default  # 接入名字服务使用的接入点，该选项会覆盖 address_list 和 cluster_service
```

上面这两个 join_point 配置上对应的海外节点名字即可，默认是国内服务，新加坡独立集群可以配置 singapore，其他集群可联系北极星 helper。

## 6.3 初始化相关问题

### Q1 - setup plugin selector-polaris timeout?

连不上北极星后台服务器，导致连接超时。

解决方案：

1. 开通网络策略，确保所在机器能够连上 idc 上的北极星后台服务器，环境问题可以咨询 polaris helper。
2. 老版本使用 dns 寻址北极星后台服务器，存在有些机器 dns 没法使用的问题，可以直接升级到最新版本。
3. 如果容器核数只有 0.1 核，初始化资源不够也启动不了，需要把容器 cpu 核数调大。
4. 升级 trpc-naming-polaris 到 v0.2.7 版本及以上

### Q2 - 进程启动正常，北极星注册失败？

123 平台部署的服务，是 123 平台自动注册北极星的，trpc_go.yaml 框架配置的 polaris 插件配置也是 123 平台自动填充的，不要自己手动改。
如果是自己配置的插件配置，那么一定要注意正确填写插件配置，务必保证 server.service.name 和 plugins.polaris.service.name 是一致的。

## 6.4 多环境相关问题

### Q1 - 多环境规则不生效？

多环境不生效有很多原因，确保全部满足以下条件：

1. 主调和被调必须两边都注册到北极星。
2. 多环境只在测试环境生效。
3. 只支持 rpc 间调用，不支持 mq 中转。
4. 多环境之间的关系自己确认是否正确，不允许跨环境调用，只能在同个环境相互调用或者继承环境调用基线环境。
5. 地址填写的地方不能使用 `WithTarget` 的方式，必须使用 `WithServiceName` 或者 trpc_go.yaml 配置 `client.service.name`。
6. 发起 rpc 请求的 ctx 必须是请求入口的 ctx，不能是自己创建的 background context，如要启动异步任务，可直接使用框架提供的 api：`trpc.Go(ctx, timeout, handler)`（这里的 ctx 直接传入请求入口的 ctx 即可，内部会自动复制 ctx）。
7. 插件版本升级到最新版，低版本插件有 bug。

### Q2 - 如何启用或关闭多环境功能？

北极星名字服务插件默认启用多环境功能（也就是 service router），可以自己配置关闭，分两种方式：针对所有 rpc 请求的全局关闭，针对特定 rpc 请求的单个关闭，见以上 1.1 小节。

## 6.5 set 相关问题

### Q1 - selector instance empty?

请检查 set 的规则，对应的服务端 set 启用了 set，但根据 set 规则没有相应的节点，或者没有存活的节点。

### Q2 - route set division with set group rule not match, source set name is xxx, not instances found in this set group,please check?

检查下是否使用了 `WithCalleeSetName`，且被调方没有对应的 set。

### Q3 - 分 set 部署，但是发生跨 set 调用问题？

* 注意检查是否使用了 naming-polaris 插件，不要注册自己的 selector 或者其他 selector，请检查是否 import 了其他 selector（CL5 等），或者 `"git.code.oa.com/trpc-go/trpc-naming-polaris/selector"` 等也不要。
* 注意千万不要使用 `client.WithDisableServiceRouter` 选项。
* 注意不要用 `WithTarget` 的方式，使用 `WithServiceName`，注意检查配置文件 client 的 service 下面是不是配置了 target。
* 检查调用的 set 名是否写对。

### Q4 - 我是纯客户端，我想按 set 调用有什么办法吗？

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)

func main() {
    LoadConfig()
}

// 加载 ./trpc_go.yaml 配置，主要是为了让 trpc-naming-polaris 插件启动成功。
func LoadConfig() {
    cfg, err := trpc.LoadConfig("./trpc_go.yaml")
    if err != nil {
        panic("parse config fail: " + err.Error())
    }
    // 保存到全局配置里面，方便其他插件获取配置数据
    trpc.SetGlobalConfig(cfg)

    // 加载插件
    err = trpc.Setup(cfg)
    if err != nil {
        panic("setup plugin fail: " + err.Error())
    }
}
```

在 trpc_go.yaml 中必须包含以下配置：

```yaml
plugins:
  selector:
    polaris:
      # address_list: 9.141.66.8:8081,9.141.66.121:8081,9.141.66.27:8081,9.141.66.125:8081,9.136.124.80:8081,9.136.121.211:8081,9.136.124.240:8081,9.136.125.12:8081,9.136.124.229:8081,9.141.66.84:8081 # 名字服务远程地址列表
      protocol: grpc #北极星交互协议支持 http，grpc，trpc
      discovery:
        refresh_interval: 10000 # 北极星服务发现刷新间隔，123 默认 10000，即 10s
```

### Q5 - 启用了 set，能否在同一个 set 内再启用就近原则？

不能，set 和就近属于互斥，且 set 的第二段本来就为地区信息（area），可以将地区信息纳入到 set 信息中，比如 mtt.sz.1 ,mtt.sz.2, mtt.sh.1, mtt.sh.2。

# 7. 更多文章

1. [tRPC-Go 北极星指北](https://km.woa.com/articles/show/581792)
2. [naming-polaris README](https://git.woa.com/trpc-go/trpc-naming-polaris#clientwithservicename-%E5%AF%BB%E5%9D%80%E4%B8%8E-clientwithtarget-%E5%AF%BB%E5%9D%80%E7%9A%84%E5%8C%BA%E5%88%AB%E4%BB%A5%E5%8F%8A-enable_servicerouter-%E7%9A%84%E8%AF%AD%E4%B9%89)

# 8. 附录

1. [北极星 规则路由使用指南](https://iwiki.woa.com/pages/viewpage.action?pageId=102467866)
2. [tRPC-Go 客户端开发向导](https://iwiki.woa.com/pages/viewpage.action?pageId=284289117)
3. [tRPC-Go 多环境路由](https://iwiki.woa.com/pages/viewpage.action?pageId=99485673)
4. [tRPC-Go Set 路由](https://iwiki.woa.com/pages/viewpage.action?pageId=118669392)
5. [tRPC-Go 金丝雀路由](https://iwiki.woa.com/pages/viewpage.action?pageId=500499679)
6. [KM trpc-go 的寻址 WithTarget, WithServiceName 傻傻分不清楚](https://km.woa.com/group/22063/articles/show/424728)
