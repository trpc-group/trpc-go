**注**：框架提供了 [trpc-robust 插件](https://iwiki.woa.com/p/4012215462) 和 [trpc-overload-control 插件](https://iwiki.woa.com/p/776262500) 两种过载保护实现，其区别和使用场景参考 [tRPC-Go 过载保护](https://iwiki.woa.com/p/4012215466)

## 1 前言

RPC 框架应该为服务提供稳定性保障。这里的稳定性指，当大量请求涌入时，应该：

- 保证成功请求的链路耗时稳定在较低的值，不会剧烈波动，也不会膨胀；
- 对于超出处理能力的请求，及早拒绝，防止链路超时；
- 避免因协程数过多或队列过长造成部分服务 OOM。

解决链路稳定性问题有两种不同的思路。一种是基于配额的限流策略，另一种是服务端自适应过载保护。
限流通过限制请求的 QPS，使整个服务链路保持在一个较低的负载水平下。它有单机和分布式两种，是常见的策略。
服务端自适应过载保护通过监控服务本身的运行状态，拒绝过多的请求，来使服务稳定在一个最佳的状态。

通常，服务端自适应过载保护足以为你的服务提供稳定性保障。如果你需要限制某类请求的流量，也可以将两者搭配起来使用。

## 2 服务端自适应过载保护
>
> 设计细节请参考 tRPC 提案：[A10_overload_control](https://git.woa.com/trpc/trpc-proposal/blob/master/A10-overload-control.md)。
> trpc-go 从 [v0.7.0](https://git.woa.com/trpc-go/trpc-go/blob/v0.7.0/CHANGELOG.md) 开始支持服务端自适应过载保护。
> 请与算法库 [trpc-go/trpc-overload-control](https://git.woa.com/trpc-go/trpc-overload-control) 配合使用。
> 请使用最新版 [v1.4.2](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.4.2/CHANGELOG.md)。

tRPC-Go 提供了基于三种指标的过载保护策略：

1. 协程最大调度耗时（对应配置项 `goroutine_schedule_delay` ）

- 解释：在框架的服务端异步逻辑中，任务会先放在协程池中，在放入协程池之前记录一个 start，在协程池开始正式运行这个任务时再记录一个 end，协程调度耗时即为 end-start，当过载发生时，该指标会明显增大，[实现](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.4.6/goroutine_schedule_delay_metric_sink.go#L36)
- 注意：由于该指标为框架内部使用协程池时进行上报而得，因此只用于框架 tcp transport 并开启了服务端异步的（默认开启）场景，比如基于 tcp 的 trpc 协议（以及只实现了 codec，复用框架 tcp transport 的其他业务协议），对于其他情况（比如 udp，自定义 transport 实现，http 系列的协议等），请使用协程睡眠漂移（`sleep_drift`）

2. 协程睡眠漂移（对应配置项 `sleep_drift` ）

- 解释：在一个背景 goroutine 中循环执行 `time.Sleep(interval)`，计算实际的睡眠时间相比 `interval` 增加了多少，当过载发生时，该指标会明显增大，[实现](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.4.6/probe_sleep_drift.go#L25)

3. 请求最大耗时（默认不开启，对应配置项 `request_latency` ）

- 解释：完整请求的耗时（包括所有业务逻辑的处理时间，即收到请求到返回响应的总耗时）

**注意：** 这三个指标任选其一即可，推荐 `goroutine_schedule_delay` 和 `sleep_drift` 二选一

[过载保护算法库](https://git.woa.com/trpc-go/trpc-overload-control) 会自动计算这些指标，用户通过配置来指定这些指标的最大期望值。

当框架监控到指标大于最大期望值时，会基于优先级（由客户端通过元数据透传，见下一节），对部分请求进行限流，立刻返回过载错误，保证它稳定在最大期望值附近，从而保证服务稳定。

<span style="color:red">请先在测试环境开启「日志监控」和「dry-run 模式」，确保默认协程调度耗时表现良好。</span>
<span style="color:red">如果你的上游是 tRPC-CPP 服务，请确保它的版本大于 [v0.10.0](https://git.woa.com/trpc-cpp/trpc-cpp/blob/v0.10.0/CHANGELOG.md)。因为在此之前，CPP 的北极星熔断会统计过载错误，不符合过载保护的预期。</span>
<span style="color:red">使用自适应过载保护，服务 QPS 需要至少在 300 以上，QPS 很低建议直接使用基于 Quota 的限流，如 [rate limiter](https://pkg.go.dev/golang.org/x/time/rate) 或者本文档中的北极星限流。</span>

要开启自适应过载保护，只需要匿名导入过载保护包，该包会注册名为 `overload_control` 的默认过载保护拦截器。并同时引入 `metrics-runtime` 以透明过载数据。

> 如果你已经是 trpc-overload-ctrl 的用户，升级时只需要注意以下两点：
>
> 1. trpc-overload-ctrl 以及 trpc-metrics-runtime 均使用最新版
> 2. 插件配置中要加上 runtime: stat 项以加载 metrics-runtime 插件

```go
import _ "git.code.oa.com/trpc-go/trpc-overload-control/filter"

// 再匿名 import metrics-runtime 插件，用于透明过载数据以便于治理
// 见 https://trpc.woa.com
import _ "git.code.oa.com/trpc-go/trpc-metrics-runtime"
```

然后执行以下命令来获取最新版的 `trpc-overload-control` 以及 `metrics-runtime`：

```shell
go get git.code.oa.com/trpc-go/trpc-overload-control@latest
go get git.code.oa.com/trpc-go/trpc-metrics-runtime@latest
```

然后执行 `go mod tidy`。

再将 `overload_control` 拦截器以及 `metrics-runtime` 插件添加到 `trpc_go.yaml` 中：

**注：**

- `runtime: stat` 中可能用到的 `bu_id` 可以暂时前往 <https://iwiki.woa.com/p/4012207091> 进行申请
- 柔性治理平台链接：<https://trpc.woa.com>

在 `trpc_go.yaml` 配置中，需要配置框架服务端以及插件配置以使用 trpc-overload-ctrl 插件。

### 框架服务端配置

过载保护插件的服务端配置位置分为两种：

- filter 前过载保护：过载保护插件在 filter 前、decode 后生效，这样的话被拒绝的请求不会走反序列化逻辑，性能更高，但是由于不走 filter，因此监控上报、降级策略等基于 filter 的逻辑会走不到（但是不影响 <https://trpc.woa.com> 柔性上报）
- filter 中过载保护：过载保护插件配置在 filter 中，需要配置到监控拦截器之后，以便监控能够捕捉到过载保护插件产生的过载保护错误，由于走 filter 时就已经执行了反序列化逻辑，因此不管请求是否拒绝，反序列化的开销一定存在，即使请求全部拒绝，这些请求仍然至少存在反序列化的开销，好处是监控拦截器以及其他业务拦截器可以走到，方便实现一些降级策略

#### filter 前过载保护

```yaml
server:
  overload_ctrl: default  # 对于 trpc-overload-ctrl 插件，此处固定配置为 default 即可，要求 trpc-go 框架版本 >= v0.19.0
  service:
    - name: xxx
      overload_ctrl: default  # 对于 trpc-overload-ctrl 插件，此处固定配置为 default 即可，要求 trpc-go 框架版本 >= v0.8.1
```

**注意：** server 级别的配置在较高版本 trpc-go 才支持，可以对每个 service 分别配置 overload_ctrl 以使用 filter 前过载保护。

#### filter 中过载保护

```yaml
server:
  filter: 
    - galileo # 将监控拦截器放在 overload_control 之前，从而能够上报服务端的过载错误
    - overload_control
  service:
    - name: xxx
```

注意：服务端过载保护的 filter 要配在 `server` 下面，不要错配到 `client` 下面。

### 插件配置

```yaml
plugins:
  # 注意：必须添加 runtime: stat 插件配置以加载 metrics-runtime 插件
  runtime:
    stat:
      robust:
        # 上报数据至 trpc 官方柔性治理平台 https://trpc.woa.com
        debug: false  # 开启 debug 日志，默认关闭
        # bu_id 用于对业务进行标识，避免存在重复 app/server，以方便柔性平台查看（层级为 bu_id - app - server）
        # 123 平台用户可以直接删除此项，123 平台下该默认值为 "PCG-123"
        # 非 123 平台用户建议在柔性平台申请一个 id（不需要每个服务申请一个，该 id 可以在多个服务共用）进行填写，非 123 平台下该默认值为 "default"
        bu_id: some-bu-id
  overload_control: # 插件类型，必须是 overload_control
    # 插件名，同时也是插件注册的拦截器名
    # 如果使用 overload_control，则会覆盖注册的默认拦截器
    # 如果使用其他插件名，则必须在代码中手动调用一次 plugin.Register 方法进行注册
    overload_control:
      # 所有配置项都可以留空，这时会使用默认值
      server: # 服务端过载保护
        # 插件在 v1.4.7 版本之后，提供了导出变量以及 HTTP Handler 以供动态修改，详见 2.6 小节
        dry_run: false # 是否开启 dry run 模式，默认关闭，开启后，过载保护总是放行请求，可以在不影响业务的情况下通过日志来观察算法的状态
        # 以下三个指标只选其中一个开启即可，其余写为 0ms，推荐在 goroutine_schedule_delay 和 sleep_drift 中二选一，request_latency 一般不使用
        # 注意：goroutine_schedule_delay 只有在使用了框架的 tcp transport 并开启服务端异步（默认开启）时才能生效
        # 通常来说，对于 trpc 协议（使用 tcp 传输层协议）来说，goroutine_schedule_delay 均可生效
        # 并且要注意，这个 trpc 协议接口应当是进行压测以及服务上线的主要接口，才能时算法的 goroutine_schedule_delay 配置发挥最佳效果
        # 如果请求完全或者很少到达该接口，那么 goroutine_schedule_delay 将不会生效，此时需要手动配置 sleep_drift: 3ms 以开启睡眠飘移指标
        # 睡眠飘移指标不受协议的影响，均可适用，区别：在高 QPS 下，睡眠飘移的采样相较于协程调度耗时偏少（固定时间采样 vs 每个请求采样一次），实际效果以用户测试为准
        # 以上注意事项的实际例子：主要使用 HTTP RPC / HTTP 标准 / RESTful 等服务的情况下，goroutine_schedule_delay 会不生效，需要手动配置 sleep_drift: 3ms 以开启睡眠飘移指标
        goroutine_schedule_delay: 3ms # 期望的最大协程调度耗时，默认 3ms，如果你需要调整这个值，请先阅读过载保护提案
        sleep_drift: 0ms # 期望的最大协程睡眠漂移，0，默认不开启，如果你的服务没有使用原生 trpc 协议，或者只使用了 udp（协程调度耗时无法生效时）请将该值配为 3ms，如果你需要调整这个值，请先阅读过载保护提案
        request_latency: 0ms # 期望的最大请求耗时，0，默认不开启，如果你需要调整这个值，请先阅读过载保护提案
        # 注意：下面这个 cpu_threshold 指标的作用是开启算法，开启之后算法是否开始丢请求要看算法本身根据上述的三个指标计算的并发数来做决定
        # 也就是说这是一个基础指标，不是超过这个这个指标就开始丢请求，而是超过之后，算法开始走流程，走的流程中再经过上述的三个指标进行判定是否要丢弃请求
        # 此外，这个 cpu_threshold 不要调的过低，否则会导致误限，建议维持在 75% 以上水平
        cpu_threshold: 0.75 # 过载保护生效时的最低 CPU 使用率（整个容器的），默认 75%
        # 以下两个配置用于消除 CPU 毛刺带来的偶发瞬时过载拒绝，在 trpc-overload-ctrl 版本 >= v1.4.25 支持
        # 用三个状态来表示毛刺消除所处于的阶段：
        #  正常状态：未过载
        #  准备状态：观测到 CPU 高于阈值时就立即从正常状态迁移到这个准备状态，准备状态不会实质拒绝任何请求
        #  过载状态：处于准备状态后，假如在 start_reject_grace_period 这段时间内，CPU 高于阈值的请求比例大于 70%（固定比例），从准备状态迁移到过载状态（否则直接回到正常状态），在过载状态中，算法判定的过载请求均会被实质拒绝
        # 处于过载状态时，假如 CPU 持续 quiescent_period 时间都低于阈值，那么回归到正常状态
        # start_reject_grace_period 表示在 CPU 处于高负载维持多长时间后才开始对新判断的过载请求做实质的拒绝
        # 这个值越大，可以忍受的 CPU 毛刺时间越长，但是对于高负载的灵敏度也会降低
        start_reject_grace_period: 3s  # v1.4.28 后默认值为 3s
        # quiescent_period 表示在没有过载请求多久之后把状态重置
        quiescent_period: 1m  # v1.4.28 后默认值为 1m
```

CPU 毛刺消除的具体设计可以参考 [trpc-overload-ctrl 毛刺延迟判定设计](https://doc.weixin.qq.com/doc/w3_AGkAxgZOAFMbwGEOpWhQfa9DO0Vmm?scode=AJEAIQdfAAoTkYWg36AGkAxgZOAFM)。

启用 `sleep_drift` 的简要配置如下：

```yaml
plugins:
  runtime:
    stat:
      robust:
        # 上报数据至 trpc 官方柔性治理平台 https://trpc.woa.com
        debug: false  # 开启 debug 日志，默认关闭
        bu_id: some-bu-id
  overload_control: 
    overload_control:
      server:
        dry_run: false
        goroutine_schedule_delay: 0ms # 启用 sleep_drift 时此处必须显式设置为 0ms
        sleep_drift: 3ms 
        request_latency: 0ms 
        cpu_threshold: 0.75 
```

最简配置：

```yaml
plugins:
  runtime:
    stat:
  overload_control: 
    overload_control:
      server:
```

### 2.1 优先级

过载保护支持优先级功能，在服务过载时，优先让高优请求通过，拒绝低优请求。

实现中会自动从元数据中提取优先级信息，包括用户优先级和业务优先级：

- 用户优先级通过 `[0,255]` 之间的数字进行区分
- 业务优先级通过 `0,15` 之间的数字进行区分

数字越大，优先级越高。

如果在元数据中未发现优先级相关信息，则会按照如下规则进行默认生成，生成后的优先级会往下游一路透传：

- 默认生成的业务优先级为 `0`
- 默认生成的用户优先级在 `[0,255]` 范围内随机

业务优先级和用户优先级推荐在整个链路的入口服务中设置，其中业务优先级则推荐在整个链路上进行讨论做统一后，再行启用。

VIP 优先级为最高优先级，算法会保证这类请求的通过。

sceneID 为业务场景标识，推荐和业务优先级一块使用，不同的业务优先级对应不同的业务场景标识，该标识仅用于展示，不影响算法运行。

下面的方法可以给请求打上优先级标记：

```go
import (
    overloadctrl "git.code.oa.com/trpc-go/trpc-overload-control"
    rcodec "git.code.oa.com/trpc-go/trpc-utils/robust/codec"
)

// 按需使用客户端/服务端拦截器来使用优先级功能

func clientFilter(
    ctx context.Context,
    req, rsp interface{},
    handle filter.ClientHandleFunc,
) error {
    // 设置客户端请求的用户优先级为 2，业务优先级为 0，非 VIP
    // 业务优先级最大值为 15，用户优先级最大值为 255
    userPriority, scenePriority, vip := uint16(2), uint8(0), false
    ctx = overloadctrl.SetClientRequestPriorityAll(
        ctx,
        userPriority,
        scenePriority,
        vip,
    )
    // 设置 scene id (非必须)
    sceneID := "sceneID" // 场景标识，不同业务场景使用不同的 scene id
    msg := codec.Message(ctx)
    rcodec.WithClientRequestSceneID(msg, sceneID)
    return handle(ctx, req, rsp)
}

func serverFilter(
    ctx context.Context,
    req interface{},
    handle filter.ServerHandleFunc,
) (interface{}, error) {
    // 设置服务端请求的用户优先级为 2，业务优先级为 0，非 VIP
    // 业务优先级最大值为 15，用户优先级最大值为 255
    userPriority, scenePriority, vip := uint16(2), uint8(0), false
    ctx = overloadctrl.SetServerRequestPriorityAll(
        ctx,
        userPriority,
        scenePriority,
        vip,
    )
    // 设置 scene id (非必须)
    sceneID := "sceneID" // 场景标识，不同业务场景使用不同的 scene id
    msg := codec.Message(ctx)
    rcodec.WithServerRequestSceneID(msg, sceneID)
    return handle(ctx, req)
}

const filterName = "set_priority"

func init() {
    // 注册拦截器以便配置文件使用
    filter.Register(filterName, serverFilter, clientFilter)
}
```

代码用法：

```go
func main() {
    // 服务端拦截器在 trpc.NewServer 处添加 option 以进行使用
    s := trpc.NewServer(server.WithNamedFilter(filterName, serverFilter))
    // 客户端拦截器则在初始化 client proxy 时添加 option 以进行使用
    p := pb.NewHelloClientProxy(client.WithNamedFilter(filterName, clientFilter))
}

```

配置用法（使用配置后，不需要使用服务端或客户端的 option 做设置）：

```yaml
server:
  filter: 
    - set_priority
  service:
    - name: xxx
      filter:  # 为某个 service 单独设置
        - set_priority
client:
  filter: 
    - set_priority
  service:
    - name: xxx
      filter:  # 为某个 service 单独设置
        - set_priority
```

它会在请求的 meta data 中设置优先级，随着请求链，一路[透传](https://iwiki.woa.com/pages/viewpage.action?pageId=284269846)下去。

### 2.2 plugin 配置

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
        # 插件在 v1.4.7 版本之后，提供了导出变量以及 HTTP Handler 以供动态修改，详见 2.6 小节
        dry_run: false # 是否开启 dry run 模式，默认关闭，开启后，过载保护总是放行请求，可以在不影响业务的情况下通过日志来观察算法的状态
        # 注意：goroutine_schedule_delay 只有在使用了框架的 tcp transport 并开启服务端异步（默认开启）时才能生效
        # 通常来说，对于 trpc 协议（使用 tcp 传输层协议）来说，goroutine_schedule_delay 均可生效
        # 并且要注意，这个 trpc 协议接口应当是进行压测以及服务上线的主要接口，才能时算法的 goroutine_schedule_delay 配置发挥最佳效果
        # 如果请求完全或者很少到达该接口，那么 goroutine_schedule_delay 将不会生效，此时需要手动配置 sleep_drift: 3ms 以开启睡眠飘移指标
        # 睡眠飘移指标不受协议的影响，均可适用（效果或许不如 goroutine_schedule_delay，以用户测试为准）
        # 以上注意事项的实际例子：主要使用 HTTP RPC / HTTP 标准 / RESTful 等服务的情况下，goroutine_schedule_delay 会不生效，需要手动配置 sleep_drift: 3ms 以开启睡眠飘移指标
        goroutine_schedule_delay: 3ms # 期望的最大协程调度耗时，默认 3ms，如果你需要调整这个值，请先阅读过载保护提案
        sleep_drift: 0ms # 期望的最大协程睡眠漂移，0，默认不开启，如果你的服务没有使用原生 trpc 协议，或者只使用了 udp（协程调度耗时无法生效时）请将该值配为 3ms，如果你需要调整这个值，请先阅读过载保护提案
        request_latency: 0ms # 期望的最大请求耗时，0，默认不开启，如果你需要调整这个值，请先阅读过载保护提案
        # 注意：下面这个 cpu_threshold 指标的作用是开启算法，开启之后算法是否开始丢请求要看算法本身根据上述的三个指标计算的并发数来做决定
        # 也就是说这是一个基础指标，不是超过这个这个指标就开始丢请求，而是超过之后，算法开始走流程，走的流程中再经过上述的三个指标进行判定是否要丢弃请求
        # 此外，这个 cpu_threshold 不要调的过低，否则会导致误限，建议维持在 75% 以上水平
        cpu_threshold: 0.75 # 过载保护生效时的最低 CPU 使用率（整个容器的），默认 75%
        cpu_interval: 1s # 计算过去 1s 内的 CPU 使用率，这个值越大，过载保护在开启和关闭间切换得越慢，默认 1s
        log_interval: 0ms # 过载保护状态日志的最小时间间隔，用于调试，0 为不开启日志。过载保护日志级别为 Info
        # 以下是黑/白名单，当同时配置时，只有白名单会生效
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
```

**注意：** `dry_run: true` 的时候相当于过载保护算法实际不生效，只是显示一个 log 打印，因此不会产生任何错误码，也不会拒绝任何请求，必须设置 `dry_run: false` 时算法才会实际生效。

> 如何判断算法是否丢弃了请求？
>
> 可以查看日志 `grep overload`，信息大概如下所示：
>
> `INFO filter/plugin.go:210 default overload control, service: .., method: .., maxConcurrency: 123, inEffectStrategy: <nil>, ...`
>
> 假如 `inEffectStrategy` 的值不为 `<nil>` 的话，说明某个指标生效了。

> 如何评价算法的效果？
>
> 在压测时控制负载从小逐渐增到大（在一段时间内），然后将 `dry_run` 按照 `true`（开启）和 `false`（关闭）分别运行两次（运行过程中控制变量，保证只有 `dry_run` 的值是不同的）：
>
> - 在过载保护关闭时，现象是时延逐渐上升到不可控，最终请求大部分超时，成功率很低
> - 在过载保护开启时，现象是时延最终可以稳定下来，大部分请求成功，成功率最终维持在一个较高的水平
>
> **注意：** 算法的效果不是通过看 CPU 的负载来确定，不是说开启算法后，CPU 负载降得很低就说明算法有效，而是主要看 QPS 和时延的控制。

如果自定义插件名不为 `overload_control`，需要手动注册插件：

```go
import "git.code.oa.com/trpc-go/trpc-go/plugin"
import "git.code.oa.com/trpc-go/trpc-overload-control/filter"

plugin.Register(name, filter.NewPlugin(/* options */))
```

`NewPlugin` 方法可以接收参数，允许修改 plugin 创建的默认过载保护策略。而 yaml 配置，则可以在默认策略的基础上，进行调整。

plugin 会注册一个与插件名相同的拦截器，使用时，将插件名填入 filter 中即可。
注意，请将过载保护拦截器配在监控拦截器之后，这样被调监控就可以上报过载错误了。
当需要全局开启一个过载保护策略，而对某个 service 自定义策略时，可以将该 service 加入全局策略的黑名单中，再在 service 的 filter 中单独加一个拦截器。

### 2.3 通过代码添加过载保护拦截器

plugin 只提供了部分能力，通过代码，可以创建更加精细的过载保护策略。具体请参考 [`RegisterServer`](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/filter/filter.go#L39) 方法和 [`server`](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/overloadctrl.go#L28) 过载保护库的各种 [`Opt`](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/options.go#L45)。

### 2.4 如何对比过载保护使用前后的效果

本小节专门介绍业务如何通过压测来对比出使用过载保护的前后效果。

核心思想：控制压测速率、服务端其余各种配置、场景都不变，只变化过载保护的开启与否（通过调节 `dryrun` 参数，或者生效的算法参数，比如 `sleep_drift` 从 `0ms` 调节到 `3ms`），也就是控制单一变量。

- 压测端：
  - 错误做法：维护 n 个 goroutine 对应 n 个服务端的连接，每个 goroutine 上面循环发送接收，这种压测方式对应的发送速率实际上是不定的，在过载保护生效时，会更快地返回一个过载错误，而这个快速反应会导致压测端发送更多的请求，最终导致的效果就是过载保护生效时的实际承受的 QPS 更高，造成对比的不公平（没有控制单一变量）。
  - 正确做法：使用 [rate](https://pkg.go.dev/golang.org/x/time/rate) 等限流器使发送速率能够保持在一个可控的水平（而不是受到服务端自身的处理能力影响），在相同的发送速率场景下，观察开启过载保护前后，服务的成功请求数以及成功请求的平均耗时/P99 耗时。

实际业务示例：

- 压测端示例：<https://git.woa.com/docx-online/docx-online/merge_requests/29196>
  - 包含逻辑：初始发送速率为一个定值，维持一小段时间后，再逐渐增高到一个值，查看过载保护不生效与生效之间的区别，最后发送速率再回落到较低值，查看过载保护是否能够回归不限制的水平

实际可能的监控效果及解说见如下几个图片：

![cost](../../.resources/user_guide/overload_control/testing_cost.png)
![succ_percent](../../.resources/user_guide/overload_control/testing_succ_percent.png)
![cpu](../../.resources/user_guide/overload_control/testing_cpu.png)

从图中得到的几个总结点：

- 开启过载保护之后，大盘的成功率不一定变大，因为错误计算会包含过载错误
- 开启过载保护之后，CPU 使用率不会明显降低，因为良好的过载保护工作状态是（当过载发生时）维持系统处于高 CPU 利用率的同时能够以低耗时处理请求，而低 CPU 负载说明没有充分利用系统的性能
- 过载保护效果的实际衡量指标是**成功请求**的数量以及耗时，加了过载保护之后，相同的流量场景下，成功请求的数量会增多，并且耗时能够下降

### 2.5 如何做过载后的降级策略

当过载错误产生后，用户通常期望能够执行一些兜底逻辑，返回一些默认的数据以达到降级目的，这一功能可以通过在过载保护拦截器前面添加自定义的降级拦截器以实现类似的效果，比如：

```yaml
server:
  filter: 
    - fallback_logic  # 用于执行过载后的降级策略
    - overload_control
  service:
    - name: xxx
```

> **注意**：overload_control 插件的用法分为 filter 前配置和 filter 中配置，分别对应 1. 在 filter 前、decode 后做拦截，2. filter 链中做拦截，对于上述的降级策略来说，必须使用 filter  中配置形式的用法。

然后代码中注册该拦截器：

```go
import (
 "context"

 "git.code.oa.com/trpc-go/trpc-go/errs"
 "git.code.oa.com/trpc-go/trpc-go/filter"
)

func main() {
 // 在加载配置 (比如 trpc.NewServer) 前进行拦截器注册
 filter.Register("fallback_logic",
  func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
   rsp, err := next(ctx, req)
   if errs.Code(err) == errs.RetServerOverload { // 判断是过载错误，执行降级策略
    return fallbackLogic(ctx, req)
   }
   return rsp, err
  }, nil)
  // ... trpc.NewServer()
}

func fallbackLogic(ctx context.Context, req interface{}) (interface{}, error) {
 // ...
}
```

### 2.6 动态修改插件和 dry_run 开关

在 2.2 小节中提到，在配置文件中可以通过 `dry_run: true` 来开启插件的 dry_run 模式，`dry_run` 模式下插件不执行任何过载保护的逻辑，仅记录请求信息。

从 v1.4.7 版本开始，插件提供了导出变量和 HTTP Handler，用于动态修改过载保护插件和 `dry_run` 模式的开关。

其中：

- `DisableOverloadControl` 用于禁用 / 启用过载保护插件，当该变量的值为 `true` 时，插件将被禁用，不执行任何过载保护逻辑。
- `DryRunOverloadControl` 用于开启 / 关闭 `dry_run` 模式，当该变量的值为 `true` 时，插件将进入 `dry_run` 模式。

以下代码可以动态修改插件的配置：

```go
import (
    "git.code.oa.com/trpc-go/trpc-overload-control/flag"
)

func main() {
    // 禁用过载保护插件，禁用后，过载保护将不会生效
    // 也就是说，插件被禁用后，完全不执行过载保护的算法逻辑
    flag.DisableOverloadControl.Store(true)
    // 启用过载保护
    flag.DisableOverloadControl.Store(false)
    // 获取禁用过载保护的状态
    _ = flag.DisableOverloadControl.Load()
    // do something

    // 启用 dry run 模式，当插件未被禁用时，dry run 模式下的过载保护实际不会生效，但会记录过载保护相关日志
    // 也就是说，插件启用时，开启 dry run 模式后，插件将执行算法逻辑，但是不会根据算法计算的结果来实际丢弃请求
    // 可以根据打印的日志观测算法逻辑的执行表现
    flag.DryRunOverloadControl.Store(true)
    // 禁用 dry run 模式，插件未被禁用时，会根据算法计算结果来实际丢弃过载的请求
    flag.DryRunOverloadControl.Store(false)
    // 获取 dry run 模式的状态
    _ = flag.DryRunOverloadControl.Load()
    // do something
}
```

此外，`DisableOverloadControl` 和 `DryRunOverloadControl` 都支持通过监听远程配置来动态修改，从而可以通过远程配置的改动来动态开关过载保护功能，例如：

```go
func Reload(c *config.MangoConndConfig) {
    if c.OverLoadJSONConfig.Switch == 1 {
        log.Infof("Switch on")
        flag.DisableOverloadControl.Store(false)
    } else {
        log.Infof("Switch off")
        flag.DisableOverloadControl.Store(true)
    }
}
```

同时，插件向 `trpc-go` 的 `admin` 注册了 HTTP Handler，支持指定 `admin` 的 `ip:port` 通过 HTTP 接口动态修改过载保护插件和 `dry_run` 的配置：

- 禁用过载保护：`curl -X PUT "http://admin_ip:port/cmds/overloadctrl?disable=1"`
- 启用过载保护：`curl -X PUT "http://admin_ip:port/cmds/overloadctrl?disable=0"`
- 启用 `dry_run` 模式：`curl -X PUT "http://admin_ip:port/cmds/overloadctrl?dryrun=1"`
- 禁用 `dry_run` 模式：`curl -X PUT "http://admin_ip:port/cmds/overloadctrl?dryrun=0"`
- 获取配置当前值：`curl "http://admin_ip:port/cmds/overloadctrl"`

HTTP 接口返回示例：

```json
{
  "currentDisableOverloadControl": false,
  "currentDryRunOverloadControl": false,
  "errorcode": 0,
  "message": "Note: If you want to modify disable or dryrun flags in OverloadControl, please use HTTP PUT method."
}
```

**注意：**

- 当使用 HTTP 接口时，只有 `PUT` 方法才能用于修改配置。而 `GET` 方法仅用于获取当前 `flags` 的值。
- 在业务逻辑层面，`DisableOverloadControl` / `disable` 的优先级高于 `DryRunOverloadControl` / `dryrun`。这意味着：
  - 当插件被禁用时（即 `DisableOverloadControl.Load() == true` ）时，整个插件的功能都不会生效。也就是说，此时无论 `dry_run`
      如何设置，过载保护的逻辑都不会执行。但是仍然可以修改 `DryRunOverloadControl` 的值，对其的修改将保留在该变量中。
  - 插件在启用的情况下（即 `DisableOverloadControl.Load() == false` ）时，对 `DryRunOverloadControl` 的修改效果与修改
      plugin 的 `dry_run` 字段相同，如 2.2 节所述。
- 仅从值的修改上说，`DisableOverloadControl` 和 `DryRunOverloadControl`
  具有相同的优先级，两者不存在依赖关系，因此不必拘泥于修改顺序的先后。
- 再次提醒，只有插件启用且关闭 `dry_run` 模式的情况下，过载保护才会实际生效，实际效果请参考如下表格：

| `disable` | `dry_run` | 实际效果                      |
|-----------|-----------|---------------------------|
| `false`   | `false`   | 过载保护生效，会拦截实际请求            |
| `false`   | `true`    | 仅执行过载保护算法逻辑并打印日志，不会拦截实际请求 |  
| `true`    | `false`   | 过载保护不生效                   |
| `true`    | `true`    | 过载保护不生效                   |  

## 3 限流

tRPC 目前提供了基于北极星的限流策略，请参考这个[提案](https://git.woa.com/trpc/trpc-proposal/blob/master/A9-polaris-limiter.md#%E5%8C%97%E6%9E%81%E6%98%9F%E9%99%90%E6%B5%81)。

### 3.1 北极星

tRPC-Go 的北极星限流是通过 [trpc-filter/polaris/limiter](https://git.woa.com/trpc-go/trpc-filter/tree/master/limiter/polaris) 插件实现的。它是对北极星 [SDK](https://git.woa.com/polaris/polaris-go) 的封装，让 tRPC 用户方便地接入北极星限流。当请求被限流时，服务端会返回框架错误码 `23`，客户端会返回框架错误码 `123`。
详细的北极星限流能力请参考[访问限流使用指南](https://iwiki.woa.com/pages/viewpage.action?pageId=89656472)。下面简单介绍插件及限流策略的配置。

#### 3.1.1 tRPC-Go 服务配置

在代码中匿名引用插件：

```go
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

#### 3.1.2 在北极星控制台配置限流策略

> 本节的北极星截图不保证与最新版北极星控制台一致。

这是[北极星控制台](http://v2.polaris.woa.com/#/services/list)。找到你的服务后，新建限流策略：
![polaris_console](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconsole.png)
北极星支持分布式和单机限流两种模式，我们以分布式限流为例子，单机限流策略配置类似。
![polaris_config_limiter](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconfiglimiter.png)
大部分配置字段都有清晰的含义，这里，我们只关注如何填写维度。

tRPC-Go 北极星限流插件是[基于北极星 SDK 访问限流](https://iwiki.woa.com/p/89656472)的能力，限流插件会上报两个维度：`method` 和 `caller`，即被调方法名和主调服务名。
主调服务 `trpc.app.server.service_A` 调用被调服务  `trpc.app.server.service_B`时，插件的 client 端或 server 端限流都是基于被调服务 `trpc.app.server.service_B` 在北极星平台配置的限流规则。
下面列举几个主调服务 `trpc.app.server.service_A` 调用被调服务  `trpc.app.server.service_B` 的限流场景。

##### 场景 1

为被调服务  `trpc.app.server.service_B` 的方法 `M1` 配置一个 100/s 的限流值，可以这样填写：
![polaris_config_limiter_m1](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconfiglimiterm1.png)
创建出下面的限流策略：
![polaris_config_limiter_m1_policies](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconfiglimiterm1policies.png)

你可以在 `trpc.app.server.service_B` 开启 server 端限流，此时会在请求到达 `trpc.app.server.service_B` 之后被限流。

```yaml
server:
  filter: [polaris_limiter]  # 开启 server 端限流
  service: trpc.app.server.service_B
```

你可以在 `trpc.app.server.service_A` 开启 client 端限流，此时会在请求到达 `trpc.app.server.service_B` 之前提前被限流。

```yaml
client:
  filter: [polaris_limiter]
  service: trpc.app.server.service_B
```

##### 场景 2

限制被调服务  `trpc.app.server.service_B` 的方法 `M1` 的请求数为 100/s，限制来自上游 `trpc.app.server.service_A`，调用被调服务  `trpc.app.server.service_B` 的方法 `M2` 的请求数为 50/s，需要配置两个限流策略。第一个策略与场景1一样，第二个策略如下配置维度：
![polaris_config_limiter_m1_scenario2](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconfiglimiterm1scenario2.png)
最终创建出下面的两个限流策略：
![polaris_config_limiter_m1_policies_scenario2](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconfiglimiterm1policiesscenario2.png)

##### 场景 3：自定义维度

默认限流插件只提供了 `method` 和 `caller` 两个维度。如果你要基于自定义维度进行限流，必须自行注册一个新的 filter。
比如，你想对来自北京的请求进行限流，即需要两个维度：`method` 和 `city`。
自定义新的 limiter：

```go
 l, err := limiter.New(
    limiter.WithSDKCtx(polarisSDKCtx), // 多个 limiter 可以复用同一个北极星 SDK context。省略时，New 方法自动初始化一个新的北极星 SDK context。
    limiter.WithNamespace(namespaceForTRPCServer), // 必填，因为是服务端限流，所以使用 namespaceForTRPCServer。
    limiter.WithService(serviceForTRPCCallee), // 必填
    limiter.AddLabels(
        labelForTRPCCallerService, // 尽可能地将所有可能用到的维度合并进同一个 limiter 中，而非为每种维度组合分别创建 limiter。
        labelForTRPCCalleeMethod, // 维度 method
        labelCity),
    ) // 维度 city
```

其中 `labelCity` 需要你自己实现，比如：

```go
func labelCity(ctx context.Context, req, rsp interface{}) (key, val string) {
    return "city", getCityFromCtx(ctx)
}
```

将 `l` 注册为一个新的名为 limiter_by_city 的拦截器：

```go
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

在北极星控制台创建一个维度有 `city: Peking` 的限流策略：
![polaris_config_limiter_city](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/overload_control/polarisconfiglimitercity.png)

需要注意的是，北极星采用[维度匹配规则](https://git.woa.com/trpc/trpc-proposal/blob/master/A9-polaris-limiter.md#%E5%8C%97%E6%9E%81%E6%98%9F%E9%99%90%E6%B5%81)，请尽可能地将所有可能用到的维度合并进同一个 limiter 中，而非为每种维度组合分别创建 limiter。

## 4 FAQ

### Q1：过载时的错误码是什么？

- 过载保护：server 端返回 `22`，client 端返回 `124`。
- 北极星限流：server 端返回 `23`，client 端返回 `123`。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
