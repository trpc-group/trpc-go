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

package metrics_test

import (
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/metrics"
)

// 这里 timer 的误差精度设定为 1s，通过 time.sleep 来打桩模拟业务操作，测试 timer 的工作效果
func Test_timer_Record(t *testing.T) {

	precision := time.Second

	tests := []struct {
		name string
		wait time.Duration
	}{
		{"timer-1us", time.Microsecond},
		{"timer-10us", time.Microsecond * 10},
		{"timer-1ms", time.Millisecond},
		{"timer-10ms", time.Millisecond * 10},
		{"timer-1s", time.Second},
		{"timer-2s", time.Second * 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := metrics.Timer(tt.name)
			// do something
			time.Sleep(tt.wait)
			passed := tm.Record()
			deviation := passed - tt.wait
			if !(passed >= tt.wait && deviation <= precision) {
				t.Fatalf("timer record duration, want = %v, got = %v, deviation = %v",
					tt.wait, passed, deviation)
			} else {
				t.Logf("timer record duration, want = %v, got = %v, deviation = %v",
					tt.wait, passed, deviation)
			}
		})
	}
}
