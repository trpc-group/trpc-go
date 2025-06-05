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

package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/metrics"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestMetricsConsoleSink() {
	sink := metrics.NewConsoleSink()
	metrics.RegisterMetricsSink(sink)
	s.T().Cleanup(func() {
		metrics.RegisterMetricsSink(noopSink{sink.Name()})
	})
	s.startServer(&TRPCService{})

	roundTripCostGauge := metrics.Gauge("request-cost")
	roundTripCostTimer := metrics.Timer("request-cost")
	roundTripCostHistogram := metrics.Histogram("request-cost", metrics.NewDurationBounds(
		time.Microsecond,
		10*time.Microsecond,
		50*time.Microsecond,
		100*time.Microsecond,
		200*time.Microsecond,
		500*time.Microsecond,
		1000*time.Microsecond,
	))
	requestCounter := metrics.Counter("request-count")
	payloadSizeHistogram := metrics.Histogram("request-size", metrics.NewValueBounds(1, 2, 5, 10))

	for _, req := range newSimpleRequests(s.T(), 10) {
		size := float64(len(req.GetPayload().GetBody()))
		payloadSizeHistogram.AddSample(size)
		requestCounter.Incr()
		roundTripCostTimer.Reset()
		startTime := time.Now()

		_, err := s.newTRPCClient().UnaryCall(trpc.BackgroundContext(), req)

		endTime := time.Since(startTime).Milliseconds()
		roundTripCostGauge.Set(float64(endTime))
		roundTripCostHistogram.AddSample(float64(endTime))
		roundTripCostTimer.Record()

		metrics.ReportSingleDimensionMetrics("max-request-size", size, metrics.PolicyMAX)
		metrics.Report(metrics.NewMultiDimensionMetricsX("metrics-test",
			[]*metrics.Dimension{
				{
					Name:  "module",
					Value: "trpc-go",
				},
				{
					Name:  "package",
					Value: "test",
				},
				{
					Name:  "file",
					Value: "metrics_test.go",
				},
			},
			[]*metrics.Metrics{
				metrics.NewMetrics("request-count", 1, metrics.PolicySUM),
				metrics.NewMetrics("request-cost", float64(endTime), metrics.PolicyAVG),
				metrics.NewMetrics("request-size", size, metrics.PolicyHistogram),
			}),
			metrics.WithMeta(map[string]interface{}{"module": 1, "package": 2, "file": 3}),
		)
		require.Nil(s.T(), err)
	}
}

type noopSink struct {
	name string
}

func (s noopSink) Name() string {
	return s.name
}

// Report reports a record.
func (s noopSink) Report(_ metrics.Record, _ ...metrics.Option) error {
	return nil
}

func newSimpleRequests(t *testing.T, n int) []*testpb.SimpleRequest {
	t.Helper()

	requests := make([]*testpb.SimpleRequest, 0, n)
	for size := 0; size < n; size++ {
		payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(size))
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, &testpb.SimpleRequest{
			ResponseType: testpb.PayloadType_COMPRESSABLE,
			ResponseSize: int32(size),
			Payload:      payload,
		})
	}
	return requests
}
