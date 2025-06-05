[English](README.md) | 中文

# 1. Metrics

监控指标可以简单地理解为一系列的数值测量。
不同的应用程序，需要测量的内容不同。
例如，对于 Web 服务器，可能是请求时间；对于数据库，可能是活动连接或活动查询的数量，等等。

监控指标在理解你的应用程序为什么以某种方式工作方面起着重要作用。
假设你正在运行一个 Web 应用程序，并发现它运行缓慢。
要了解你的应用程序发生了什么，你需要一些信息。
例如，当请求量很高时，应用程序可能会变慢。
如果你有请求计数监控指标，你可以确定原因并增加服务器数量以应对负载。

# 2. 指标类型

根据监控指标数据维度上的不同，监控指标可以分为单维监控指标和多维监控指标。

## 2.1 单维监控指标

单维监控指标由指标名字、指标值和指标聚合策略三部分组成。
指标名字唯一地标识了单维监控指标。
指标聚合策略描述了如何将指标值聚合在一起，如求和、取平均值、取最大值和取最小值。
例如你要监控 CPU 的平均负载，则可以定义并上报指标名称为 "cpu.avg.load" 的单维监控指标：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/metrics"
)

if err := metrics.ReportSingleDimensionMetrics("cpu.avg.load", 70.0, metrics.PolicyAVG); err != nil {
    log.Infof("reporting cpu.avg.load metric failed: %v", err)
}
```

### 2.1.1 常用的监控指标

根据聚合策略，指标值的值域以及对指标值可能采取的操作，metrics 包提供了 counter、gauge、timer 和 histogram 等几种常见类型的单维监控指标。
建议优先考虑使用这几种内置的常见监控指标，如果不能满足需求，再自定义其他类型的单维监控指标。

#### 2.1.1.1 Counter

Counter 用于统计某类指标的累积量，它将保存从系统启动开始持续的累加值。
支持对 Counter 进行 +1, -1, -n, +n 的操作。注意这里定义的 Counter 和其他的监控系统可能不一样，例如 [prometheus 中的 Counter](https://prometheus.io/docs/concepts/metric_types/#counter) 的值是只能单调递增，此时如果使用 [trpc-metrics-prometheus](https://git.woa.com/trpc-go/trpc-metrics-prometheus) 对 Counter 进行 "-1" 操作，则可能会报错。
这种情况下，建议使用 Gauge，或者使用两个 Counter，最后两个 Counter 的值相减。
例如你要监控某个微服务的请求数量，则可以定义一个指标名称为 "request.num" 的 Counter：

```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```

#### 2.1.1.2 Gauge

Gauge 用于统计某类指标的时刻量。
例如你要监控 CPU 的平均负载，则可以定义并上报指标名称为 "cpu.load.avg" 的 Gauge：

```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

_ = metrics.Gauge("cpu.avg.load")
metrics.SetGauge("cpu.avg.load", 0.75)
```

Gauge 只能设置值，不能对值进行累加，如果需要对值进行累加的话，你可以在 Gauge 的基础上封装一层：

```go
import (
    "sync"
    "git.code.oa.com/trpc-go/trpc-go/metrics"
)


metrics.RegisterMetricsSink(metrics.NewConsoleSink())
g := newGauge(metrics.Gauge("abc"))
g.Set(3.2)
g.Add(4.2)
g.Add(5.2)


type gauge struct {
    ig  metrics.IGauge
    mu  sync.Mutex
    val float64
}

func newGauge(ig metrics.IGauge) *gauge {
    return &gauge{ig: ig}
}

func (g *gauge) Set(v float64) {
    g.mu.Lock()
    g.val = v
    g.ig.Set(g.val)
    g.mu.Unlock()
}

func (g *gauge) Add(v float64) {
    g.mu.Lock()
    g.val += v
    g.ig.Set(g.val)
    g.mu.Unlock()
}
```

#### 2.1.1.3 Timer

Timer 是一种特殊的 Gauge, 可以根据一个操作的开始时间和结束时间统计某个操作的耗时情况。
例如你要监控某个操作耗费的时间，则可以定义并上报指标名称为 "operation.time.cost" 的 Timer：

```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

