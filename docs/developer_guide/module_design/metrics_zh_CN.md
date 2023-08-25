## tRPC-Go 模块：metrics

### 背景

服务决不能裸奔，对于某些异常、错误、业务指标数据必须具备一定的监控能力。

通常开发者会使用 Logging 记录服务运行期间的操作流水、错误日志等，也会通过 Tracing 来分析整个调用链中的异常、错误。
除了这些以外，开发者还需要一些异常、错误、业务指标数据的监控上报能力，针对数据项本身的类型，我们通常需要支持累积量、时刻量、直方图、计时器等等。

根据这些指标数据维度上的不同，其实还可以进一步细分为一维、多维数据，多维数据多要结合后端的监控平台来对数据做不同维度的计算、展示。

Logging、Tracing、和 Metrics 的结合建立起一个更加全面的监控体系。
- Logging : 用于记录离散的事件。例如：应用程序的调试信息或错误信息。它是我们诊断问题的依据。
- Tracing : 用于记录请求范围内的信息。例如：一次远程方法调用的执行过程和耗时。可以用来排查系统性能问题
- Metrics : 用于记录可聚合的数据。例如：队列的当前深度可被定义为一个度量值，在元素入队或出队时被更新；HTTP 请求个数可被定义为一个计数器，新请求到来时进行累加。

### 原理

针对普通业务开发者，Package metrics 定义了常见粒度的监控指标，如 Counter、Gauge、Timer、Histogram。

针对框架开发者、插件开发者，Package metrics 支持单维、多维数据的上报，通过以下方法创建单维和多维数据：
- `rec := metrics.NewSingleDimensionMetrics(name, value, policy)`，创建单维数据
- `rec := metrics.NewMultiDimensionMetrics(dimensions, metrics)`，创建多维数据
  然后可以通过方法 `metrics.Report(rec)` 进行上报。

除此之外，`metrics` 提供 `Sink` interface 对接外部监控系统如外部开源的 Prometheus，对接不同监控系统时只需要实现`Sink`接口就可以。

下面是 metrics 的整体设计：

![metric architecture](/.resources/developer_guide/module_design/metrics/metric_architecture.png)


### 实现

#### metric 的类型

trpc-go metrics 有以下几种基本类型
- Counter: 主要用于统计某类指标的数量，它将保存从系统启动开始持续的累加值。支持对 Counter 进行 +1, -1, -n, +n 的操作。
- Histogram: 用于统一某类指标的分布情况，如最大，最小，平均值，标准差，以及各种分位数，例如 90%，95% 的数据分布在某个范围内。
- Gauge: 用于测量系统内任意数据的瞬态值
- Timer: 是一种特殊的 Gauge, 可以方便的统计某个业务接口的 QPS，RT 等数据。

#### 对接外部监控系统 Sink

Sink 定义了 Counter、Gauge、Timer、Histogram 类型数据的上报的通用接口：
```go
type Sink interface {
	// Name returns the name of the monitor system.
	Name() string
	// Report reports a record to monitor system.
	Report(rec Record, opts ...Option) error
}
```

例如，metrics 包内实现的 ConsoleSink 允许将 metric 上报到 Console。