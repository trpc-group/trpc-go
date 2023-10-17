//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package metrics

// IGauge is the interface that emits gauge metrics.
type IGauge interface {
	// Set sets the gauges absolute value.
	Set(value float64)
}

// gauge defines the gauge. gauge is reported to each external Sink-able system.
type gauge struct {
	name string
}

// Set updates the gauge value.
func (g *gauge) Set(v float64) {
	if len(metricsSinks) == 0 {
		return
	}
	r := NewSingleDimensionMetrics(g.name, v, PolicySET)
	for _, sink := range metricsSinks {
		sink.Report(r)
	}
}
