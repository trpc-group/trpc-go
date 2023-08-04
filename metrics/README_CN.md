# tRPC-Go 监控功能及实现

针对框架开发者、插件开发者，Package metrics 支持单维、多维数据的上报，通过以下方法创建单维和多维数据：
- `rec := metrics.NewSingleDimensionMetrics(name, value, policy)`，创建单维数据
- `rec := metrics.NewMultiDimensionMetrics(dimensions, metrics)`，创建多维数据
然后可以通过方法 `metrics.Report(rec)` 进行上报。

针对普通业务开发者，Package metrics 定义了常见粒度的监控指标，如 Counter、Gauge、Timer、Histogram，
并在此基础上定义了与具体的外部监控系统对接的接口`type Sink interface`，对接具体的监控如公司 Monitor
或者外部开源的 Prometheus 等，对接不同监控系统时只需要实现`Sink`接口的 Plugin 就可以。

下面以业务开发者常用的 metrics 指标为例，说明下使用方法。为了使用方便，分别提供了两套常用方法：
- 第一种，首先通过"指标名字"来实例化一个指标变量（如处理过程中各异常分支总的失败量之和），通过该变量来上报，适用于同一个指标多处使用的情况；
- 第二种，直接使用"指标名字"来上报（如统计请求量之和），这种通常在方法入口处上报即可，适用于只使用一次的情况；

1. counter
- reqNumCounter := metrics.Counter("req.num")
  reqNumCounter.Incr()
- metrics.IncrCounter("req.num", 1)

2. gauge
- cpuAvgLoad := metrics.Gauge("cpu.avgload")
  cpuAvgLoad.Set(0.75)
- metrics.SetGauge("cpu.avgload", 0.75)

3. timer
- timeCostTimer := metrics.Timer("req.proc.timecost")
  timeCostTimer.Record()
- timeCostDuration := 2 * time.Second
  metrics.RecordTimer("req.proc.timecost", timeCostDuration)
                                                                                                             >
4. histogram
- buckets := metrics.NewDurationBounds(time.Second, 2 * time.Second, 5 * time.Second),
  timeCostDist := metrics.Histogram("timecost.distribution", buckets)
  timeCostDist.AddSample(float64(3 * time.Second))
- metrics.AddSample("timecost.distribution", buckets, float64(3 * time.Second))
