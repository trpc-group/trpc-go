// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rand

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testSafeRand = NewSafeRand(time.Now().UnixNano())

func TestIntnFix(t *testing.T) {
	fixRand := NewSafeRand(1)
	for i := 0; i < 100; i++ {
		ranNum := fixRand.Intn(10)
		assert.Less(t, ranNum, 10)
		t.Logf("c=%v", ranNum)
	}
}
func TestSafeRand_Int63n(t *testing.T) {
	type args struct {
		n int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "1",
			args: args{
				n: 5,
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testSafeRand.Int63n(tt.args.n); got == tt.want {
				t.Errorf("Int63n() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeRand_Intn(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "1",
			args: args{
				n: 5,
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testSafeRand.Intn(tt.args.n); got == tt.want {
				t.Errorf("Int63n() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeRand_Float64(t *testing.T) {
	tests := []struct {
		name string
		want float64
	}{
		{
			name: "ok",
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testSafeRand.Float64(); got == tt.want {
				t.Errorf("Float64() = %v, want %v", got, tt.want)
			}
		})
	}
}
