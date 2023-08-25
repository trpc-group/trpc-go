## 前言

服务发现是 RPC 框架的重要一环，公司几乎全部统一到了 polaris，而
[naming-polaris](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/v0.5.1) 则是 tRPC-Go 与 polaris 的粘合剂。
这篇文章重点不在于 naming-polaris 如何使用，而是深入到 tRPC-Go 和 Polaris SDK，讲清楚 service router 底层的工作原理，
扫清阅读其他功能文档时的各种困惑，比如：
- target 模式和 service name 模式的区别是什么，什么情况下必须使用其中某一种？
- 什么是 service router，为什么必须开启它？

知其然知其所以然。

## TL;DR

北极星使用问题，几乎全部集中在服务路由部分，这段作一简要整理。
```
+ target
|   当配置了 yaml.client.service[i].target 或者使用了 client.WithTarget 时，进入该模式
|   使用北极星原生 GetOneInstance，路由链如下
|   + dstMetaRouter（目标元数据路由）
|   |   由 client.WithCalleeEnvName 专门指定 env key，或由 client.WithCalleeMetadata 指定任意 kv
|   V
|   + ruleBasedRouter（规则路由）
|   |   + 被调规则
|   |   |   在被调服务端创建的被调规则
|   |   |   存在时，禁用主调规则
|   |   |   当 service router disabled 时，请求匹配规则非空的被调规则会匹配失败，即被跳过
|   |   |     因为 disable service router 会将主调信息置空，只有空的请求匹配规则才能通过
|   |   or
|   |   + 主调规则
|   |       在主调服务端创建的主调规则
|   |       在启动服务时，123 平台会默为它认创建主调规则，用于多环境路由
|   |       会被 disable service router 禁用
|   |         适用于某些情况下，你希望元数据路由的结果不要被规则路由过滤掉
|   V
|   + nearbyBasedRouter（就近路由）
|   V
|   + canaryRouter（金丝雀路由）
|       在 123 平台，只有正式环境可用
or
+ service name
    tRPC-Go 自定义规则
    + if (service router disabled)
    |   + dstMetaRouter
    |   |   只能用于匹配目标环境名，即 client.WithCalleeEnvName，无法像 target 模式一样使用 client.WithCalleeMetadata
    |   V
    |   + setDivisionRouter（set 路由）
    |   V
    |   + nearbyBasedRouter
    |   V
    |   + canaryRouter
    |
    + else if (EnvTransfer is empty)
    |   EnvTransfer 用于 123 多环境路由，当你在 tRPC-Go service 实现中调用下游服务时，这个值一般是非空的
    |   什么情况下 EnvTransfer 是空的？
    |     - 使用了 trpc.BackgroundCtx 或 ctx.Background，如整个 RPC 调用链的起始位置
    |     - 上游调用时，指定了 client.WithDisableServiceRouter，它会关闭 service router，并清空 EnvTransfer，打断其传播
    |   + if (规则路由.主调规则 is empty）
    |   |   对应主调服务没有在 123 注册，如纯 tRPC client，非常少见的场景
    |   |   setDivisionRouter -> nearbyBasedRouter -> canaryRouter
    |   |
    |   + else
    |       通过规则路由.主调规则，生成 EnvTransfer，该字段会随着 tRPC 框架一路透传
    |       + ruleBasedRouter
    |       |   只匹配主调规则，强制忽略被调规则
    |       V
    |       + setDivisionRouter
    |       V
    |       + nearbyBasedRouter
    |       V
    |       + canaryRouter
    |
    + else
        + envTransferRouter
        |   专门和 123 平台多环境路由配合
        |   以 `,` 分隔 EnvTransfer，在得到的 env list 中，从前往后，依次查找，节点非空则立刻返回这些节点
        |   实际实现基于 ruleBasedRouter，但是会忽略 polaris 控制平台上的所有主调与被调规则
        V
        + setDivisionRouter
        V
        + nearbyBasedRouter
        V
        + canaryRouter
```
service router 并没有一个统一的含义。在上一段中搜索 `service router`，以了解它在各种场景下的效果。  
`client.WithDisableServiceRouter`、`yaml.client.serivce[i].disable_servicerouter` 或
`yaml.plugins.selector.polaris.enable_servicerouter` 都可以设置 service router，但只有前两个可以将 EnvTransfer 置空，打断其传播。

