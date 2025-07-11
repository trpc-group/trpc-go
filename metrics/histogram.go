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

package metrics

import (
	"math"
	"sort"
	"sync"
	"time"
)

// IHistogram is the interface that emits histogram metrics.
type IHistogram interface {
	// AddSample records a sample into histogram.
	AddSample(value float64)
	// GetBuckets get histogram buckets.
	GetBuckets() []*bucket
}

// RegisterHistogram registers all named Histogram configurations to all Sink(s) which implement
// HistogramSink.
func RegisterHistogram(name string, o HistogramOption) {
	metricsSinksMutex.Lock()
	defer metricsSinksMutex.Unlock()
	for _, sink := range metricsSinks {
		if histSink, ok := sink.(HistogramSink); ok {
			histSink.Register(name, o)
		}
	}
}

// HistogramSink extends Sink in a way that allows to load a named Histogram configuration.
// Those who do not implement HistogramSink must define their own default bucket configuration.
type HistogramSink interface {
	Register(name string, o HistogramOption)
}

// HistogramOption defines configurations when register a histogram.
type HistogramOption struct {
	BucketBounds BucketBounds
}

// histogram defines the histogram. Each sample is added to one of predefined buckets.
type histogram struct {
	Name          string
	Meta          map[string]interface{}
	Spec          BucketBounds
	Buckets       []*bucket
	LookupByValue []float64
}

// newHistogram creates a named histogram with buckets.
func newHistogram(name string, buckets BucketBounds) *histogram {
	ranges := newBucketRanges(buckets)
	h := &histogram{
		Name:          name,
		Spec:          buckets,
		Buckets:       make([]*bucket, 0, len(ranges)),
		LookupByValue: make([]float64, 0, len(ranges)),
	}
	for _, r := range ranges {
		h.LookupByValue = append(h.LookupByValue, r.upperBoundValue)
		h.Buckets = append(h.Buckets, &bucket{
			h:               h,
			samples:         0.0,
			ValueLowerBound: r.lowerBoundValue,
			ValueUpperBound: r.upperBoundValue,
		})
	}
	return h
}

// AddSample adds a new sample.
func (h *histogram) AddSample(value float64) {
	idx := sort.SearchFloat64s(h.LookupByValue, value)
	h.Buckets[idx].mu.Lock()
	h.Buckets[idx].samples += value
	h.Buckets[idx].mu.Unlock()

	if len(metricsSinks) == 0 {
		return
	}

	r := NewSingleDimensionMetrics(h.Name, value, PolicyHistogram)
	for _, sink := range metricsSinks {
		sink.Report(r)
	}
}

// GetBuckets gets the buckets.
func (h *histogram) GetBuckets() []*bucket {
	return h.Buckets
}

// BucketBounds allows developers to customize Buckets of histogram.
type BucketBounds []float64

// NewValueBounds creates a value bounds.
func NewValueBounds(bounds ...float64) BucketBounds {
	return bounds
}

// NewDurationBounds creates duration bounds.
func NewDurationBounds(durations ...time.Duration) BucketBounds {
	bounds := make([]float64, 0, len(durations))
	for _, v := range durations {
		bounds = append(bounds, float64(v))
	}
	return bounds
}

// Len implements sort.Interface.
func (v BucketBounds) Len() int {
	return len(v)
}

// Swap implements sort.Interface.
func (v BucketBounds) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Less implements sort.Interface.
func (v BucketBounds) Less(i, j int) bool {
	return v[i] < v[j]
}

func (v BucketBounds) sorted() []float64 {
	valuesCopy := clone(v)
	sort.Sort(BucketBounds(valuesCopy))
	return valuesCopy
}

// bucket is used to assemble a histogram. Every bucket contains a counter.
type bucket struct {
	h *histogram

	ValueLowerBound float64
	ValueUpperBound float64

	mu      sync.Mutex
	samples float64
}

const inf = math.MaxFloat64

// newBucketRanges creates a set of bucket pairs from a set of Buckets describing the lower and upper newBucketRanges
// for each derived bucket.
func newBucketRanges(buckets BucketBounds) []bucketRange {
	if len(buckets) < 1 {
		s := bucketRange{lowerBoundValue: -inf, upperBoundValue: inf}
		return []bucketRange{s}
	}

	// if Buckets range is [A,B), don't forget (~,A) and [B,~)
	ranges := make([]bucketRange, 0, buckets.Len()+2)
	sortedBounds := buckets.sorted()
	lowerBoundValue := -inf

	for i := 0; i < buckets.Len(); i++ {
		ranges = append(ranges, bucketRange{lowerBoundValue: lowerBoundValue, upperBoundValue: sortedBounds[i]})
		lowerBoundValue = sortedBounds[i]
	}
	ranges = append(ranges, bucketRange{
		lowerBoundValue: sortedBounds[len(sortedBounds)-1],
		upperBoundValue: inf,
	})
	return ranges
}

// clone deeply copies a slice.
func clone(values []float64) []float64 {
	valuesCopy := make([]float64, len(values))
	copy(valuesCopy, values)
	return valuesCopy
}

// bucketRange is the bucket range.
type bucketRange struct {
	lowerBoundValue float64
	upperBoundValue float64
}
