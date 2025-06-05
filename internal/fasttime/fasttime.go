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

// Package fasttime provides a fast way to get current timestamp.
package fasttime

import (
	"sync/atomic"
	"time"
)

var now int64 // atomic value

func init() {
	now = time.Now().Unix()
	go func() {
		for range time.Tick(time.Millisecond * 100) {
			atomic.StoreInt64(&now, time.Now().Unix())
		}
	}()
}

// Timestamp returns the current timestamp in seconds.
func Timestamp() int64 {
	return atomic.LoadInt64(&now)
}
