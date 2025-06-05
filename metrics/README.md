English | [中文](README.zh_CN.md)

# 1. Metrics

Metrics are can be simply understood as a series of numerical measurements.
Different applications require different measurements.
For example, For a web server, it might be request times; for a database, it might be the number of active connections or active queries, and so on.

Metrics play a crucial role in understanding why your application works in a certain way.
Suppose you are running a web application and find it running slowly.
To understand what happened to your application, you need some information.
For example, the application may slow down when the number of requests is high.
If you have request count metrics, you can determine the cause and increase the number of servers to handle the load.

# 2. Metric types

Metrics can be categorized into unidimensional and multidimensional based on their data dimensions.

## 2.1 Unidimensional metrics

Unidimensional metrics consist of three parts: metric name, metric value, and metric aggregation policy.
A metric name uniquely identifies a unidimensional monitoring metric.
The metric aggregation policy describes how to aggregate metric values, such as summing, averaging, maximizing, and minimizing.
For example, if you want to monitor the average CPU load, you can define and report a unidimensional monitoring metric with the metric name "cpu.avg.load":

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/metrics"
)

if err := metrics.ReportSingleDimensionMetrics("cpu.avg.load", 70.0, metrics.PolicyAVG); err ! = nil {
    log.Infof("reporting cpu.avg.load metric failed: %v", err)
}
```

### 2.1.1 Common metrics

The metrics package provides several common types of unidimensional metrics such as counter, gauge, timer, and histogram, depending on the aggregation policy, the value range of the metric value, and the possible actions that can be taken on the metric value.
It is recommended to prioritize the use of these built-in metrics, and then customize other types of unidimensional metrics if they do not meet your needs.

#### 2.1.1.1 Counter

Counter is used to count the cumulative amount of a certain type of metrics, it will save the cumulative value continuously from system startup.
It supports +1, -1, -n, +n operations on Counter. Note that the Counter defined here may be different from other monitoring systems, for example, the value of [Counter in Prometheus](https://prometheus.io/docs/concepts/metric_types/#counter) can only be monotonically increasing.
If you perform a "-1" operation on the Counter using [trpc-metrics-prometheus](https://git.woa.com/trpc-go/trpc-metrics-prometheus) in this case, it may result in an error.
In this case, it is recommended to use Gauge, or use two Counters, and finally subtract the values of the two Counters.
For example, if you want to monitor the number of requests for a particular microservice, you can define a Counter with the metric name "request.num":

```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```

#### 2.1.1.2 Gauge

Gauge is used to count the amount of moments of a certain type of metric.
For example, if you want to monitor the average CPU load, you can define and report a Gauge with the metric name "cpu.load.avg":

```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

_ = metrics.Gauge("cpu.avg.load")
metrics.SetGauge("cpu.avg.load", 0.75)
```

Gauge can only set values, but cannot accumulate values.
If you need to accumulate values, you can encapsulate a layer based on Gauge:

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

Timer is a special type of Gauge, which can count the time consumed by an operation according to its start time and end time.
For example, if you want to monitor the time spent on an operation, you can define and report a timer with the name "operation.time.cost":

```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

_ = metrics.Timer("operation.time.cost")
// The operation took 2s.
timeCost := 2 * time.Second
metrics.RecordTimer("operation.time.cost", timeCost)
```

#### 2.1.1.4 Histogram

Histograms are used to count the distribution of certain types of metrics, such as maximum, minimum, mean, standard deviation, and various quartiles, e.g. 90%, 95% of the data is distributed within a certain range.
Histograms are created with pre-divided buckets, and the sample points collected are placed in the corresponding buckets when the Histogram is reported.
For example, if you want to monitor the distribution of request sizes, you can create buckets and put the collected samples into a histogram with the metric "request.size":

```go
buckets := metrics.NewValueBounds(1, 2, 5, 10)
metrics.AddSample("request.size", buckets, 3)
metrics.AddSample("request.size", buckets, 7)
```

## 2.2 Multidimensional metrics

Multidimensional metrics usually need to be combined with backend monitoring platforms to calculate and display data in different dimensions.
Multidimensional metrics consist of a metric name, metric dimension information, and multiple unidimensional metrics.
For example, if you want to monitor the requests received by a service based on different dimensions such as application name, service name, etc., you can create the following multidimensional metrics:

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

# 3. Reporting to external monitoring systems

Metrics need to be reported to various monitoring systems, either internal to the company or external to the open source community, such as Prometheus.
The metrics package provides a generic `Sink` interface for this purpose:

```go
// Sink defines the interface an external monitor system should provide.
type Sink interface {
    // Name returns the name of the monitor system.
    Name() string
    // Name returns the name of the monitor system. Name() string // Report reports a record to monitor system.
    Report(rec Record, opts .... Option) error
}
```

To integrate with different monitoring systems, you only need to implement the Sink interface and register the implementation to the metrics package.
For example, to report metrics to the console, the following three steps are usually required.

1. Create a `ConsoleSink` struct that implements the `Sink` interface. The metrics package already has a built-in implementation of `ConsoleSink`, which can be created directly via `metrics.NewConsoleSink()`

2. Register the `ConsoleSink` to the metrics package.

3. Create various metrics and report them.

The following code snippet demonstrates the above three steps:

```go
import "git.code.oa.com/trpc-go/trpc-go/log"

