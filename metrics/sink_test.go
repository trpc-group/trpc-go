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

package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/metrics"
)

func TestConsoleSink(t *testing.T) {
	sink := metrics.NewConsoleSink()

	// 以 cpu 负载上报为例
	rec := metrics.NewSingleDimensionMetrics("cpu.load.top", 70.0, metrics.PolicyMAX)
	err := sink.Report(rec)
	assert.Nil(t, err, "report cpu.load.top")

	// method 1
	rec = metrics.NewSingleDimensionMetrics("cpu.load.avg", 70.0, metrics.PolicyAVG)
	err = sink.Report(rec)
	assert.Nil(t, err, "report cpu.load.avg")

	// method 2
	metrics.ReportSingleDimensionMetrics("cpu.load.top", 70.0, metrics.PolicyMAX)
	metrics.ReportSingleDimensionMetrics("cpu.load.avg", 70.0, metrics.PolicyAVG)

	// 以模调等多维上报为例
	dims := []*metrics.Dimension{
		{
			Name:  "srcService",
			Value: "HelloASvr",
		}, {
			Name:  "dstService",
			Value: "HelloBSvr",
		}, {
			Name:  "interface",
			Value: "HelloAction",
		},
	}

	indices := []*metrics.Metrics{
		metrics.NewMetrics("req.timecost", float64(time.Second), metrics.PolicyAVG),
		metrics.NewMetrics("req.count", float64(1), metrics.PolicySUM),
	}

	// method 1
	rec = metrics.NewMultiDimensionMetricsX("a", dims, indices)
	metrics.Report(rec)

	// method 2
	metrics.ReportMultiDimensionMetricsX("b", dims, indices)
}

func TestMetrics(t *testing.T) {
	m := metrics.NewMetrics("req.count", float64(1), metrics.PolicySUM)
	assert.Equal(t, m.Name(), "req.count")
	assert.Equal(t, m.Value(), float64(1))
	assert.Equal(t, m.Policy(), metrics.Policy(metrics.PolicySUM))

	var n *metrics.Metrics
	assert.Zero(t, n.Name(), "")
	assert.Zero(t, n.Value())
	assert.Zero(t, n.Policy())
}

func TestRecord(t *testing.T) {
	rec := metrics.NewSingleDimensionMetrics("cpu.load.avg", 70.0, metrics.PolicyAVG)
	d := rec.GetDimensions()
	assert.Equal(t, len(d), 0)

	m := rec.GetMetrics()
	assert.Equal(t, m[0].Name(), "cpu.load.avg")
	assert.Equal(t, m[0].Value(), 70.0)
	assert.Equal(t, m[0].Policy(), metrics.Policy(metrics.PolicyAVG))
}

func TestPolicy(t *testing.T) {
	r := metrics.NewSingleDimensionMetrics("cpu.load.top", 70.0, metrics.PolicyMIN)
	assert.Equal(t, r.GetMetrics()[0].Policy(), metrics.Policy(metrics.PolicyMIN))

	r = metrics.NewSingleDimensionMetrics("cpu.load.top", 70.0, metrics.PolicyMAX)
	assert.Equal(t, r.GetMetrics()[0].Policy(), metrics.Policy(metrics.PolicyMAX))

	r = metrics.NewSingleDimensionMetrics("cpu.load.top", 70.0, metrics.PolicyMID)
	assert.Equal(t, r.GetMetrics()[0].Policy(), metrics.Policy(metrics.PolicyMID))
}
