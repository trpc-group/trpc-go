[TOC]

# Introduction

Metrics define some commonly used monitoring metrics, such as `counter`, `gauge`, `timer`, and `histogram`.

# Principles and Implementation

## Counter

Counter is just a counter, generally used to count cumulative amounts.

The current implementation of the counter uses floating-point numbers to count. In order to improve efficiency when concurrent operations are performed, we hope to replace Mutex operations with CAS operations. Therefore, the value is composed of an integer and a float64bits corresponding to the floating-point type. The actual value of the counter is the sum of the integer and the float part corresponding to the float64bits.

Note: In the future, similar statistics may be directly accumulated and reported in the sink of each plugin. No longer accumulate by yourself in the counter implementation. Because some monitoring systems need to report the original data to the backend, framework doing the data accumulation weakens the extensibility.

## Gauge

Gauge is an instantaneous value, generally used to count momentary amounts.

The current implementation of the gauge uses floating-point numbers to count. Because it is an instantaneous value, it can be updated directly using CAS operations.

Note: In the future, similar statistics may be directly accumulated and reported in the sink of each plugin. No longer accumulate by yourself in the counter implementation. Because some monitoring systems need to report the original data to the backend, framework doing the data accumulation weakens the extensibility.

## Timer

The timer calculates the time-consuming information of an operation based on the start time and end time of an operation. Taking `timer.RecordAction(fn)` as an example, it will record a time t before executing fn, and calculate the `time.Since(t)` after fn is executed to obtain the time-consuming information.

Note: In the future, similar statistics may be directly accumulated and reported in the sink of each plugin. No longer accumulate by yourself in the counter implementation. Because some monitoring systems need to report the original data to the backend, framework doing the data accumulation weakens the extensibility.

## Histogram

Histogram puts the collected sample points into the corresponding bucket according to the pre-divided buckets, so that it is convenient to view the number of sample points, average value, maximum value, minimum value, and sum of different intervals (upper and lower bounds of the bucket). The specific values of each interval are determined by the statistical strategy.

The histogram sink implementation will convert the histogram into platform-specific metrics for display.

Note: In the future, similar statistics may be directly accumulated and reported in the sink of each plugin. No longer accumulate by yourself in the counter implementation. Because some monitoring systems need to report the original data to the backend, framework doing the data accumulation weakens the extensibility.

# Usage Examples

The usage of counter, gauge, timer, and histogram is the same. In terms of specific monitoring platforms, the difference lies only in which monitoring plugin implementation package to import and how to configure it in the configuration file.

## Sample Code


```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

// example1: counter
metrics.Counter("total.req").Incr()
metrics.IncrCounter("total.req", 1)

// example2: guage
metrics.Gauge("cpu.avg.load").Set(0.7)
metrics.SetGauge("cpu.avg.load", 0.7)

// example3: timer
metrics.Timer("rpc.timecost").RecordAction(func() {
	// do something
})

t := metrics.Timer("rpc.timecost");

// reset timer & do something1
t = t.Reset()
...
t.Record()

// reset timer & do something2
t = t.Reset()
...
t.Record()

// example4: histogram
h := metrics.Histogram("req.timecost.hist", metrics.WithBuckets(...))
h.AddSample(time.Second)
h.AddSample(time.Second * 2)
h.AddSample(time.Second * 3)
...
```

## Configuration

During development, you can refer to the examples in the sample code. Depending on the different monitoring platforms in the operation system, you also need to import different packages for implementation and configure them correctly.

Please refer to documents in trpc-go/trpc-metrics-${name}.

