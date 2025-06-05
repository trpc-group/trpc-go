**注**：框架提供了 [trpc-robust 插件](https://iwiki.woa.com/p/4012215462) 和 [trpc-overload-control 插件](https://iwiki.woa.com/p/776262500) 两种过载保护实现，其区别和使用场景参考 [tRPC-Go 过载保护](https://iwiki.woa.com/p/4012215466)

robust 包提供基于请求优先级的自适应过载保护插件。

相关提案：

* [A23-robust_system_fields.md](https://git.woa.com/trpc/trpc-proposal/blob/master/A23-robust_system_fields.md)
* [A24-robust_system_oc_algorithm.md](https://git.woa.com/trpc/trpc-proposal/blob/master/A24-robust_system_oc_algorithm.md)
* [A25-robust_system_config.md](https://git.woa.com/trpc/trpc-proposal/blob/master/A25-robust_system_config.md)

## 如何使用

robust 以 tRPC 插件的形式使用，具体使用方式如下：

### 1. 注册插件

```golang
import (
    // 匿名引入来注册 tRPC-Go 插件
    _ "git.woa.com/trpc-go/trpc-robust"
    // 再匿名 import metrics-runtime 插件，用于透明过载数据以便于治理
    // 见 https://trpc.woa.com
    _ "git.code.oa.com/trpc-go/trpc-metrics-runtime"
)
```

（必须，更新到最新版本）然后执行以下命令来获取最新版的 `trpc-robust` 以及 `metrics-runtime`：

```shell
go get git.code.oa.com/trpc-go/trpc-robust@latest
go get git.code.oa.com/trpc-go/trpc-metrics-runtime@latest
```

然后执行 `go mod tidy`。

### 2. 增加插件配置

过载保护插件的服务端配置位置分为两种：

* filter 前过载保护：过载保护插件在 filter 前、decode 后生效，这样的话被拒绝的请求不会走反序列化逻辑，性能更高，但是由于不走 filter，因此监控上报、降级策略等基于 filter 的逻辑会走不到（但是不影响 <https://trpc.woa.com> 柔性上报）
* filter 中过载保护：过载保护插件配置在 filter 中，需要配置到监控拦截器之后，以便监控能够捕捉到过载保护插件产生的过载保护错误，由于走 filter 时就已经执行了反序列化逻辑，因此不管请求是否拒绝，反序列化的开销一定存在，即使请求全部拒绝，这些请求仍然至少存在反序列化的开销，好处是监控拦截器以及其他业务拦截器可以走到，方便实现一些降级策略

**注意**：以下两种配置（filter 前过载保护 或 filter 中过载保护）只能二选一，如果两个都配的，同一个请求会走两次过载判断及上报逻辑

**配置调节注意**：一般参数不需要更改，特殊场景下常见需要调节的参数如下：

* 如果服务中存在协议没有使用 trpc tcp transport 时（比如 HTTP 协议），需要配置 `start_overload_sleep_drift_ms: 2` 以及 `start_overload_ms: 0`
* 如果想要调节用于判断过载的 CPU 使用率，建议观察业务高峰期时节点维度上最高的 CPU 使用率（一定要看每个节点自身的 CPU 使用率，不要看大盘平均值，高峰期一般是某一些节点的 CPU 使用率过高导致失败率上升，但是看大盘的平均 CPU 使用率的话会掩盖掉这些节点的异常），然后设置 `start_overload_cpu_usage` 为异常高负载节点 CPU 使用率的 80% 左右，比如高峰期时节点 CPU 使用率最高为 100%，那么可以设置 `start_overload_cpu_usage` 为 80% 左右
* 如果想控制 CPU 使用率与调度时延（wait_latency，即 goroutine 调度耗时 `start_overload_ms` 或睡眠漂移 `start_overload_sleep_drift_ms` 的统称）的生效关系，可以设置 `overload_policy` 为以下四种之一（要求 trpc-robust 版本 >= v0.0.10）：
  * "wait_latency && cpu" 表示等待请求处理时间超过 wait_latency 阈值并且 CPU 使用率超过 start_overload_cpu_usage 时，开始过载保护（默认值）
  * "wait_latency || cpu" 表示等待请求处理时间超过 wait_latency 阈值或者 CPU 使用率超过 start_overload_cpu_usage 时，开始过载保护
  * "wait_latency" 表示等待请求处理时间超过 wait_latency 阈值时，开始过载保护
  * "cpu" 表示 CPU 使用率超过 start_overload_cpu_usage 阈值时，开始过载保护

#### filter 前过载保护

请在框架配置文件 `trpc_go.yaml` 中增加对应插件配置

```yaml
server:
  filter:  # 配了 filter 前过载保护时，filter 处一定不要再配置 trpc-robust filter，否则同一请求会走两次过载判断
  overload_ctrl: trpc-robust  # 对于 trpc-robust 插件，此处固定配置为 trpc-robust 即可，要求 trpc-go 框架版本 >= v0.19.0
  service:
    - name: xxx
      overload_ctrl: trpc-robust  # 对于 trpc-robust 插件，此处固定配置为 trpc-robust 即可，要求 trpc-go 框架版本 >= v0.8.1
plugins:
  runtime:
    stat: # 必须配置，用于 metrics-runtime 插件，可以透明过载数据，当请求拒绝时方便排查原因
  overload_control:
    trpc-robust:
      server:
        # 以下两个指标 update_every_request 以及 update_duration 为或关系
        # 建议将 update_every_requests 设置为一个较大值，使 update_duration 作为主要优先级阈值的触发条件
        update_every_requests: 100000  # 每处理这么多请求就更新一下优先级阈值，一般不需要更改
        update_duration: 1s  # 每经过这么长时间就更新一下优先级阈值，一般不需要更改

        # 过载保护策略，支持以下四种形式：（要求 trpc-robust 版本 >= v0.0.10）
        # 1. "wait_latency && cpu" 表示等待请求处理时间超过 wait_latency 阈值并且 CPU 使用率超过 start_overload_cpu_usage 时，开始过载保护
        # 2. "wait_latency || cpu" 表示等待请求处理时间超过 wait_latency 阈值或者 CPU 使用率超过 start_overload_cpu_usage 时，开始过载保护
        # 3. "wait_latency" 表示等待请求处理时间超过 wait_latency 阈值时，开始过载保护
        # 4. "cpu" 表示 CPU 使用率超过 start_overload_cpu_usage 阈值时，开始过载保护
        # 默认为 "wait_latency && cpu"
        overload_policy: "wait_latency && cpu" # 一般不需要更改

        # cpu 使用率阈值
        start_overload_cpu_usage: 0.8  # 取值范围 (0,1)
        # wait_latency 阈值，分为两种：
        # 1. start_overload_ms 表示 goroutine 调度耗时阈值，单位毫秒，可为浮点数
        # 2. start_overload_sleep_drift_ms 表示 goroutine 睡眠漂移阈值，单位毫秒，可为浮点数，与 start_overload_ms 二选一
        start_overload_ms: 2  # goroutine 调度耗时阈值，单位毫秒，可为浮点数
        # 注意：如果服务中存在协议没有使用 trpc tcp transport 时（比如 HTTP 协议），
        # 需要配置 start_overload_sleep_drift_ms: 2 以及 start_overload_ms: 0
        start_overload_sleep_drift_ms: 0  # goroutine 睡眠漂移阈值，单位毫秒，可为浮点数，与 start_overload_ms 二选一

        # 以下两个配置用于消除 CPU 毛刺带来的偶发瞬时过载拒绝，要求 trpc-robust 版本 >= v0.0.6
        # 用三个状态来表示毛刺消除所处于的阶段：
        #  正常状态：未过载
        #  准备状态：观测到 CPU 高于阈值时就立即从正常状态迁移到这个准备状态，准备状态不会实质拒绝任何请求
        #  过载状态：处于准备状态后，假如在 start_reject_grace_period 这段时间内，CPU 高于阈值的请求比例大于 70%（固定比例），从准备状态迁移到过载状态（否则直接回到正常状态），在过载状态中，算法判定的过载请求均会被实质拒绝
        # 处于过载状态时，假如 CPU 持续 quiescent_period 时间都低于阈值，那么回归到正常状态
        # start_reject_grace_period 表示在 CPU 处于高负载维持多长时间后才开始对新判断的过载请求做实质的拒绝
        # 这个值越大，可以忍受的 CPU 毛刺时间越长，但是对于高负载的灵敏度也会降低
        start_reject_grace_period: 3s  # 默认值为 3s
        # quiescent_period 表示在没有过载请求多久之后把状态重置
        quiescent_period: 1m  # 默认值为 1m
```

#### filter 中过载保护

请在框架配置文件 `trpc_go.yaml` 中增加对应插件配置

```yaml
server:
  filter:
    # 注意：在 filter 这里 galileo 这一项不是必须配置，只有 trpc-robust 是必须配置
    # 这里写 galileo 是作为示例，意思是 trpc-robust 要放在 galileo 这种监控拦截器之后
    # 即 trpc-robust 要放到用户使用的监控拦截器之后
    - galileo  # 将监控拦截器放在 overload_control 之前，从而能够上报服务端的过载错误
    - trpc-robust  # 注册 robust filter
plugins:
  runtime:
    stat: # 必须配置，用于 metrics-runtime 插件，可以透明过载数据，当请求拒绝时方便排查原因
  overload_control:
    trpc-robust:
      server:
        # 以下两个指标 update_every_request 以及 update_duration 为或关系
        # 建议将 update_every_requests 设置为一个较大值，使 update_duration 作为主要优先级阈值的触发条件
        update_every_requests: 100000  # 每处理这么多请求就更新一下优先级阈值，一般不需要更改
        update_duration: 1s  # 每经过这么长时间就更新一下优先级阈值，一般不需要更改

        # 过载保护策略，支持以下四种形式：（要求 trpc-robust 版本 >= v0.0.10）
        # 1. "wait_latency && cpu" 表示等待请求处理时间超过 wait_latency 阈值并且 CPU 使用率超过 start_overload_cpu_usage 时，开始过载保护
        # 2. "wait_latency || cpu" 表示等待请求处理时间超过 wait_latency 阈值或者 CPU 使用率超过 start_overload_cpu_usage 时，开始过载保护
        # 3. "wait_latency" 表示等待请求处理时间超过 wait_latency 阈值时，开始过载保护
        # 4. "cpu" 表示 CPU 使用率超过 start_overload_cpu_usage 阈值时，开始过载保护
        # 默认为 "wait_latency && cpu"
        overload_policy: "wait_latency && cpu" # 一般不需要更改

        # cpu 使用率阈值
        start_overload_cpu_usage: 0.8  # 取值范围 (0,1)
        # wait_latency 阈值，分为两种：
        # 1. start_overload_ms 表示 goroutine 调度耗时阈值，单位毫秒，可为浮点数
        # 2. start_overload_sleep_drift_ms 表示 goroutine 睡眠漂移阈值，单位毫秒，可为浮点数
        start_overload_ms: 2  # goroutine 调度耗时阈值，单位毫秒，可为浮点数
        # 注意：如果服务中存在协议没有使用 trpc tcp transport 时（比如 HTTP 协议），
        # 需要配置 start_overload_sleep_drift_ms: 2 以及 start_overload_ms: 0
        start_overload_sleep_drift_ms: 0  # goroutine 睡眠漂移阈值，单位毫秒，可为浮点数，与 start_overload_ms 二选一

        # 以下两个配置用于消除 CPU 毛刺带来的偶发瞬时过载拒绝，要求 trpc-robust 版本 >= v0.0.6
        # 用三个状态来表示毛刺消除所处于的阶段：
        #  正常状态：未过载
        #  准备状态：观测到 CPU 高于阈值时就立即从正常状态迁移到这个准备状态，准备状态不会实质拒绝任何请求
        #  过载状态：处于准备状态后，假如在 start_reject_grace_period 这段时间内，CPU 高于阈值的请求比例大于 70%（固定比例），从准备状态迁移到过载状态（否则直接回到正常状态），在过载状态中，算法判定的过载请求均会被实质拒绝
        # 处于过载状态时，假如 CPU 持续 quiescent_period 时间都低于阈值，那么回归到正常状态
        # start_reject_grace_period 表示在 CPU 处于高负载维持多长时间后才开始对新判断的过载请求做实质的拒绝
        # 这个值越大，可以忍受的 CPU 毛刺时间越长，但是对于高负载的灵敏度也会降低
        start_reject_grace_period: 3s  # 默认值为 3s
        # quiescent_period 表示在没有过载请求多久之后把状态重置
        quiescent_period: 1m  # 默认值为 1m
```

注：`trpc-metrics-runtime` 插件配置（`runtime:stat`）用于 tRPC 官方柔性治理，链接见 <https://trpc.woa.com>

### 3. 【可选】插件详细配置

如果较熟悉算法，可以针对服务优化下算法配置，详细如下：

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
  overload_control:
    trpc-robust:
      server:
        # 以下两个指标 update_every_request 以及 update_duration 为或关系
        # 建议将 update_every_requests 设置为一个较大值，使 update_duration 作为主要优先级阈值的触发条件
        update_every_requests: 100000  # 每处理这么多请求就更新一下优先级阈值
        update_duration: 1s  # 每经过这么长时间就更新一下优先级阈值
        # 如果新收到的请求距离上一次收到的请求已经超过了过期时间，就会将优先级阈值设置为最低
        expire_duration: 10s  
        start_overload_ms: 2  # goroutine 调度耗时阈值，单位毫秒，可为浮点数
        # 注意：如果服务中存在协议没有使用 trpc tcp transport 时（比如 HTTP 协议），
        # 需要配置 start_overload_sleep_drift_ms: 2 以及 start_overload_ms: 0
        start_overload_sleep_drift_ms: 0  # goroutine 睡眠漂移阈值，单位毫秒，可为浮点数，与 start_overload_ms 二选一
        # 超过上述阈值后每一毫秒对应的负载点数，一般不需要更改
        # 不参与算法的实际工作，只用于观测，用于表征负载程度
        point_per_ms: 30  
        overload_recover_fail_count: 3  # 从过载状态恢复时，假如排队时间的增加次数超过这个配置，则判断为仍处于过载状态，一般不需要更改
        # 认为 CPU 使用率为多少以上才处于高负载状态
        # 此处可以大约认为最终过载时在调整后的 CPU 使用率在该值上下进行波动
        start_overload_cpu_usage: 0.8  # 取值范围 (0,1)
        cpu_usage_interval: 1s  # CPU 利用率采集的时间范围，一般不需要更改
      client:
        # 分为 sre, dagor 两种策略
        strategy: sre  # 选择 sre 或 dagor，为空的时候默认为 sre
        # strategy 为 sre 时以下选项生效
        overload_error_codes: [22,23]  # 判断下游是否过载的错误码
        start_overload_success_rate: 0.5  # 开始过载的成功率，低于此值认为下游过载，取值区间 (0,1)
        window: 1s  # 统计时间窗口大小
        max_reject_rate: 0.99  # 最大拒绝概率，取值范围 [0,1]，一般不需要更改
        start_working_request: 300  # 在窗口期，请求量少于此值主调过载保护不生效
        # strategy 为 dagor 时以下选项生效
        cleanup_interval: 10s  # 客户端记录各个节点优先级阈值的清理时间间隔（清理时只清理过期的优先级阈值）
        expire_time_in_seconds: 5  # 客户端记录各个节点优先级阈值的过期时间
```

### 常见问题

#### 优先级设置

推荐在客户端侧通过拦截器设置（服务端也可以通过拦截器设置）：

```go
import (
    rcodec "git.code.oa.com/trpc-go/trpc-utils/robust/codec"
)

// 按需使用客户端/服务端拦截器来使用优先级功能

func clientFilter(
    ctx context.Context,
    req, rsp interface{},
    handle filter.ClientHandleFunc,
) error {
    msg := codec.Message(ctx)
    // 设置客户端请求的用户优先级为 2，业务优先级为 0，非 VIP
    // 业务优先级最大值为 15，用户优先级最大值为 255
    userPriority, scenePriority, vip := uint16(2), uint8(0), false
    rcodec.WithClientRequestPriority(msg, userPriority, scenePriority, isVIP)
    // 设置 scene id (非必须)
    sceneID := "sceneID" // 场景标识，不同业务场景使用不同的 scene id
    rcodec.WithClientRequestSceneID(msg, sceneID)
    return handle(ctx, req, rsp)
}

func serverFilter(
    ctx context.Context,
    req interface{},
    handle filter.ServerHandleFunc,
) (interface{}, error) {
    msg := codec.Message(ctx)
    // 设置服务端请求的用户优先级为 2，业务优先级为 0，非 VIP
    // 业务优先级最大值为 15，用户优先级最大值为 255
    userPriority, scenePriority, vip := uint16(2), uint8(0), false
    rcodec.WithServerRequestPriority(msg, userPriority, scenePriority, isVIP)
    // 设置 scene id (非必须)
    sceneID := "sceneID" // 场景标识，不同业务场景使用不同的 scene id
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

这些优先级信息可以通过 trpc 协议的 metadata 一路透传，这些请求达到开启了 robust 插件的服务后，robust 会使用这些优先级信息来进行过载保护。

假如服务端收到的请求中找不到优先级信息，那么该请求会被随机分配一个优先级，并一路透传，该优先级的用户优先级部分取值范围为 `[0,255]`，业务优先级则固定为 `0`。

#### 如何做过载后的降级策略

当过载错误产生后，用户通常期望能够执行一些兜底逻辑，返回一些默认的数据以达到降级目的，这一功能可以通过在过载保护拦截器前面添加自定义的降级拦截器以实现类似的效果，比如：

```yaml
server:
  filter: 
    - fallback_logic  # 用于执行过载后的降级策略
    - trpc-robust
  service:
    - name: xxx
```

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
