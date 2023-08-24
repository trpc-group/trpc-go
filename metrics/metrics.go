// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package metrics defines some common metrics, such as Counter, IGauge, ITimer and IHistogram.
// The method MetricsSink is used to adapt to external monitor systems, such as monitors in our
// company or open source prometheus.
//
// For convenience, we provide two sorts of methods:
//
//  1. counter
//     - reqNumCounter := metrics.Counter("req.num")
//     reqNumCounter.Incr()
//     - metrics.IncrCounter("req.num", 1)
//
//  2. gauge
//     - cpuAvgLoad := metrics.Gauge("cpu.avgload")
//     cpuAvgLoad.Set(0.75)
//     - metrics.SetGauge("cpu.avgload", 0.75)
//
//  3. timer
//     - timeCostTimer := metrics.Timer("req.proc.timecost")
//     timeCostTimer.Record()
//     - timeCostDuration := time.Millisecond * 2000
//     metrics.RecordTimer("req.proc.timecost", timeCostDuration)
//
//  4. histogram
//     - buckets := metrics.NewDurationBounds(time.Second, time.Second*2, time.Second*5),
//     timeCostDist := metrics.Histogram("timecost.distribution", buckets)
//     timeCostDist.AddSample(float64(time.Second*3))
//     - metrics.AddSample("timecost.distribution", buckets, float64(time.Second*3))
package metrics

import (
	"fmt"
	"sync"
	"time"
)

var (
	// metricsSinks emits same metrics information to multi external system at the same time.
	metricsSinksMutex = sync.RWMutex{}
	metricsSinks      = map[string]Sink{}

	countersMutex = sync.RWMutex{}
	counters      = map[string]ICounter{}

	gaugesMutex = sync.RWMutex{}
	gauges      = map[string]IGauge{}

	timersMutex = sync.RWMutex{}
	timers      = map[string]ITimer{}

	histogramsMutex = sync.RWMutex{}
	histograms      = map[string]IHistogram{}
)

// RegisterMetricsSink registers a Sink.
func RegisterMetricsSink(sink Sink) {
	metricsSinksMutex.Lock()
	metricsSinks[sink.Name()] = sink
	metricsSinksMutex.Unlock()
	if histSink, ok := sink.(HistogramSink); ok {
		histogramsMutex.Lock()
		for _, hist := range histograms {
			if h, ok := hist.(*histogram); ok {
				histSink.Register(h.Name, HistogramOption{BucketBounds: h.Spec})
			}
		}
		histogramsMutex.Unlock()
	}
}

// GetMetricsSink gets a Sink by name
func GetMetricsSink(name string) (Sink, bool) {
	metricsSinksMutex.RLock()
	sink, ok := metricsSinks[name]
	metricsSinksMutex.RUnlock()
	return sink, ok
}

// Counter creates a named counter.
func Counter(name string) ICounter {
	countersMutex.RLock()
	c, ok := counters[name]
	countersMutex.RUnlock()
	if ok && c != nil {
		return c
	}

	countersMutex.Lock()
	c, ok = counters[name]
	if ok && c != nil {
		countersMutex.Unlock()
		return c
	}
	c = &counter{name: name}
	counters[name] = c
	countersMutex.Unlock()

	return c
}

// Gauge creates a named gauge.
func Gauge(name string) IGauge {
	gaugesMutex.RLock()
	c, ok := gauges[name]
	gaugesMutex.RUnlock()
	if ok && c != nil {
		return c
	}

	gaugesMutex.Lock()
	c, ok = gauges[name]
	if ok && c != nil {
		gaugesMutex.Unlock()
		return c
	}
	c = &gauge{name: name}
	gauges[name] = c
	gaugesMutex.Unlock()

	return c
}

// Timer creates a named timer.
func Timer(name string) ITimer {
	timersMutex.RLock()
	t, ok := timers[name]
	timersMutex.RUnlock()
	if ok && t != nil {
		return t
	}

	timersMutex.Lock()
	t, ok = timers[name]
	if ok && t != nil {
		timersMutex.Unlock()
		return t
	}
	t = &timer{name: name, start: time.Now()}
	timers[name] = t
	timersMutex.Unlock()

	return t
}

// NewTimer creates a named timer whose start is set to time.Now().
func NewTimer(name string) ITimer {
	t := Timer(name)
	t.Reset()
	return t
}

// Histogram creates a named histogram with buckets.
func Histogram(name string, buckets BucketBounds) IHistogram {
	h, ok := GetHistogram(name)
	if ok && h != nil {
		return h
	}

	// histogramsMutex 的锁范围不应该包括 metricsSinksMutex 的锁。
	histogramsMutex.Lock()
	h, ok = histograms[name]
	if ok && h != nil {
		histogramsMutex.Unlock()
		return h
	}
	h = newHistogram(name, buckets)
	histograms[name] = h
	histogramsMutex.Unlock()

	metricsSinksMutex.Lock()
	for _, sink := range metricsSinks {
		if histSink, ok := sink.(HistogramSink); ok {
			histSink.Register(name, HistogramOption{BucketBounds: buckets})
		}
	}
	metricsSinksMutex.Unlock()

	return h
}

// GetHistogram gets the histogram by key.
func GetHistogram(key string) (v IHistogram, ok bool) {
	histogramsMutex.RLock()
	h, ok := histograms[key]
	histogramsMutex.RUnlock()
	if !ok {
		return nil, false
	}

	hist, ok := h.(*histogram)
	if !ok {
		return nil, false
	}
	return hist, true
}

// IncrCounter increases counter key by value. Counters should accumulate values.
func IncrCounter(key string, value float64) {
	Counter(key).IncrBy(value)
}

// SetGauge sets gauge key to value. An IGauge retains the last set value.
func SetGauge(key string, value float64) {
	Gauge(key).Set(value)
}

// RecordTimer records timer named key with duration.
func RecordTimer(key string, duration time.Duration) {
	Timer(key).RecordDuration(duration)
}

// AddSample adds one sample key with value.
func AddSample(key string, buckets BucketBounds, value float64) {
	h := Histogram(key, buckets)
	h.AddSample(value)
}

// Report reports a multi-dimension record.
func Report(rec Record, opts ...Option) (err error) {
	var errs []error
	for _, sink := range metricsSinks {
		err = sink.Report(rec, opts...)
		if err != nil {
			errs = append(errs, fmt.Errorf("sink-%s error: %v", sink.Name(), err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("metrics sink error: %v", errs)
}
