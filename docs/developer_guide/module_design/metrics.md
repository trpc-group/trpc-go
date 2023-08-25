## tRPC-Go module: metrics
### Introduction

Are your microservices naked and afraid? 
If your microservices run naked, you will get little context about it, which is not conducive to locating online problems and understanding your own services.
You may need some monitoring capabilities for certain exceptions, errors, and business metrics data.

Usually developers use Logging to record the operation flow and error logs during service operation, and use Tracing to analyze the exceptions and errors in the whole call chain.
In addition to these, developers also need the ability to monitor and report exceptions, errors, and business metrics data, and for the type of data items themselves, a RPC framework usually need to support cumulative, momentary, histogram, timer, and so on.

According to the difference in these metric dimensions, in fact, those metrics can be further subdivided into unidimensional data, and multidimensional data combined with the back-end monitoring platform to do different dimensions of the data calculation, display.

The Combination of Logging, Tracing, and Metrics establish a more comprehensive monitoring system.
- Logging: Used to record discrete events. For example, debugging information or error information of the application. It is the basis for us to diagnose the problem.
- Tracing: Used to record information about the scope of the request. For example, the execution process and time taken for a remote method call. Can be used to troubleshoot system performance problems
- Metrics: Used to record aggregated data. For example: the current depth of a queue can be defined as a metric that is updated when elements are queued in or out; the number of HTTP requests can be defined as a counter that is added up when a new request arrives.


### Principle

For general business developers, Package metrics provides common granularity monitoring metrics, such as Counter, Gauge, Timer, and Histogram.
For framework developers and plug-in developers, Package metrics supports the reporting of unidimensional and multidimensional data by creating unidimensional and multidimensional data with the following methods:

- `rec := metrics.NewSingleDimensionMetrics(name, value, policy)` creates unidimensional data.
- `rec := NewMultiDimensionMetrics(dimensions, metrics)` creates multidimensional data.

`metrics.Report(rec)` can report both types of data.
 

In addition, `metrics` provides a `Sink` interface for interacting with external monitoring system such as Prometheus. 
When interacting with different monitoring systems, you only need to implement the `Sink` interface.

The following is the overall design of metrics:

![metric architecture](/.resources/developer_guide/module_design/metrics/metric_architecture.png)


### Implementation


####  Types of metric

trpc-go metrics have the following metric types:

- Counter: mainly used to count the number of certain types of metrics, and it will store the cumulative values continuously from the system start-up. It supports +1, -1, -n, +n operations on Counter. 
- Histogram: used to standardize the distribution of certain types of indicators, such as maximum, minimum, mean, standard deviation, and various quartiles, e.g. 90%, 95% of the data are distributed in a certain range. 
- Gauge: used to measure the transient value of any data in the system.
- Timer: a special kind of Gauge that can easily count the QPS, RT, etc. of a business interface.

#### Sink: the interface for external monitoring system 

Sink defines a common interface for reporting all kinds of Record including Counter, Gauge, Timer, and Histogram:

```go
type Sink interface {
	// Name returns the name of the monitor system.
	Name() string
	// Report reports a record to monitor system.
	Report(rec Record, opts ...Option) error
}
```

For example, the ConsoleSink implementation within the metrics package allows metrics to be reported to the Console.