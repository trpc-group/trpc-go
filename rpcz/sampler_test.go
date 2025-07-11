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

package rpcz

import (
	"math"
	"reflect"
	"testing"
)

func Test_newSpanIDRatioSampler(t *testing.T) {
	const (
		maxFraction   = 1.0
		minFraction   = 0.0
		maxUpperBound = math.MaxInt64
	)
	type args struct {
		fraction float64
	}
	tests := []struct {
		name string
		args args
		want *spanIDRatioSampler
	}{
		{
			name: "fraction equals maxFraction",
			args: args{fraction: maxFraction},
			want: &spanIDRatioSampler{spanIDUpperBound: maxUpperBound},
		},
		{
			name: "fraction greater than maxFraction",
			args: args{fraction: 1.20221111},
			want: &spanIDRatioSampler{spanIDUpperBound: maxUpperBound},
		},
		{
			name: "fraction equals minFraction",
			args: args{fraction: minFraction},
			want: &spanIDRatioSampler{spanIDUpperBound: 0},
		},
		{
			name: "fraction less than minFraction",
			args: args{fraction: -0.20221111},
			want: &spanIDRatioSampler{spanIDUpperBound: 0},
		},
		{
			name: "fraction between minFraction and maxFraction",
			args: args{fraction: 0.20221111},
			want: &spanIDRatioSampler{spanIDUpperBound: int64(float64(0.20221111) * maxUpperBound)},
		},
		{
			name: "fraction is very close to 1",
			args: args{fraction: 0.99999999999},
			want: &spanIDRatioSampler{spanIDUpperBound: int64(float64(0.99999999999) * maxUpperBound)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newSpanIDRatioSampler(tt.args.fraction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newSpanIDRatioSampler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_spanIDRatioSampler_ShouldSample(t *testing.T) {
	type fields struct {
		spanIDUpperBound int64
	}
	type args struct {
		id SpanID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "ID equals spanIDUpperBound",
			fields: fields{spanIDUpperBound: 20221111},
			args:   args{id: 20221111},
			want:   false,
		},
		{
			name:   "ID less than spanIDUpperBound",
			fields: fields{spanIDUpperBound: 20221111},
			args:   args{id: 20211111},
			want:   true,
		},
		{name: "ID greater than spanIDUpperBound",
			fields: fields{spanIDUpperBound: 20221111},
			args:   args{id: 20231111},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := spanIDRatioSampler{
				spanIDUpperBound: tt.fields.spanIDUpperBound,
			}
			if got := ss.shouldSample(tt.args.id); got != tt.want {
				t.Errorf("shouldSample() = %v, want %v", got, tt.want)
			}
		})
	}
}
