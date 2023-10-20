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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/metrics"
)

func TestWithMeta(t *testing.T) {
	want := metrics.Options{}
	monitors := map[string]interface{}{"req.total": 10001, "req.fail": 10002, "req.succ": 10003}
	opt := metrics.WithMeta(monitors)
	opt(&want)

	type args struct {
		meta map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want metrics.Options
	}{
		{"monitor", args{meta: monitors}, want},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.Options{}
			opt := metrics.WithMeta(tt.args.meta)
			opt(&got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithMeta() = %v, comp %v", got, tt.want)
			}
		})
	}
}

func TestGetOptions(t *testing.T) {
	assert.Nil(t, metrics.GetOptions().Meta)

	meta := map[string]interface{}{
		"req.total": 10000,
		"req.fail":  10001,
	}
	opts := metrics.Options{}
	o := metrics.WithMeta(meta)
	o(&opts)

	assert.Equal(t, opts.Meta, meta)
}
