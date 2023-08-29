// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import "math"

type spanIDRatioSampler struct {
	spanIDUpperBound int64
}

// shouldSample determines whether sampling is required.
func (ss spanIDRatioSampler) shouldSample(id SpanID) bool {
	return int64(id) < ss.spanIDUpperBound
}

// newSpanIDRatioSampler creates a spanIDRatioSampler that samples a given fraction of span.
// fraction >= 1 will always sample.
// fraction < 0 are treated as zero.
func newSpanIDRatioSampler(fraction float64) *spanIDRatioSampler {
	const maxUpperBound = math.MaxInt64

	var upperBound int64
	if ub := fraction * maxUpperBound; ub >= maxUpperBound {
		upperBound = maxUpperBound
	} else if ub >= 0 {
		upperBound = int64(ub)
	}

	return &spanIDRatioSampler{spanIDUpperBound: upperBound}
}