_ = metrics.Timer("operation.time.cost")
// The operation took 2s.
timeCost := 2 * time.Second
metrics.RecordTimer("operation.time.cost", timeCost)
```

#### 2.1.1.4 Histogram

Histogram 用于统计某类指标的分布情况，如最大，最小，平均值，标准差，以及各种分位数，例如 90%，95% 的数据分布在某个范围内。
创建 Histogram 时需要给定预先划分好的 buckets，上报 Histogram 时将收集到的样本点放入到对应的 bucket 中。
例如你要监控请求大小的分布情况，则可以根据实际情况创建好 buckets 后，把收集到的样本放入到指标名为 "request.size" 的 Histogram：

```go
buckets := metrics.NewValueBounds(1, 2, 5, 10)
metrics.AddSample("request.size", buckets, 3)
metrics.AddSample("request.size", buckets, 7)
```

## 2.2 多维监控指标

多维监控指标通常要结合后端的监控平台来对数据做不同维度的计算和展示。
多维监控指标指标由指标名字、指标维度信息和多个单维监控指标三部分组成。
例如你想要根据应用程序名和服务名等不同维度的对监控服务所接收到的请求，则可以创建如下的多维监控指标：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/metrics"
)

if err := metrics.ReportMultiDimensionMetricsX("request",
    []*metrics.Dimension{
        {
            Name:  "app",
            Value: "trpc-go",
        },
        {
            Name:  "server",
            Value: "example",
        },
        {
            Name:  "service",
            Value: "hello",
        },
    },
    []*metrics.Metrics{
        metrics.NewMetrics("request-count", 1, metrics.PolicySUM),
        metrics.NewMetrics("request-cost", float64(time.Second), metrics.PolicyAVG),
        metrics.NewMetrics("request-size", 30, metrics.PolicyHistogram),
    }); err != nil {
        log.Infof("reporting request multi dimension metrics failed: %v", err)
}
```

# 3. 上报外部监控系统

监控指标需要上报到各种监控系统，这些监控系统可以是公司内部的监控平台，也可以是外部开源社区的 Prometheus 等。
为此 metrics 包提供了一个通用的 `Sink` 接口：

```go
// Sink defines the interface an external monitor system should provide.
type Sink interface {
    // Name returns the name of the monitor system.
    Name() string
    // Report reports a record to monitor system.
    Report(rec Record, opts ...Option) error
}
```

上报不同监控系统时只需要实现该 `Sink` 接口并将实现注册到 metrics 包即可。

以将监控指标上报到控制台为例，通常需要以下三步。

1. 创建一个 `ConsoleSink` 结构体实现 `Sink` 接口。metrics 包已经内置实现了 `ConsoleSink`，可以通过 `metrics.NewConsoleSink()` 直接创建。

2. 将 `ConsoleSink` 注册到 metrics 包。

3. 创建各种监控指标并上报。

如下代码片段展示了上述三步：

```go
import "git.code.oa.com/trpc-go/trpc-go/log"

// 1. 创建一个 `ConsoleSink` 结构体实现 `Sink` 接口。
s := metrics.NewConsoleSink()

// 2. 将 `ConsoleSink` 注册到 metrics 包。
metrics.RegisterMetricsSink(s)

// 3. 创建各种监控指标并上报。
_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```

# 4. FAQ

这里列出了一些框架自身监控和监控平台相关的问题。

## 4.1 框架自身监控问题

### Q1 - 如何查看框架统计上报监控数据？

