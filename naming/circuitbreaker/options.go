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

package circuitbreaker

import "time"

var defaultOptions = Options{
	slidingWindowInterval:    60 * time.Second,
	slidingWindowSize:        12,
	minRequestsToOpen:        10,
	errRateToOpen:            0.5,
	continuousFailuresToOpen: 10,
	openDuration:             30 * time.Second,
	totalRequestsToClose:     10,
	successRequestsToClose:   8,
}

// Options defines the options of LRUCircuitBreakers.
type Options struct {
	slidingWindowInterval    time.Duration
	slidingWindowSize        int
	minRequestsToOpen        int
	errRateToOpen            float64
	continuousFailuresToOpen int
	openDuration             time.Duration
	totalRequestsToClose     int
	successRequestsToClose   int
}

// Opt modifies the Options.
type Opt func(*Options)

// WithSlidingWindowInterval returns an option that set sliding window interval.
func WithSlidingWindowInterval(interval time.Duration) Opt {
	return func(o *Options) {
		o.slidingWindowInterval = interval
	}
}

// WithSlidingWindowSize returns an option that set number of sliding window.
func WithSlidingWindowSize(size int) Opt {
	return func(o *Options) {
		o.slidingWindowSize = size
	}
}

// WithMinRequestsToOpen returns an option that set the min requests to open.
func WithMinRequestsToOpen(n int) Opt {
	return func(o *Options) {
		o.minRequestsToOpen = n
	}
}

// WithErrRateToOpen returns an option that set the error rate to open.
func WithErrRateToOpen(r float64) Opt {
	return func(o *Options) {
		o.errRateToOpen = r
	}
}

// WithContinuousFailuresToOpen returns an option that set the continuous failures to open.
func WithContinuousFailuresToOpen(n int) Opt {
	return func(o *Options) {
		o.continuousFailuresToOpen = n
	}
}

// WithOpenDuration returns an option that set the duration of opened phase.
func WithOpenDuration(d time.Duration) Opt {
	return func(o *Options) {
		o.openDuration = d
	}
}

// WithTotalRequestsToClose returns an option that set total requests to close.
func WithTotalRequestsToClose(n int) Opt {
	return func(o *Options) {
		o.totalRequestsToClose = n
		if o.successRequestsToClose > n {
			o.successRequestsToClose = n
		}
	}
}

// WithSuccessRequestsToClose returns an option that set number of successful requests to close.
func WithSuccessRequestsToClose(n int) Opt {
	return func(o *Options) {
		o.successRequestsToClose = n
		if o.totalRequestsToClose < n {
			o.totalRequestsToClose = n
		}
	}
}
