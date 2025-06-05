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

package fasttime

import (
	"testing"
	"time"
)

func BenchmarkTimestamp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Timestamp()
	}
}

func BenchmarkNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.Now().Unix()
	}
}

func TestTimestamp(t *testing.T) {
	delayThreshold := int64(10)
	now := Timestamp()
	if unix := time.Now().Unix(); unix-now > delayThreshold {
		t.Fatalf("expect %d got %d", unix, now)
	}
	time.Sleep(time.Second + time.Millisecond)
	now = Timestamp()
	if unix := time.Now().Unix(); unix-now > delayThreshold {
		t.Fatalf("expect %d got %d", unix, now)
	}
}