- tRPC-Go 框架默认会每天上报框架及插件版本信息到管理后台以供数据统计分析。
- 如果没有上报数据需要确认以下几点：
  - tRPC-Go 框架必须在 v0.1.0 以上。
  - 业务服务必须导入 `"https://git.woa.com/trpc-go/trpc-metrics-runtime"` 这个包，并且这个包的版本在 v0.1.3 以上。

    ```go
    import (
        // _ "git.code.oa.com/trpc-go/trpc-metrics-runtime" // 非 v2 版本 import git.code.oa.com，且必须配置 https://goproxy.woa.com/
        _ "trpc.tech/trpc-go/trpc-metrics-runtime/v2" // v2 版本请使用 trpc.tech 域名
    )
    ```

  - 查看 go.mod 版本号是否正确：

    ```text
    require (
        trpc.tech/trpc-go/trpc-go/v2 v2.0.0-beta
        trpc.tech/trpc-go/trpc-metrics-runtime/v2 v2.0.0-beta
    )
    ```

  - 在 [这里](http://show.wsd.com/show3.htm?viewId=db_k8s.t_md_trpc&) 可以查看服务是否上报成功。

### Q2 - 框架上报的 runtime 基础监控各属性分别是什么意思？

在 [这里](https://git.woa.com/trpc-go/trpc-metrics-runtime) 有指标说明。
在 [这里](https://git.woa.com/trpc-go/trpc-go/blob/master/internal/report/metrics_reports.go) 有各个属性的注释说明。

## 4.2 007 监控平台问题

### Q1 - 主调监控和被调监控分别是什么意思？

主调监控指的是当前服务调用下游服务请求的 client 端的监控，从发起请求到收到下游回包的监控。
被调监控指的是当前服务接收上游服务请求的 server 端的监控，从收到请求到业务逻辑结束的监控。

### Q2 - 北极星平台的统计上报异常数与 007 监控数据对不上？

007 和北极星本来就是两个不同的系统，上报逻辑本身就不一样，北极星是在 selector 主调调用的时候上报的，007 有被调监控和主调监控。
另外，北极星的上报只有超时和 connect 失败才算错误，007 只要任何失败全部算错误，数据肯定是完全对不上的。

### Q3 - m007 插件初始化异常？

标准输出可以看到初始化的详细 log，新版的框架会收集标准库 log 的输出。异常情况下 007 配置 `debuglogOpen: true` 可以看到 [初始化的详细步骤与每次上报的详情](https://mk.woa.com/note/1067)，注意线上不要开启。

报错：setup plugin metrics-m007 timeout
原因：007 SDK 拉取远程配置依赖北极星，一般是北极星北极星 SDK 拉取 IP 超时。需要升级插件到最新版本，支持北极星默认埋点推荐。删除 polarisAddrs、polarisProto 配置项。

报错：trpc-metrics-m007:pcgmonitor.Setup error:init error
原因：一般是机器问题，无法连接 attaagent(未启动或者机器 fd 数过多无法连接)，attaapi 错误码意义见 [这里](https://git.woa.com/atta/attaapi_go/blob/master/attaapi_go.go)。

1. 非 123 环境的话业务安装启动 atta agent，见 [这里](http://km.oa.com/articles/show/447456?kmref=search&from_page=1&no=1)。
2. 123 环境的话，要 atta 测来看了，只能提供机器信息拉群解决了。相关人：DataPlatform_helper&运维。可以切换容器快速解决下。

不支持 DevCloud 环境，不建议折腾，需要启动服务，临时删除 007 的相关配置即可。

想解决阅读 [pcgmonitor.go](https://git.woa.com/pcgmonitor/trpc_report_api_go/blob/master/pcgmonitor.go) 中的 startup 函数，了解相关的依赖，自行解决网络策略问题。主要有 3 点依赖：

- attaagent
- 北极星
- 007 远程服务，路由 64939329:131073

插件启动较慢，还有一个原因是 CPU 核数太小，比如只有 1 核，这种情况也是大概率失败的，需要把核数调大。

### Q4 - TcpServerTransportReadEOF 监控量较大是什么原因？

TcpServerTransportReadEOF 这个代表当前 server 接收到上游 client 的 close connection 信号，当前服务是正常的，是上游调用方的问题。
对于 trpc 来说，client->server 之间的连接都是长连接的，正常情况下连接会一直保持。不过 client 端默认有 50s 的链接空闲时间，如果请求量较小，一个连接超过 50s 都没有数据，client 端就会自动关闭连接，这种情况也是正常的。
其他情况，需要详细定位一下 client 为什么会频繁的主动关闭连接了，大概率是 client 有 bug，比如使用了短连接方式，或者 client 端大量超时马上关闭连接。

### Q5 - 007 上面查看不到监控项？

service name 格式必须是 trpc.app.server.service 点号分隔开的四段字符串。
如果确实不符合规范，而且是 database 调用的话，可以将 `NewClientProxy("trpc.app.server.service")` 和名字服务 `WithTarget("polaris://servicename")` 分开。`NewClientProxy` 参数必须自己定义并且符合规范，`WithTarget` 填实际服务名即可。
如果是其他情况，那么就必须用户自己通过代码定义 filter 设置 app server service method。被调上报时，如果自身 service name 或者上游调用方不符合规范，则定义 server filter：

```go
func ServerFilter(ctx, req, next) (rsp, err) {
    msg := trpc.Message(ctx)
    msg.WithCallerApp("app") // caller 是上游调用方
    msg.WithCallerServer("server")
    msg.WithCallerService("service")
    msg.WithCallerMethod("method")
    msg.WithCalleeApp("app") // callee 是自身
    msg.WithCalleeServer("server")
    msg.WithCalleeService("service")
    msg.WithCalleeMethod("method")
}
```

主调上报时，如果自身或者被调 service name 不符合规范，则定义 client filter：

```go
func ClientFilter(ctx, req, rsp, next) err {
    msg := trpc.Message(ctx)
    msg.WithCallerApp("app") // caller 是自身
    msg.WithCallerServer("server")
    msg.WithCallerService("service")
    msg.WithCallerMethod("method")
    msg.WithCalleeApp("app") // callee 是下游被调方
    msg.WithCalleeServer("server")
    msg.WithCalleeService("service")
    msg.WithCalleeMethod("method")
}
```

并且需要将 filter 配置到 m007 之前：

```yaml
server:
  service:
    name: xx  # 你自己的服务名
    filter:
      - xx    # 你前面自己定义的 filter name
      - m007
client:
  service:
    name: xxx # 被调服务名
    filter:
      - xx    # 你前面自己定义的 filter name
      - m007
```

### Q6 - 007 主调 upserver upservice 是什么意思？

007 监控上报的主调信息需要上游调用方通过协议字段把自己的服务名带下来，这里的 upserver upservice 说明框架获取不到上游服务名了，这是默认的值。
trpc.http.upserver.upservice 比如这个，说明 Web 调用的时候没有把自己信息 trpc-caller 填到 http header 里面，框架只知道是一个 http 请求，不知道是谁来调用了。具体字段看这里：[tRPC-Go 搭建泛 HTTP RPC 服务](https://iwiki.woa.com/p/490796254)。

## 4.3 天机阁使用问题

请见 [这里](https://km.woa.com/group/22063/articles/show/495740?ts=1639466804) 的说明。