// 1. Create a `ConsoleSink` struct that implements the `Sink` interface.
s := metrics.NewConsoleSink()

// 2. Register the `ConsoleSink` to the metrics package.
metrics.RegisterMetricsSink(s)

// 3. Create various metrics and report them.
_ = metrics.Counter("request.num")
metrics.IncrCounter("request.num", 30)
```

# 4. FAQ

Here are some issues related to the framework's own monitoring and the monitoring platform.

## 4.1 Issues with the framework's own monitoring

### Q1 - How to view the framework's statistics and reported monitoring data?

- The tRPC-Go framework will default to reporting the framework and plugin version information to the management backend daily for data statistics and analysis.
- If there is no reported data, the following points need to be confirmed:
  - The tRPC-Go framework must be above version v0.1.0.
  - The business service must import the package `"https://git.woa.com/trpc-go/trpc-metrics-runtime"`, and the version of this package must be above v0.1.3.

    ```go
    import (
        // _ "git.code.oa.com/trpc-go/trpc-metrics-runtime" // For non-v2 versions, import git.code.oa.com, and it must be configured with https://goproxy.woa.com/.
        _ "trpc.tech/trpc-go/trpc-metrics-runtime/v2" // For v2 versions, please use the domain name trpc.tech.
    )
    ```

  - Check if the version number in go.mod is correct:

    ```text
    require (
        trpc.tech/trpc-go/trpc-go/v2 v2.0.0-beta
        trpc.tech/trpc-go/trpc-metrics-runtime/v2 v2.0.0-beta
    )
    ```

  - You can check whether the service has successfully reported at [this link](http://show.wsd.com/show3.htm?viewId=db_k8s.t_md_trpc&).

### Q2 - What are the meanings of each attribute in the runtime basic monitoring reported by the framework?

The metric descriptions can be found [here](https://git.woa.com/trpc-go/trpc-metrics-runtime).
Annotations for each attribute can be found [here](https://git.woa.com/trpc-go/trpc-go/blob/master/internal/report/metrics_reports.go).

## 4.2 007 Monitoring Platform Issues

### Q1 - What do primary call monitoring and callee monitoring mean respectively?

Primary call monitoring refers to the monitoring of the client side of the current service calling downstream services, from initiating the request to receiving the downstream response packet.
Callee monitoring refers to the monitoring of the server side of the current service receiving upstream service requests, from receiving the request to the end of business logic.

### Q2 - The statistical report exception count on the Polaris platform does not match the 007 monitoring data?

007 and Polaris are two different systems with inherently different reporting logics. Polaris reports during the selector's primary call invocation, while 007 includes both callee monitoring and primary call monitoring.
Additionally, Polaris only considers errors as timeouts and connection failures, whereas 007 counts any failure as an error, so the data will definitely not match completely.

### Q3 - m007 plugin initialization exception?

Standard output can show detailed logs of initialization. The new version of the framework will collect the output of standard library logs. In case of an exception, setting `debuglogOpen: true` in the 007 configuration will allow you to see [detailed steps of initialization and details of each report](https://mk.woa.com/note/1067). Note that this should not be enabled in production.

Error: setup plugin metrics-m007 timeout
Reason: The 007 SDK pulls remote configurations which depend on Polaris, typically due to the Polaris SDK timing out when fetching IP addresses. It is necessary to upgrade the plugin to the latest version to support Polaris' default tracking recommendations. Remove the polarisAddrs and polarisProto configuration items.

Error: trpc-metrics-m007:pcgmonitor.Setup error:init error
Reason: Generally, it is a machine issue, unable to connect to attaagent (either not started or the machine has too many file descriptors and cannot connect). For the meaning of attaapi error codes, see [here](https://git.woa.com/atta/attaapi_go/blob/master/attaapi_go.go).

1. For non-123 environments, see [here](http://km.oa.com/articles/show/447456?kmref=search&from_page=1&no=1) for instructions on installing and starting the atta agent for your business.
2. In a 123 environment, it requires atta testing to check. Only machine information can be provided to resolve the issue through a group chat. Relevant personnel: DataPlatform_helper & Operations. Switching containers can provide a quick solution.

Not supported in DevCloud environment, not recommended to tamper with it. To start the service, temporarily delete the relevant configurations of 007.

To solve the issue, read the `startup` function in [pcgmonitor.go](https://git.woa.com/pcgmonitor/trpc_report_api_go/blob/master/pcgmonitor.go) to understand the related dependencies and solve the network policy issue on your own. There are mainly three dependencies:

- attaagent
- polaris
- 007 Remote service, route 64939329:131073

Another reason for the slow startup of the plugin is that there are too few CPU cores, such as only 1 core. In this case, failure is also highly likely, and the number of cores needs to be increased.

### Q4 - What is the reason for the large monitoring volume of TcpServerTransportReadEOF?

TcpServerTransportReadEOF indicates that the current server has received a close connection signal from the upstream client. The current service is normal; it is an issue with the upstream caller.
For trpc, the connections between client and server are persistent by default, and under normal circumstances, the connection will be maintained continuously. However, the client has a default idle time of 50 seconds for the connection. If the request volume is small and a connection remains idle without data for more than 50 seconds, the client will automatically close the connection, which is also normal.
In other cases, it is necessary to thoroughly locate why the client frequently initiates closing the connection. It is highly probable that there is a bug in the client, such as using a short connection method, or the client side closes the connection immediately after a large number of timeouts.

### Q5 - Can't see the monitoring items on 007?

The format of the service name must be trpc.app.server.service, a four-part string separated by dots.
If it indeed does not meet the specification, and it is a database call, you can separate `NewClientProxy("trpc.app.server.service")` from the naming service `WithTarget("polaris://servicename")`. The parameter for `NewClientProxy` must be defined by yourself and comply with the specification, while `WithTarget` should be filled with the actual service name.
In other cases, users must define the filter through code to set the app server service method. When reporting a downstream call, if the own service name or the upstream caller does not meet the specification, then define a server filter:

```go
func ServerFilter(ctx, req, next) (rsp, err) {
    msg := trpc.Message(ctx)
    msg.WithCallerApp("app") // Caller is the upstream caller.
    msg.WithCallerServer("server")
    msg.WithCallerService("service")
    msg.WithCallerMethod("method")
    msg.WithCalleeApp("app") // Callee is the self (the called party).
    msg.WithCalleeServer("server")
    msg.WithCalleeService("service")
    msg.WithCalleeMethod("method")
}
```

When reporting as the caller, if the service name of the caller or callee does not meet the specifications, then define a client filter:

```go
func ClientFilter(ctx, req, rsp, next) err {
    msg := trpc.Message(ctx)
    msg.WithCallerApp("app") // Caller is the self (the calling party).
    msg.WithCallerServer("server")
    msg.WithCallerService("service")
    msg.WithCallerMethod("method")
    msg.WithCalleeApp("app") // Callee is the downstream callee.
    msg.WithCalleeServer("server")
    msg.WithCalleeService("service")
    msg.WithCalleeMethod("method")
}
```

And the filter needs to be configured before m007:

```yaml
server:
  service:
    name: xx  # Your own service name.
    filter:
      - xx    # The filter name you defined earlier.
      - m007
client:
  service:
    name: xxx # The callee service name.
    filter:
      - xx    # The filter name you defined earlier.
      - m007
```

### Q6 - What does "007 main call upserver upservice" mean?

The '007' monitoring report requires the upstream caller to bring down their service name through the protocol field. Here, 'upserver upservice' indicates that the framework cannot obtain the upstream service name, which is the default value.
For example, 'trpc.http.upserver.upservice' indicates that during a Web call, the information 'trpc-caller' was not filled into the HTTP header. The framework only knows it's an HTTP request and doesn't know who made the call. For specific fields, see here: [Building a Generic HTTP RPC Service with tRPC-Go](https://iwiki.woa.com/p/490796254).

## 4.3 Questions about the use of Tianji Pavilion

Please refer to the instructions [here](https://km.woa.com/group/22063/articles/show/495740?ts=1639466804).
