[English](./README.md)

# Metrics

监控指标是可以简单地理解为一系列的数值测量。
不同的应用程序，需要测量的内容不同。
例如。对于 Web 服务器，可能是请求时间；对于数据库，可能是活动连接或活动查询的数量，等等。

监控指标在理解你的应用程序为什么以某种方式工作方面起着重要作用。
假设你正在运行一个 Web 应用程序，并发现它运行缓慢。
要了解你的应用程序发生了什么，你需要一些信息。
例如，当请求量很高时，应用程序可能会变慢。
如果你有请求计数监控指标，你可以确定原因并增加服务器数量以应对负载。

## 指标类型

根据根据监控指标数据维度上的不同，监控指标可以分为单维监控指标和多维监控指标。

### 单维监控指标

单维监控指标由指标名字，指标值，和指标聚合策略三部分组成。
指标名字唯一地标识了单维监控指标。
指标聚合策略描述了如何将指标值聚合在一起，如求和，取平均值，取最大值，和取最小值。
例如你要监控 CPU 的平均负载，则可以定义并上报指标名称为 "cpu.avg.load" 的单维监控指标：

```golang
import (
    "trpc.group/trpc-go/trpc-go/log"
    "trpc.group/trpc-go/trpc-go/metrics"
)

if err := metrics.ReportSingleDimensionMetrics("cpu.avg.load", 70.0, metrics.PolicyAVG); err != nil {
    log.Infof("reporting cpu.avg.load metric failed: %v", err)
}
```

#### 常用的监控指标

根据聚合策略，指标值的值域以及对指标值可能采取的操作，metrics 包提供了 counter、gauge、timer 和 histogram 等几种常见类型的单维监控指标。
建议优先考虑使用这几种内置的常见监控指标，如果不能满足需求，再自定义其他类型的单维监控指标。

##### Counter

Counter 用于统计某类指标的累积量，它将保存从系统启动开始持续的累加值。
支持对 Counter 进行 +1, -1, -n, +n 的操作。
例如你要监控某个微服务的请求数量，则可以定义一个指标名称为 "request.num" 的 Counter：

```go
import "trpc.group/trpc-go/trpc-go/metrics"

_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```

##### Gauge

Gauge 用于统计某类指标的时刻量。
例如你要监控 CPU 的平均负载，则可以定义并上报指标名称为 "cpu.load.avg" 的 Gauge：

```go
import "trpc.group/trpc-go/trpc-go/metrics"

_ = metrics.Gauge("cpu.avg.load")
metrics.SetGauge("cpu.avg.load", 0.75)
```

##### Timer

Timer 是一种特殊的 Gauge, 可以根据一个操作的开始时间、结束时间，统计某个操作的耗时情况。
例如你要监控某个操作耗费的时间，，则可以定义并上报指标名称为 "operation.time.cost" 的 Timer：

```go
import "trpc.group/trpc-go/trpc-go/metrics"

_ = metrics.Timer("operation.time.cost")
// The operation took 2s.
timeCost := 2 * time.Second
metrics.RecordTimer("operation.time.cost", timeCost)
```
##### Histogram

Histogram 用于统计某类指标的分布情况，如最大，最小，平均值，标准差，以及各种分位数，例如 90%，95% 的数据分布在某个范围内。
创建 Histogram 时需要给定预先划分好的 buckets，上报 Histogram 时将收集到的样本点放入到对应的 bucket 中。
例如你要监控请求大小的分布情况，则可以根据实际情况创建好 buckets 后，把收集到的样本放入到指标名为 "request.size" 的 Histogram：

```golang
buckets := metrics.NewValueBounds(1, 2, 5, 10)
metrics.AddSample("request.size", buckets, 3)
metrics.AddSample("request.size", buckets, 7)
```

### 多维监控指标

多维监控指标通常要结合后端的监控平台来对数据做不同维度的计算和展示。
多维监控指标指标由指标名字，指标维度信息，和多个单维监控指标三部分组成。
例如你想要根据应用程序名，服务名等不同维度的对监控服务所接收到的请求，则可以创建如下的多维监控指标：
```go
import (
    "trpc.group/trpc-go/trpc-go/log"
    "trpc.group/trpc-go/trpc-go/metrics"
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

## 上报外部监控系统

监控指标需要上报到各种监控系统，这些监控系统可以是公司内部的监控平台，也可以是外部开源社区的 Prometheus 等。
为此 metrics 包提供了一个通用的 `Sink` 接口：

```golang
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

1. 创建一个 `ConsoleSink` 结构体实现 `Sink` 接口。
   metrics 包已经内置实现了 `ConsoleSink`，可以通过 `metrics.NewConsoleSink()` 直接创建。

2. 将 `ConsoleSink` 注册到 metrics 包。

3. 创建各种监控指标并上报。

如下代码片段展示了上述三步：

```golang
import "trpc.group/trpc-go/trpc-go/log"

// 1. 创建一个 `ConsoleSink` 结构体实现 `Sink` 接口。
s := metrics.NewConsoleSink()

// 2. 将 `ConsoleSink` 注册到 metrics 包。
metrics.RegisterMetricsSink(s)

// 3. 创建各种监控指标并上报。
_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```