## tRPC-Go 是如何调用到 naming-polaris 的

tRPC-Go 提供了插件化的服务发现接口，位于 [`trpc-go/naming`](https://git.woa.com/trpc-go/trpc-go/tree/v0.14.0/naming) 目录下。
其中，`circuitebreaker`、`discovery`、`loadbalance`、`registry`、`servicerouter` 和 `selector` 都提供了 `DefaultXxx` 全局变量。
北极星 SDK 就是通过修改这些全局变量，替换框架默认的寻址方式。  
这些模块中，`selector` 最特殊。它其实是比其他四个（除了 `registry`，它和 `selector` 是平级关系，一个工作在服务端，一个工作在 client 端）
模块高了一个层级，虽然，直接从文件目录看，它们好像是平级关系。
默认的 `DefaultSelector` 是 `TrpcSelector`，它将 `Default-{discovery, servicerouter, loadbalance, circuitbreaker}` 按顺序组
织起来。

tRPC-Go 提供了两种方式决定 `selector`：
- target：通过 `client.WithTarget` 或 `yaml.client.service[i].target` 设置。
- service name：未设置 target 即为 service name 模式。

target 的格式为 `${selector}://${target}`，前者决定了 selector 实例（会替换默认的 `DefaultSelector`），后者决定
`Selector.Select` 方法的第一个参数。  
service name 模式下，selector 实例即 `DefaultSelector`，`Selector.Select` 方法的第一个参数为 `msg.CalleeServiceName()`。
通常情况下，该 name 会[在桩代码中指定](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/testdata/helloworld.trpc.go#L101)，
但也可以通过 `client.WithServiceName` [替换](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/client.go#L129)它。

> 思考题 1：  
> 桩代码中生成的 `client.WithCalleeServiceName` 这一行有什么作用？  
> 提示：它的作用一定发生在「[替换](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/client.go#L129)」之前。

看完 tRPC 框架提供的接口，我们再来看看 naming-polaris 这边。  
naming-polaris 是一个 tRPC [插件](https://iwiki.woa.com/pages/viewpage.action?pageId=500033089)，它对 polaris SDK 进行了封装。
即然是插件，那它的初始化入口就一定是 [`Setup`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/plugin/plugin.go#L16)。 
[继续深入](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/naming.go#L236)，你会发现它将 tRPC-Go 中的
`DefaultXxx` 都改成 naming-polaris 的。
但是 `selector` 是个例外，naming-polaris 并没有修改 tRPC 中的 `DefaultSelector`，而是
[将自己注册到 tRPC 框架中](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/selector/selector.go#L42)。

至此，我们厘清了 target 和 service name 在 naming-polaris 侧的区别是什么了：
- 使用 target `polaris://xxx`，会走到 naming-polaris 的 selector 独立逻辑，它会直接调用北极星的 
  [`GetOneInstance`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/selector/selector.go#L219) 接口。
- 使用 service name，会依次走到 naming-polaris 的
  [`discovery`](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/v0.5.1/discovery)、
  [`servicerouter`](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/v0.5.1/servicerouter) 和
  [`loadbalance`](https://git.woa.com/trpc-go/trpc-naming-polaris/tree/v0.5.1/loadbalance)。
  其中，`servicerouter` 包含了 tRPC 自己定义的大部分特殊规则，如多环境路由。

## `WithTarget` 提供了哪些能力

`WithTarget` 即北极星原生的 `GetOneInstance` 接口。虽说它是原生接口，但 tRPC 还是对请求进行了定制化。我们先看看北极星 SDK 内部的流程，
然后再看 tRPC 定制化的请求会产生什么效果。

我们需要特别关注 naming-polaris 是如何初始化 polaris.consumer.servicerouter 的。在 naming-polaris 中，它是在
[插件初始化时](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/naming.go#L462)设置的（还一处是在
[`selector.New`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/selector/selector.go#L115) 中，但框架并未调用
该接口，部分用户自己初始化 naming-polaris selector 可能会用到，这里我们不讨论它）。可以看到，北极星的路由链为
[`dstMetaRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/dstmeta) ->
[`ruleBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/rulebase) ->
[`nearbyBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/nearbybase) ->
[`canaryRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/canary)。
其中，`dstMetaRouter` 和 `canaryRouter` 是 naming-polaris 添加的，`ruleBasedRouter` 和 `nearbyBasedRouter` 是
[北极星 SDK 默认生成的](https://git.woa.com/polaris/polaris-go/blob/v0.11.2/pkg/config/servicerouter.go#L112)。
这些路由规则，都可以在北极星 [iwiki](https://iwiki.woa.com/pages/viewpage.action?pageId=201083535) 上找到说明。
注意，这里面没有 [`setDivisionRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/setdivision)，
即 target 模式不支持 [set 路由](https://iwiki.woa.com/pages/viewpage.action?pageId=118669392)。

naming-polaris 决定了 `GetOneInstance` 参数的内容。  
其中 `GetOneInstanceRequest.MetaData` 一定是非空的，它会用来进行 
`dstMetaRouter` 的路由匹配，即筛选目标服务在北极星上的实例标签。  
![ 'polaris_instance_labels.png'](/.resources/user_guide/naming_polaris/polaris_instance_labels.png)  
`env` 是该 Metadata 的一个内置 key，value 由 
[`client.WithCalleeEnvName`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L207) 设置。
其他字段来自于 [`client.WithCalleeMetadata`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L207)。
你可以认为，tRPC 的这些 option 就是专门为北极星添加的。  
[`servicerouter` 是否 enabled](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/selector/selector.go#L159)，
决定了 `GetOneInstanceRequest.SourceService` 是否为 nil，即
[是否开启主调规则](https://git.woa.com/polaris/polaris-go/blob/v0.11.2/pkg/flow/data/object.go#L236)
（主调规则是北极星**规则路由**的一个特殊规则，相应的，还有被调规则，后面我们会用一个案例详细地介绍它们）。
注意，这里的 servicerouter 并不是 naming-polaris 下的 servicerouter 模块。
为了避免混淆，当 servicerouter 可以 enable 或 disable 时，我们指的是 bool 变量，其他场景则对应 naming-polaris 的模块。
你需要接受这个奇怪的设定，后面还多次见到。
servicerouter 可以通过
[`client.WithDisableServiceRouter`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L148)、
[`yaml.client.service[i].disable_servicerouter`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/config.go#L35)
或 [`yaml.plugins.selector.polaris.enable_servicerouter`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/naming.go#L35) 
设置，它默认开启。  
开启 servcierouter 后，naming-polaris 会填充源 metadata，它可以由 
[`client.WithCallerEnvName`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L163) 或 
[`client.WithCallerMetadata`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L199) 设置，
会用于规则路由的主调规则中。

## `service name` 模式提供了哪些能力

naming-polaris 的 `discovery` 和 `loadbalance` 是两个简单模块，这里不作介绍。`servicerouter` 模块则实现了 naming-polaris 大部分
特殊的路由功能。
概括地说，`servicerouter` 是 naming-polaris 对 polaris SDK service router 的解构与重组。

我们从入口 `ServiceRouter.Filter` 开始看。其逻辑分支为关键的三个部分 
[`filterWithoutServiceRouter`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L390)、
[`filter`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L395) 和
[`filterWithEnv`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L397)。

先看 `servicerouter` 被 disabled 的场景，即 `filterWithoutServiceRouter`。它依次组装了
[`dstMetaRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/dstmeta) ->
[`setDivisionRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/setdivision) ->
[`nearbyBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/nearbybase) ->
[`canaryRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/canary)。
除了 `nearbyBasedRouter` 外，其他都是按条件组装（从软件工程角度看，这些条件应该全部下沉到对应的 router 内部）。
而 `dstMetaRouter` 更是特殊，它只用来匹配目标环境名，并不是真正意义上的目标元数据匹配。

再看 `EnvTransfer` 为空的场景（`EnvTransfer` 为空的具体含义会在介绍多环境路由时说明），即 `filter`。
此时多了一个主调出规则，根据它是否为空，分为两种组装方式：
- 主调出规则为空：  
  [`setDivisionRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/setdivision) ->
  [`nearbyBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/nearbybase) ->
  [`canaryRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/canary)。
- 主调出规则非空：  
  [`ruleBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/rulebase) ->
  [`setDivisionRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/setdivision) ->
  [`nearbyBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/nearbybase) ->
  [`canaryRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/canary)。  
  为了支持主调出规则，这里加上了规则路由 `ruleBasedRouter`。但是，这个规则路由并没有设置 dst metadata，即用户为被调设置的被调规则并不会生效。
  这里主调 metadata 有一点需要特别关注，`key` 和 `env` 变成了二选一，它们分别由 
  [`client.WithEnyKey`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L156) 和
  [`client.WithCallerEnvName`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L163) 设置。
  显然，tRPC 的这两个 option 又是为其也平台专门定制的。

最后是 `EnvTransfer` 非空的场景，即 `filterWithEnv`。
这里的路由链是这么组装的：  
[`ruleBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/rulebase) ->
[`setDivisionRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/setdivision) ->
[`nearbyBasedRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/nearbybase) ->
[`canaryRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/canary)。  
与前一段的区别在于，这里的主调规则是 naming-polaris 自己构建的，而非是从北极星拉取，而且，源 metadata 只设置源 env name，
`client.WithCallerMetaData` 对它是没有用的。

总得来看，servicerouter 有非常多的特殊逻辑，单从代码，很难理解为什么它要这么写。
实际上，disable servicerouter 外的两个场景都是专门为多环境路由服务的。
需要结合 123 平台，才能进一步理解这些代码。

## [123](https://iwiki.woa.com/space/123) 与多环境路由

如果关闭 naming-polaris 中的配置 
[`yaml.plugins.registry.polaris.register_self`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/registry/registry_factory.go#L26)，
[框架就不会自动往北极星注册/反注册服务](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/registry/registry.go#L89)
（但是仍旧会进行心跳上报）。
在 123 平台上，这个正是默认关闭的。123 在注册服务时，同时也会为该服务注册出规则路由。比如，下图中，`trpc.cooper.server.Hello` 这个服务有
在三个环境部署，一个基线测试环境 `test`，两个特性环境 `127664a0` 和 `786180c4`，北极星注册了三个出路由规则（即主调规则）：  
![caller_router_rules](/.resources/user_guide/naming_polaris/caller_router_rules.png)  
首先，这三个规则，从上往下，[优先级依次降低](https://git.woa.com/polaris/polaris-go/blob/v0.11.2/plugin/servicerouter/rulebase/base.go#L596)。
以第一条规则为例，它表示，从 `786180c4` 发起的请求，会优先匹配 `786180c4` 环境中的节点，如果 `786180c4` 为空，则匹配 `test` 环境的节点。
这其实就是特性环境不存在时，回基线环境的逻辑。  
假如请求来自于 `127664a0`，第一条规则的请求匹配规则就会失败，跳过它的实例匹配规则，进入第二条规则。当然，这条规则就可以匹配成功了。  
假如前两个规则的请求匹配规则都会失败，进入第三条规则。
与前两条不同，这个主调规则的请求匹配规则的 `valueType` 为 `PARAMETER`。
`PARAMETER` 表示，`env` 的值，来自于请求参数中 `callerMetadata["env"]`。
显然，这里的请求匹配规则必然可以成功，实际上，删除这条请求匹配规则也不会有任何问题。
然后是实例匹配规则，它也是 `PARAMETER` 类型。这相当于在筛选 `env` 为 `callerMetadata["env"]` 的目标实例，即选择与主调处于同一环境的节点。
如果主调环境里没有被调实例，则在基线 `test` 寻找，即所有非基线环境兜底回基线。


> 思考 2：  
> 你可能注意到了，第三个规则已经包含了前面两个规则。那为什么不删除前面两个规则，只保留最后一个？  
> 请看完本节再回答。
 
> 思考 3：  
> 如何对 123 注册的服务实现目标的元数据匹配规则？

上面讲到的主调路由规则会对所有开启 `ruleBasedRouter` 生效，即对 target 模式和 service name + enable service router + empty
EnvTransfer 生效。然而，这远不是多环境路由的全貌。在多环境路由文档中，下面的功能是如何实现的呢？  
![multiple_envs_router](/.resources/user_guide/naming_polaris/multiple_envs_router.png)  
如果按我们之前讲的主调规则，SVR2 是如何重新回到特性 SVR3 的？  
假设用户不使用 target 模式，在整个调用链路中都使用 service name 模式。当 SVR1 调用 SVR2 时，它会进入
[`servicerouter.filter`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L280) 逻辑，
在前一节，我们忽略了一段关键逻辑 [`newEnvStr`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L293)，
它会找到主调规则中所有包含本节点所在 env 的请求匹配规则，并把它们实例匹配规则中所有可能通往的 env 拼接起来。
比如，设 SVR1 在环境 `127664a0` 内，SVR2 在 `test` 内，拼接出的 `newEnvStr` 为 `127664a0,test`。
这个值会用在 [tRPC-Go 主框架](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/client.go#L444)中，并
[透传给下游服务](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/codec.go#L604)。
另外，还可以看到，如果 disable 了 service router，
[多环境路由关键的 env 链就会会打碎](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/client.go#L449)，
重新回到 123 注册的主调规则。  
接下来，我们看基线 `test` 的 SVR2 是如何调用到特性 `127664a0` 的 SVR3。
因为[上游给 SVR2 带来了 env transfer](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/client.go#L115)，
所以，SVR2 naming-polaris 走到了 
[`servicerouter.filterWithEnv`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L397)
逻辑。其中的 [`buildRouteRules`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L401)
实际就是在根据 env transfer，自己建了一个主调规则，按 env list 的顺序，
[优先级逐步降低](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L433)。
所以 `127664a0,test` 会优先匹配 127664a0，其次才回到基线 test。所以，SVR2 正确地调到了 SVR3。  
SVR3 到 SVR4 也是类似的原理。

这时，再回头看 [tRPC-Go 多环境路由](https://iwiki.woa.com/pages/viewpage.action?pageId=99485673)，应该不会感到迷茫。

## set 路由补遗

本节是对 [tRPC-Go set 路由](https://iwiki.woa.com/pages/viewpage.action?pageId=118669392)的补充。

在北极星控制台上，开启 set 路由的服务会多两个实例标签，即 `internal-enable-set` 和 `internal-set-name`。123 需要在「服务详情/资源配置」
中添加新的 set 组。扩容时，可以指定 set 组。123 在往北极星注册服务时，自动打上前面的两个标签，并且修改 `trpc_go.yaml` 模板，在
`yaml.global` 中添加 [`enable_set`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/config.go#L56) 和
[`full_set_name`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/config.go#L58) 两个字段。在创建 tRPC ctx 时，会
[自动带上 set 信息](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/trpc_util.go#L38)。

[`setDivisionRouter`](https://git.woa.com/polaris/polaris-go/tree/v0.11.2/plugin/servicerouter/setdivision) 分两种工作模式，
一种是自动判断是否开启 set，另一种是强制路由到目标 set（可以通过
[`client.WithCalleeSetName`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/options.go#L177) 或
[`yaml.client.service[i].set_name`](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/config.go#L31) 来指定）。
前者遵循支持通配组，即 set 最后一段为 `*`；后者不支持通配组，并将 `*` 作为普通字段，进行完全匹配。
无论哪种场景，只要 set 路由生效了，就近路由就会关闭。

## 思考解答

**思考 1**

它用于 client yaml 配置加载。当你通过 `pb.NewXxxClientProxy()` 创建一个新 client 实例，有没有想过它是怎么对应到
`yaml.client.service[i]` 的。关键就在于桩代码中的 `calee service name` 会和 `yaml.client.service[i].callee` 进行
[匹配](https://git.woa.com/trpc-go/trpc-go/blob/v0.14.0/client/client.go#L91)。

**思考 2**

因为多环境路由需要这些信息去构建
[`env transfer`](https://git.woa.com/trpc-go/trpc-naming-polaris/blob/v0.5.1/servicerouter/servicerouter.go#L293)。

**思考 3**

要使用目标元数据匹配规则，只能使用 target 模式。
需要注意，`dstMetaRouter` 后，还会再走 123 给我们主调服务添加的 `ruleBasedRouter` 的主调规则。如果我们想从特性环境 A 调到另一个特性环境 B，123
的规则就会阻止我们。为了绕过它，我们可以给被调服务添加一个总是成功的被调规则：
![special_caller_rule_for_dst_meta_router](/.resources/user_guide/naming_polaris/special_caller_rule_for_dst_meta_router.png)  
其中，`metadata_key` 是我们要匹配的目标元数据。
在存在被调规则的情况下，`ruleBasedRouter` 会跳过主调规则。这并不会影响 123 servcie name 的能力，因为 servicerouter 调用的
[`GetFilterInstances`]() 并不会去获取被调规则。  
注意，你需要用 `client.WithCallerMetadata` 来传递需要匹配的 callee 元数据，这是 `PARAMETER` 工作的原理。
