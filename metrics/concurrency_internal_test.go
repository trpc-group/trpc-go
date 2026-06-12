//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package metrics

import (
	"sync"
	"testing"
)

type concurrentSink struct {
	name string
}

func (s concurrentSink) Name() string {
	return s.name
}

func (s concurrentSink) Report(Record, ...Option) error {
	return nil
}

func isolateMetricsForTesting(t *testing.T) {
	t.Helper()

	metricsSinksMutex.Lock()
	oldSinks := metricsSinks
	metricsSinks = map[string]Sink{}
	metricsSinksMutex.Unlock()

	countersMutex.Lock()
	oldCounters := counters
	counters = map[string]ICounter{}
	countersMutex.Unlock()

	gaugesMutex.Lock()
	oldGauges := gauges
	gauges = map[string]IGauge{}
	gaugesMutex.Unlock()

	timersMutex.Lock()
	oldTimers := timers
	timers = map[string]ITimer{}
	timersMutex.Unlock()

	histogramsMutex.Lock()
	oldHistograms := histograms
	histograms = map[string]IHistogram{}
	histogramsMutex.Unlock()

	t.Cleanup(func() {
		metricsSinksMutex.Lock()
		metricsSinks = oldSinks
		metricsSinksMutex.Unlock()

		countersMutex.Lock()
		counters = oldCounters
		countersMutex.Unlock()

		gaugesMutex.Lock()
		gauges = oldGauges
		gaugesMutex.Unlock()

		timersMutex.Lock()
		timers = oldTimers
		timersMutex.Unlock()

		histogramsMutex.Lock()
		histograms = oldHistograms
		histogramsMutex.Unlock()
	})
}

func TestConcurrentSinkRegistrationAndReporting(t *testing.T) {
	isolateMetricsForTesting(t)

	const goroutines = 8
	const iterations = 100

	name := t.Name()
	Histogram(name, NewValueBounds(1, 10, 100))

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				RegisterMetricsSink(concurrentSink{name: name + "-sink"})
				RegisterMetricsSink(concurrentSink{name: name + "-sink-alt"})
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				Counter(name).IncrBy(float64(i + j))
				Gauge(name).Set(float64(j))
				AddSample(name, NewValueBounds(1, 10, 100), float64(j%100))
				_ = Report(NewSingleDimensionMetrics(name, 1, PolicySUM))
			}
		}()
	}
	wg.Wait()
}
