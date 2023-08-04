# tRPC-Go Metrics

For framework/plugin developers, package metrics provides single/multiple dimension(s) report. You can create them in
the following methods:
- `rec := metrics.NewSingleDimensionMetrics(name, value, policy)`, creates a single dimension metric.
- `rec := metrics.NewMultiDimensionMetrics(dimensions, metrics)`, creates a multiple dimensions metric.
Then, report them by `metrics.Report(rec)`.

For business developers, package metrics provides some common used metrics, such as Counter, Gauge, Timer and Histogram.
An interface `Sink` is used to communicate with external monitor systems, such as Tencent Monitor or Prometheus. The
external monitor just provides a Plugin which implements Sink.

The following uses business metrics as an example to illustrate how to use them. For the convenience, we provide two
sets of methods.
- The first one instantiate a metric by name. You may use it anywhere to report, such as the sum of total number of
  failures.
- The second one use metric name to report directly. It's suitable for only one use. For example, report total number of
  requests at the beginning.

1. counter
   - `reqNumCounter := metrics.Counter("req.num")`  
     `reqNumCounter.Incr()`
   - `metrics.IncrCounter("req.num", 1)`
2. gauge
   - `cpuAvgLoad := metrics.Gauge("cpu.avgload")`  
     `cpuAvgLoad.Set(0.75)`
   - `metrics.SetGauge("cpu.avgload", 0.75)`
3. timer
   - `timeCostTimer := metrics.Timer("req.proc.timecost")`  
     `timeCostTimer.Record()`
   - `timeCostDuration := 2 * time.Second`  
     `metrics.RecordTimer("req.proc.timecost", timeCostDuration)`
4. histogram
   - `buckets := metrics.NewDurationBounds(time.Second, 2 * time.Second, 5 * time.Second)`  
     `timeCostDist := metrics.Histogram("timecost.distribution", buckets)`  
     `timeCostDist.AddSample(float64(3 * time.Second))`
   - `metrics.AddSample("timecost.distribution", buckets, float64(3 * time.Second))`
