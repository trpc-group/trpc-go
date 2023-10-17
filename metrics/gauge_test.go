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

	"trpc.group/trpc-go/trpc-go/metrics"

	"github.com/stretchr/testify/assert"
)

func Test_gauge_Set(t *testing.T) {
	metrics.RegisterMetricsSink(&metrics.ConsoleSink{})
	type fields struct {
		name string
	}
	type args struct {
		v float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"gauge-cpu.avgload", fields{"cpu.avgload"}, args{0.75}},
		{"gauge-mem.avgload", fields{"mem.avgload"}, args{0.80}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := metrics.Gauge(tt.fields.name)
			assert.NotNil(t, g)
			g.Set(tt.args.v)
		})
	}
}
