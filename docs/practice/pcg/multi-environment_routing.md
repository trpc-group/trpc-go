推荐先阅读 [tRPC-Go 服务路由（tRPC 知识库）](https://iwiki.woa.com/pages/viewpage.action?pageId=4008319150)

# 1 前言

多环境路由是在北极星规则路由的基础上，通过规则来控制流量路由到不同测试环境的服务节点上，要使用多环境功能，`主调和被调服务都必须注册到北极星上面`，**注意！！！主调服务也必须注册到北极星上面！！！**

在使用多环境路由之前请先仔细阅读 [北极星规则路由文档](https://iwiki.woa.com/pages/viewpage.action?pageId=102467866)。

**注：** 本文中的“上游”指的是主调，“下游”指的是被调。

------------------------------------------------------------------------------------------------------------

# 2 多环境的原理和实现

## 2.1 服务路由的流程

服务路由模块从所有可用的服务实例中，根据用户配置的规则和策略，筛选本次路由的服务实例子集，实现对服务出流量和入流量的控制。具体流程如下图。

![routing-overview](../../../.resources/practice/pcg/multi-environment_routing/routing-overview.png)

- 服务发现：根据服务名从缓存中查询所有可用的服务实例列表。
- 规则路由：根据服务名和匹配参数从缓存中查询用户配置的路由规则，根据路由规则筛选符合条件的服务实例子集。
- 就近路由：从规则路由输出的服务实例子集中，筛选和主调方位置相近的服务实例，作为本次路由的服务实例子集。
- 负载均衡：从本次路由的服务实例子集中，根据具体的负载均衡策略，选出本次调用的服务实例。

## 2.2 路由规则的使用

目前只支持 `出规则`，框架默认采用 "env" 来区分不同的环境，不同的环境的 env 值不同。下面代码就是匹配 env:test1 的路由规则代码。

```go
opts := []client.Option{
    // 被调的 namespace
    client.WithNamespace("Development"),
    // 被调的 service
    client.WithTarget("polaris://trpc.app.server.service"),
    // 主调的 namespace，用于主调服务出规则路由查找
    client.WithCallerNamespace("namespace"),
    // 主调的 service，用于主调服务出规则路由查找
    client.WithCallerServiceName("service"),
    // 设置主调服务环境名，用于匹配路由规则
    client.WithCallerEnvName("test1"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error("call by polaris discovery err: %s", err.Error())
    return 
}
```

如果使用框架的 ctx，则会默认把 trpc_go.yaml 配置文件里面的主调 service，主调服务 namespace 和主调环境名传入，用户不再需要显示通过 option 去指定这些参数，则上述代码可以简化为：

```go
opts := []client.Option{
    // 被调的 namespace
    client.WithNamespace("Development"),
    // 被调的 service
    client.WithTarget("polaris://trpc.app.server.service"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error("call by polaris discovery err: %s", err.Error())
    return 
}
```

### 2.2.1 上游环境透传

trpc 框架支持自定义实现 `服务发现、负载均衡、服务路由、熔断` 等组件。使用 `client.WithServiceName` 指定寻址，则会组合使用北极星的服务发现组件、负载均衡组件、服务路由组件、熔断器组件来进行寻址，使用 `client.WithTarget` 寻址，则会整个使用北极星的 `GetOneInstance` 接口，不会关心内部的各个组件的配合。

在 trpc_go.yaml 配置下游请求 service 的 name 或者 callee 字段，或者直接在代码中使用 `client.WithServiceName` 指定，都属于 `client.WithServiceName` 指定寻址。

在 trpc_go.yaml 配置下游请求 service 的 target，或者直接在代码中使用 `client.WithTarget` 指定，都属于 `client.WithTarget` 指定寻址。

如果 service 的 name, callee, target 都配了的话，target 的优先级最高。

trpc 框架默认通过 ctx 把上游服务的环境透传到下游，例如：上游节点所在环境有 test0 和 test1 则 "test0,test1" 会被透传到下游。使用 `client.WithServiceName` 寻址则会使用透传的环境信息。使用 `client.WithTarget` 则忽略环境信息。

透传环境的 `优先级大于` 用户在北极星配置路由规则，默认会使用上游透传的环境去找寻相应环境的节点，找不到将会直接报错，错误信息将包含 `filter instance with env err` 关键字，可以使用下面的代码关闭环境透传功能。

```go
// 框架的 ctx
// 向下游发起请求前执行下面的代码：
msg := trpc.Message(ctx)
msg.WithEnvTransfer("")
```

### 2.2.2 自定义路由匹配规则

```go
// 默认使用框架 ctx，如果没有则需要手动加上下面的参数，或者使用 trpc.BackgroundContext() 

/*
opts := []client.Option{
    // 主调的 namespace，用于主调服务出规则路由查找
    client.WithCallerNamespace("namespace"),
    // 主调的 service，用于主调服务出规则路由查找
    client.WithCallerServiceName("service"),
}
*/

opts := []client.Option{
    // 被调的 namespace
    client.WithNamespace("Development"),
    // 被调的 service
    client.WithTarget("polaris://trpc.app.server.service"),
    // 主调服务的路由匹配自定义元数据
    client.WithCallerMetadata("key1", "val1"),
    client.WithCallerMetadata("key2", "val2"),
    // 使用框架 ctx 默认传入 env: test1 作为匹配规则，可以加上下面这一行清空 env
    // client.WithCallerMetadata("env", ""),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error("call by polaris discovery err: %s", err.Error())
    return 
}
```

### 2.2.3  指定环境（节点）请求

有两种方式指定下游的节点进行访问：

#### 2.2.3.1 通过自定义规则路由匹配

例如：

a. 设置请求匹配规则，用户通过在调用的时候指定 env 为 04024680 来匹配这条规则。

```json
标签：{ "env": { "value": "04024680", "type": "EXACT" } }
```

b. 设置上述规则对应的目标匹配规则，下述规则将会匹配所有节点的 metadata 里面包含 `env:04024680` 的节点：

```json
标签： { "env": { "value": "04024680", "type": "EXACT" } }
优先级： 0
权重： 100
```

可以设置多个规则（任何用户自定的规则）来对应不同的路由逻辑，详细请参考 [北极星规则路由文档](https://iwiki.woa.com/pages/viewpage.action?pageId=102467866)。

#### 2.2.3.2 指定环境请求（默认使用 env 来区分）

下述代码默认把流量导入到 metadata 包含 `env: 62a30eec` 的节点：

```go
opts := []client.Option{
    client.WithNamespace("Development"),
    // client.WithTarget("polaris://trpc.app.server.service"),
    client.WithServiceName("trpc.app.server.service"),
    // 设置被调服务环境
    client.WithCalleeEnvName("62a30eec"),
    // 关闭服务路由
    client.WithDisableServiceRouter()
}
```

### 2.2.4 123 平台使用

123 平台相关概念：

- 命名空间（namespace）：测试环境都为 `Development`，现网都为 `Production`。
- 服务名：（service name）：服务名，通过服务名和命名空间可以唯一确定一个服务。
- 环境（env）：123 运营平台通过 env 来区分不同的环境，不同的测试环境 env 的值不同，正式环境只有 `formal` 一个环境。
- 基线环境：稳定的测试环境。
- 特性环境：基于基线环境继承的环境。

123 平台会针对测试环境生成不同的服务规则，以保证下面几点：

- 不同基线环境的服务不能相互调用，也就是不同的 env 不能够互相调通。
- 测试环境的不能调通现网环境，也就是命名空间 Development 不能调通 Production。
- 默认调用本环境服务，本环境没有节点并且存在对应的基线环境则调用基线环境。
- 利用 trpc 框架环境透传功能，实现基线调用特性环境。

下面分别介绍常用的几种使用：

#### 2.2.4.1 基线环境和特性环境隔离

![baseline_and_feature_env_1](../../../.resources/practice/pcg/multi-environment_routing/baseline_and_feature_env_1.png)

确保使用框架的 ctx 或者 `trpc.BackgroundContext()` 的情况下，代码可以简化为：

```go
opts := []client.Option{
    // 被调的 namespace
    client.WithNamespace("Development"),
    // 被调的 service
    client.WithTarget("polaris://trpc.app.server.service"),
    // 使用 client.WithServiceName 的时候，如果上游服务处在不同的环境，
    // 环境信息会被透传到下游，导致服务规则失效。
    // client.WithServiceName("trpc.app.server.service"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error("call by polaris discovery err: %s", err.Error())
    return 
}
```

#### 2.2.4.2 特性环境服务不存在则调用基线服务

![baseline_and_feature_env_2.](../../../.resources/practice/pcg/multi-environment_routing/baseline_and_feature_env_2.png)

确保使用框架的 ctx 或者 `trpc.BackgroundContext()` 的情况下，代码可以简化为：

```go
opts := []client.Option{
    // 被调的 namespace
    client.WithNamespace("Development"),
    // 被调的 service
    client.WithTarget("polaris://trpc.app.server.service"),
    // 使用 client.WithServiceName 的时候，如果上游服务处在不同的环境，
    // 环境信息会被透传到下游，导致服务规则失效，如果上游存在同样的环
    // 境则不会有什么影响，后续服务之间可以调通。
    // client.WithServiceName("trpc.app.server.service"),
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error("call by polaris discovery err: %s", err.Error())
    return 
}
```

#### 2.2.4.3 环境优先级信息透传

![environmental_priority](../../../.resources/practice/pcg/multi-environment_routing/environmental_priority.png)

确保使用框架的 ctx 或者 `trpc.BackgroundContext()` 的情况下，代码可以简化为：

```go
opts := []client.Option{
    // 被调的 namespace
    client.WithNamespace("Development"),
    // 被调的 service
    // 必须使用使用 client.WithServiceName，如果上游服务的环境信息会被
    // 透传到下游，才能够实现基线环境调用特性环境。
    client.WithServiceName("trpc.app.server.service"),
    // 不能使用 client.WithTarget，否则不会生效。
    // XXXX client.WithTarget("polaris://trpc.app.server.service"), 
}

clientProxy := pb.NewGreeterClientProxy(opts...)
req := &pb.HelloRequest{
    Msg: "hello",
}
rsp, err := clientProxy.SayHello(ctx, req)
if err != nil {
    log.Error("call by polaris discovery err: %s", err.Error())
    return 
}
```

# 4 FAQ

请参考 [北极星插件问题](https://iwiki.woa.com/p/4008319150#6faq)。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
