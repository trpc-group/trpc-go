[中文](./README.zh_CN.md)

# Metrics

Metrics are can be simply understood as a series of numerical measurements.
Different applications require different measurements.
For example, For a web server, it might be request times; for a database, it might be the number of active connections or active queries, and so on.

Metrics play a crucial role in understanding why your application works in a certain way.
Suppose you are running a web application and find it running slowly.
To understand what happened to your application, you need some information.
For example, the application may slow down when the number of requests is high.
If you have request count metrics, you can determine the cause and increase the number of servers to handle the load.

## Metric types

Metrics can be categorized into unidimensional and multidimensional based on their data dimensions.

### Unidimensional metrics

Unidimensional metrics consist of three parts: metric name, metric value, and metric aggregation policy.
A metric name uniquely identifies a unidimensional monitoring metric.
The metric aggregation policy describes how to aggregate metric values, such as summing, averaging, maximizing, and minimizing.
For example, if you want to monitor the average CPU load, you can define and report a unidimensional monitoring metric with the metric name "cpu.avg.load":

```golang
import (
    "trpc.group/trpc-go/trpc-go/log"
    "trpc.group/trpc-go/trpc-go/metrics"
)

if err := metrics.ReportSingleDimensionMetrics("cpu.avg.load", 70.0, metrics.PolicyAVG); err ! = nil {
    log.Infof("reporting cpu.avg.load metric failed: %v", err)
}
```

#### Common metrics

The metrics package provides several common types of unidimensional metrics such as counter, gauge, timer, and histogram, depending on the aggregation policy, the value range of the metric value, and the possible actions that can be taken on the metric value.
It is recommended to prioritize the use of these built-in metrics, and then customize other types of unidimensional metrics if they do not meet your needs.

##### Counter

Counter is used to count the cumulative amount of a certain type of metrics, it will save the cumulative value continuously from system startup.
It supports +1, -1, -n, +n operations on Counter.
For example, if you want to monitor the number of requests for a particular microservice, you can define a Counter with the metric name "request.num":

```go
import "trpc.group/trpc-go/trpc-go/metrics"

_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```

##### Gauge

Gauge is used to count the amount of moments of a certain type of metric.
For example, if you want to monitor the average CPU load, you can define and report a Gauge with the metric name "cpu.load.avg":

```go
import "trpc.group/trpc-go/trpc-go/metrics"

_ = metrics.Gauge("cpu.avg.load")
metrics.SetGauge("cpu.avg.load", 0.75)
```

##### Timer

Timer is a special type of Gauge, which can count the time consumed by an operation according to its start time and end time.
For example, if you want to monitor the time spent on an operation, you can define and report a timer with the name "operation.time.cost":

```go
import "trpc.group/trpc-go/trpc-go/metrics"

_ = metrics.Timer("operation.time.cost")
// The operation took 2s.
timeCost := 2 * time.Second
metrics.RecordTimer("operation.time.cost", timeCost)
```

##### Histogram

Histograms are used to count the distribution of certain types of metrics, such as maximum, minimum, mean, standard deviation, and various quartiles, e.g. 90%, 95% of the data is distributed within a certain range.
Histograms are created with pre-divided buckets, and the sample points collected are placed in the corresponding buckets when the Histogram is reported.
For example, if you want to monitor the distribution of request sizes, you can create buckets and put the collected samples into a histogram with the metric "request.size":

```golang
buckets := metrics.NewValueBounds(1, 2, 5, 10)
metrics.AddSample("request.size", buckets, 3)
metrics.AddSample("request.size", buckets, 7)
```

### Multidimensional metrics

Multidimensional metrics usually need to be combined with backend monitoring platforms to calculate and display data in different dimensions.
Multidimensional metrics consist of a metric name, metric dimension information, and multiple unidimensional metrics.
For example, if you want to monitor the requests received by a service based on different dimensions such as application name, service name, etc., you can create the following multidimensional metrics:

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

## Reporting to external monitoring systems

Metrics need to be reported to various monitoring systems, either internal to the company or external to the open source community, such as Prometheus.
The metrics package provides a generic `Sink` interface for this purpose:

```golang
// Sink defines the interface an external monitor system should provide.
type Sink interface {
// Name returns the name of the monitor system.
Name() string
// Name returns the name of the monitor system. Name() string // Report reports a record to monitor system.
Report(rec Record, opts .... Option) error
Option) error }
```

To integrate with different monitoring systems, you only need to implement the Sink interface and register the implementation to the metrics package.
For example, to report metrics to the console, the following three steps are usually required.

1. Create a `ConsoleSink` struct that implements the `Sink` interface.
   The metrics package already has a built-in implementation of `ConsoleSink`, which can be created directly via `metrics.NewConsoleSink()`

2. Register the `ConsoleSink` to the metrics package.

3. Create various metrics and report them.

The following code snippet demonstrates the above three steps:

```golang
import "trpc.group/trpc-go/trpc-go/log"

// 1. Create a `ConsoleSink` struct that implements the `Sink` interface.
s := metrics.NewConsoleSink()

// 2. Register the `ConsoleSink` to the metrics package.
metrics.RegisterMetricsSink(s)

// 3. Create various metrics and report them.
_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```