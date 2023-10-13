// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package metrics

// ICounter is the interface that emits counter type metrics.
type ICounter interface {
	// Incr increments the counter by one.
	Incr()

	// IncrBy increments the counter by delta.
	IncrBy(delta float64)
}

// counter defines the counter. counter is report to each external Sink-able system.
type counter struct {
	name string
}

// Incr increases counter by one.
func (c *counter) Incr() {
	c.IncrBy(1)
}

// IncrBy increases counter by v and reports for each external Sink-able systems.
func (c *counter) IncrBy(v float64) {
	if len(metricsSinks) == 0 {
		return
	}
	rec := NewSingleDimensionMetrics(c.name, v, PolicySUM)
	for _, sink := range metricsSinks {
		sink.Report(rec)
	}
}
