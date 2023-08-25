[TOC]

# tRPC-Go 过载保护

## 前言
RPC 框架应该为服务提供稳定性保障。这里的稳定性指，当大量请求涌入时，应该：
- 保证成功请求的链路耗时稳定在较低的值，不会剧烈波动，也不会膨胀；
- 对于超出处理能力的请求，及早拒绝，防止链路超时；
- 避免因协程数过多或队列过长造成部分服务 OOM。  

解决链路稳定性问题有两种不同的思路。一种是基于配额的限流策略，另一种是服务端自适应过载保护。  
限流通过限制请求的 QPS，使整个服务链路保持在一个较低的负载水平下。它有单机和分布式两种，是常见的策略。  
服务端自适应过载保护通过监控服务本身的运行状态，拒绝过多的请求，来使服务稳定在一个最佳的状态。  

通常，服务端自适应过载保护足以为你的服务提供稳定性保障。如果你需要限制某类请求的流量，也可以将两者搭配起来使用。

## 服务端自适应过载保护
> 设计细节请参考 tRPC 提案：[A10_overload_control](https://git.woa.com/trpc/trpc-proposal/blob/master/A10-overload-control.md)。
trpc-go 从 [v0.7.0](https://git.woa.com/trpc-go/trpc-go/blob/v0.7.0/CHANGELOG.md) 开始支持服务端自适应过载保护。
请与算法库 [trpc-go/trpc-overload-control](https://git.woa.com/trpc-go/trpc-overload-control) 配合使用。
请使用最新版 [v1.4.2](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.4.2/CHANGELOG.md)。

tRPC-Go 提供了基于三种指标的过载保护策略。一种是协程最大调度耗时，一种是协程睡眠漂移，还有一种是请求最大耗时。请求耗时默认不开启。
当框架监控到指标大于最大期望值时，会基于优先级，对部分请求进行限流，立刻返回过载错误，保证它稳定在最大期望值附近，从而保证服务稳定。

请先在测试环境开启「日志监控」和「dry-run 模式」，确保默认协程调度耗时表现良好。
如果你的上游是 tRPC-CPP 服务，请确保它的版本大于 [v0.10.0](https://git.woa.com/trpc-cpp/trpc-cpp/blob/v0.10.0/CHANGELOG.md)。因为在此之前，CPP 的北极星熔断会统计过载错误，不符合过载保护的预期。
使用自适应过载保护，服务 QPS 需要至少在 300 以上，QPS 很低建议直接使用基于 Quota 的限流，如 [rate limiter](https://pkg.go.dev/golang.org/x/time/rate) 或者本文档中的北极星限流。

要开启自适应过载保护，只需要匿名导入过载保护包，该包会注册名为 overload_control 的默认过载保护拦截器。

```java
import _ "git.code.oa.com/trpc-go/trpc-overload-control/filter"
```
再将 `overload_control` 拦截器添加到 `trpc_go.yaml` 中：

```yaml
server:
  filter: [overload_control]
  service:
    - name: xxx
```

### 优先级
过载保护支持优先级功能，在服务过载时，优先让高优请求通过，拒绝低优请求。
请求的优先级默认为 0（最低），过载时随机拒绝。下面的方法可以给请求打上优先级标记：

```aidl
import overloadctrl "git.code.oa.com/trpc-go/trpc-overload-control"
    // 设置请求的优先级为 128，允许的最大值为 255（选择正确的方法来设置）
    ctx = overloadctrl.SetServerRequestPriority(ctx, 128) // 非下面情形时，必须使用该方法
    ctx = overloadctrl.SetRequestPriority(ctx, 128) // 在 client 拦截器中设置优先级时，必须使用该方法
```
它会在请求的 `meta data` 中设置优先级，随着请求链，一路透传下去。

### plugin 配置
通过 plugin，可以调整过载保护的参数：

```yaml
plugins:
  overload_control: # 插件类型，必须是 overload_control
    # 插件名，同时也是插件注册的拦截器名
    # 如果使用 overload_control，则会覆盖注册的默认拦截器
    # 如果使用其他插件名，则必须在代码中手动调用一次 plugin.Register 方法进行注册
    overload_control:
      # 所有配置项都可以留空，这时会使用默认值
      server: # 服务端过载保护
        # 当同时配置了白名单和黑名单时，只有白名单会生效
        whitelist: # 白名单，该拦截器只对白名单中的 service/method 生效
          service_a: # 服务名，msg.CalleeServiceName()，其下配的两个 method_1/2 在白名单中，过载保护对其他方法不生效
            method_1: # 方法名，msg.CalleeMethod()，method_1 在白名单中
            method_2: # method_2 也在白名单中
          service_b: # service_b 下的所有 method 都在白名单中
          # 其他 service 都不在白名单中，该算法对它们都不生效
        blacklist: # 黑名单，只有未配置白名单时才生效，算法会忽略黑名单中的 service/method
          service_x: # service_x 下配的两个 method_1/2 在黑名单中，其他 method 则不在
            method_1: # 方法名，msg.CalleeMethod()，method_1 在黑名单中
            method_2: # method_2 也在黑名单中
          service_y: # service_y 下的所有 method 都在黑名单中
        dry_run: false # 是否开启 dry run 模式，默认关闭，开启后，过载保护总是放行请求，可以在不影响业务的情况下通过日志来观察算法的状态
        goroutine_schedule_delay: 3ms # 期望的最大协程调度耗时，默认 3ms，如果你需要调整这个值，请先阅读过载保护提案
        sleep_drift: 0ms # 期望的最大协程睡眠漂移，0，默认不开启，如果你的服务没有使用原生 trpc 协议，或者只使用了 udp（协程调度耗时无法生效时）请将该值配为 3ms，如果你需要调整这个值，请先阅读过载保护提案
        request_latency: 0ms # 期望的最大请求耗时，0，默认不开启，如果你需要调整这个值，请先阅读过载保护提案
        cpu_threshold: 0.75 # 过载保护生效时的最低 CPU 使用率（整个容器的），默认 75%
        cpu_interval: 1s # 计算过去 1s 内的 CPU 使用率，这个值越大，过载保护在开启和关闭间切换得越慢，默认 1s
        log_interval: 0ms # 过载保护状态日志的最小时间间隔，用于 debug，0 为不开启日志
```
如果自定义插件名不为 `overload_control`，需要手动注册插件：

```aidl
import "git.code.oa.com/trpc-go/trpc-go/plugin"
import "git.code.oa.com/trpc-go/trpc-overload-control/filter"
plugin.Register(name, filter.NewPlugin(/* options */))
```

`NewPlugin` 方法可以接收参数，允许修改 plugin 创建的默认过载保护策略。而 yaml 配置，则可以在默认策略的基础上，进行调整。

plugin 会注册一个与插件名相同的拦截器，使用时，将插件名填入 filter 中即可。  
注意，请将过载保护拦截器配在监控拦截器之后，这样被调监控就要以上报过载错误了。  
当需要全局开启一个过载保护策略，而对某个 service 自定义策略时，可以将该 service 加入全局策略的黑名单中，再在 service 的 filter 中单独加一个拦截器。

### 通过代码添加过载保护拦截器
plugin 只提供了部分能力，通过代码，可以创建更加精细的过载保护策略。具体请参考 [RegisterServer](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/filter/filter.go#L39) 方法和 
[server](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/overloadctrl.go#L28) 过载保护库的
各种 [Opt](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/options.go#L45)。

## 客户端自适应过载保护
>设计细节见 [优先级过载保护](https://git.woa.com/cooperyan/trpc-proposal/blob/A12_client_oc/A12-priority.md#%E5%9C%A8%E5%AE%A2%E6%88%B7%E7%AB%AF%E6%8F%90%E5%89%8D%E9%99%90%E6%B5%81) 提案。
tRPC-Go 的版本必须 >= [v0.8.2](https://git.woa.com/trpc-go/trpc-go/blob/v0.8.2/CHANGELOG.md)。

很多时候，我们希望在整个请求链路的入口处就感知到后端服务已经过载，提前进行降级处理，避免浪费额外资源。

与第二章服务端优先级过载保护类似，tRPC-Go 中也支持客户端优先级过载保护。该算法以后端返回的错误码
- `22`：下游服务过载
- `124`：下游开启了客户端过载保护，感知到下游的下游发生过载，因此提前返回了过载错误
- `23`：北极星限流的 server 端错误
- `124`：北极星限流的 client 端错误

作为决策依据（要添加新的错误码请参考 yaml 配置中的 throttle_err_codes），通过在客户端提前拒绝一部分请求，保证服务端适当过载。
这些被拒绝的请求会返回 124 错误码，因为它们从未离开过客户端，因此可以选一个新节点，放心地进行重试。

算法会优先拒绝低优请求，放行高优请求。

### 开启客户端过载保护
要开启客户端自适应过载保护，需要先在匿名导入客户端过载保护包：
```java
import _ "git.code.oa.com/trpc-go/trpc-overload-control/filter"
```

再在 `trpc_go.yaml` 为 client 添加过载保护拦载器：

```yaml
client:
  # 注意，selector 拦截器判断配置在过载保护之前！
  # `selector` 为 tRPC-Go 框架默认的服务发现拦截器名，如果未在 filter 中显式指定，它将自动加入到所有拦截器之后。
  filter: [selector, overload_control]
  service:
    - name: xxx
```

### Plugin 配置
如非必要，不要修改默认配置。
可以在 plugin 的 yaml 配置中调整客户端过载保护的参数：
```yaml
plugins:
  overload_control: # 插件类型，必须是 overload_control
    # 插件名，同时也是插件注册的拦截器名
    # 如果使用 overload_control，则会覆盖注册的默认拦截器
    # 如果使用其他插件名，则必须在代码中手动调用一次 plugin.Register 方法进行注册
    overload_control:
      # 所有配置项都可以留空，这时会使用默认值
      client: # 客户端过载保护
        # 当同时配置了白名单和黑名单时，只有白名单会生效
        whitelist: # 白名单，该拦截器只对白名单中的 service/method 生效
          service_a: # 服务名，msg.CalleeServiceName()，其下配的两个 method_1/2 在白名单中，过载保护对其他方法不生效
            method_1: # 方法名，msg.CalleeMethod()，method_1 在白名单中
            method_2: # method_2 也在白名单中
          service_b: # service_b 下的所有 method 都在白名单中
          # 其他 service 都不在白名单中，该算法对它们都不生效
        blacklist: # 黑名单，只有未配置白名单时才生效，算法会忽略黑名单中的 service/method
          service_x: # service_x 下配的两个 method_1/2 在黑名单中，其他 method 则不在
            method_1: # 方法名，msg.CalleeMethod()，method_1 在黑名单中
            method_2: # method_2 也在黑名单中
          service_y: # service_y 下的所有 method 都在黑名单中
        throttle_err_codes: [] # 额外增加新的后端返回的错误码作为限流决策依据。比如，101, 141 等
        max_throttle_probability: 0.7 # 最大限流比例，默认最多拒绝 70% 的请求，即至少有 30% 的请求会到达后端服务
        ratio_for_accept: 1.3 # 保证发往下游的请求中，过载的比例不会大于约 0.3 =  1.3 - 1，默认 1.3，具体见优先级过载保护提案
        ema_factor: 0.8 # 在统计请求数时，指数移动平均值的因子，默认 0.8，普通用户不要设置，具体见代码
        ema_interval: 100ms # 在进行请求数的指数移动平均值统计时，使用的时间范围，默认 100ms，普通用户不要设置，具体见代码
        ema_max_idle: 30s # 在进行请求数的指数移动平均值统计时，如一段时间内没有请求，则会将数量重置为 0，默认 30s，普通用户不要设置，具体见代码
        log_interval: 0ms # 过载保护状态日志的最小时间间隔，用于 debug，0 为不开启日志
```
与 server 端过载保护类似，当自定义插件名不为 `overload_control` 时，用户也需要手动注册插件，这里不再赘述。

### 通过代码添加客户端过载保护拦截器
Plugin 只提供了部分能力，通过代码，可以创建更加精细的过载保护策略。具体请参考 [RegisterClient](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.3.2/filter/filter.go#L32) 方法
和 [client](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.3.2/client/overloadctrl.go#L26) 过载保护库的
各种 [Opt](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.3.2/client/options.go#L1)。

## 限流
tRPC 目前提供了基于北极星的限流策略，请参考这个[提案](https://git.woa.com/trpc/trpc-proposal/blob/master/A9-polaris-limiter.md#%E5%8C%97%E6%9E%81%E6%98%9F%E9%99%90%E6%B5%81)。

### 北极星
tRPC-Go 的北极星限流是通过 [trpc-filter/polaris/limiter](https://git.woa.com/trpc-go/trpc-filter/tree/master/limiter/polaris) 插件实现的。
它是对[北极星 SDK](https://git.woa.com/polaris/polaris-go) 的封装，让 tRPC 用户方便地接入北极星限流。当请求被限流时，服务端会返回框架错误码 `23`，客户端会返回框架错误码 `123`。
详细的北极星限流能力请参考[访问限流使用指南](todo)。下面简单介绍插件及限流策略的配置。

#### tRPC-Go 服务配置
在代码中匿名引用插件：
```aidl
import _ "git.code.oa.com/trpc-go/trpc-filter/limiter/polaris"
```
配置 `trpc_go.yaml`：
```yaml
client:
  filter: [polaris_limiter]  # 开启 client 端限流
  service:
    # ...
server:
  filter: [polaris_limiter]  # 开启 server 端限流
  service:
    # ...
plugins:
  limiter:
    polaris: # 不可省略
      timeout: 1s # 可省略，省略时，使用默认值 1s
      max_retries: 2 # 可省略，省略时，默认不重试
      # metrics_provider 决定是否开启插件的指标上报。默认为空，不开启。注意，北极星控制台已经提供了监控功能。
      # 目前只支持了 m007，其他选项会导致插件初始化错误。
      # 该指标的链接位于 123 平台的「服务监控」「trpc 自定义监控」「xxx_limiter_polaris_request」。
      metrics_provider: m007
```
请根据需要开启 client/server 限流。注意，示例中为整个 client/server 配置了 `polaris_limiter` 拦截器，你也可以单独为部分 service 配置拦截器。

#### 在北极星控制台配置限流策略
> 本节的北极星截图不保证与最新版北极星控制台一致。

这是[北极星控制台](http://v2.polaris.woa.com/#/services/list?owner=chrisbchen&isAccurate=1&hostOperator=1&page=1)。找到你的服务后，新建限流策略：
![polaris_console](/.resources/user_guide/overload_control/polarisconsole.png)
北极星支持分布式和单机限流两种模式，我们以分布式限流为例子，单机限流策略配置类似。
![polaris_config_limiter](/.resources/user_guide/overload_control/polarisconfiglimiter.png)
大部分配置字段都有清晰的含义，这里，我们只关注如何填写维度。

tRPC-Go 北极星限流插件会上报两个维度：`method` 和 `caller`，即被调方法名和主调服务名。

**场景 1**

为方法 M1 配置一个 100/s 的限流值，可以这样填写：
![polaris_config_limiter_m1](/.resources/user_guide/overload_control/polarisconfiglimiterm1.png)
创建出下面的限流策略：
![polaris_config_limiter_m1_policies](/.resources/user_guide/overload_control/polarisconfiglimiterm1policies.png)

**场景 2**

限制方法 M1 的请求数为 100/s，限制来自上游 `trpc.app.server.service_A`，调用方法 M2 的请求数为 50/s，需要配置两个限流策略。第一个策略与场景1一样，第二个策略如下配置维度：
![polaris_config_limiter_m1_scenario2](/.resources/user_guide/overload_control/polarisconfiglimiterm1scenario2.png)
最终创建出下面的两个限流策略：
![polaris_config_limiter_m1_policies_scenario2](/.resources/user_guide/overload_control/polarisconfiglimiterm1policiesscenario2.png)

**场景 3：自定义维度**

默认限流插件只提供了 `method` 和 `caller` 两个维度。如果你要基于自定义维度进行限流，必须自行注册一个新的 filter。
比如，你想对来自北京的请求进行限流，即需要两个维度：`method` 和 `city`。
自定义新的 limiter：
```aidl
    l, err := limiter.New(
        limiter.WithSDKCtx(polarisSDKCtx), // 多个 limiter 可以复用同一个北极星 SDK context。省略时，New 方法自动初始化一个新的北极星 SDK context。
        limiter.WithNamespace(namespaceForTRPCServer), // 必填，因为是服务端限流，所以使用 namespaceForTRPCServer。
        limiter.WithService(serviceForTRPCCallee), // 必填
        limiter.AddLabels(
            labelForTRPCCallerService, // 尽可能地将所有可能用到的维度合并进同一个 limiter 中，而非为每种维度组合分别创建 limiter。
            labelForTRPCCalleeMethod, // 维度 method
            labelCity)) // 维度 city
```

其中 `labelCity` 需要你自己实现，比如：
```aidl
func labelCity(ctx context.Context, req, rsp interface{}) (key, val string) {
    return "city", getCityFromCtx(ctx)
}
```

将 `l` 注册为一个新的名为 limiter_by_city 的拦截器：
```aidl
filter.Register("limiter_by_city", l.Intercept, nil)

```
在 `trpc_go.yaml` 中配置拦截器：
```yaml
server:
  service:
    - name: trpc.app.server.service
      filter: [limiter_by_city]
      # ...
```
在北极星控制台创建一个维度有 city: Peking 的限流策略：
![polaris_config_limiter_city](/.resources/user_guide/overload_control/polarisconfiglimitercity.png)

需要注意的是，北极星采用[维度匹配规则](https://git.woa.com/trpc/trpc-proposal/blob/master/A9-polaris-limiter.md#%E5%8C%97%E6%9E%81%E6%98%9F%E9%99%90%E6%B5%81)，请尽可能地将所有可能用到的维度合并进同一个 limiter 中，而非为每种维度组合分别创建 limiter。

## FAQ
Q1：过载时的错误码是什么？
- 过载保护：server 端返回 22，client 端返回 124。
- 北极星限流：server 端返回 23，client 端返回 123。
## OWNER
cooperyan