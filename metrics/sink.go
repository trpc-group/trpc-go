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

// Policy is the metrics aggregation policy.
type Policy int

// All available Policy(s).
const (
	PolicyNONE      = 0 // Undefined
	PolicySET       = 1 // instantaneous value
	PolicySUM       = 2 // summary
	PolicyAVG       = 3 // average
	PolicyMAX       = 4 // maximum
	PolicyMIN       = 5 // minimum
	PolicyMID       = 6 // median
	PolicyTimer     = 7 // timer
	PolicyHistogram = 8 // histogram
)

// Sink defines the interface an external monitor system should provide.
type Sink interface {
	// Name returns the name of the monitor system.
	Name() string
	// Report reports a record to monitor system.
	Report(rec Record, opts ...Option) error
}

// Record is the single record.
//
// terminologies:
//   - dimension name
//     is an attribute of a data, often used to filter data, such as a photo album business module
//     includes region and server room.
//   - dimension value
//     refines the dimension. For example, the regions of the album business module include Shenzhen,
//     Shanghai, etc., the region is the dimension, and Shenzhen and Shanghai are the dimension
//     values.
//   - metric
//     is a measurement, used to aggregate and calculate. For example, request count of album business
//     module in ShenZhen Telecom is a metric.
type Record struct {
	Name string // the name of the record
	// dimension name: such as region, host and disk number.
	// dimension value: such as region=ShangHai.
	dimensions []*Dimension
	metrics    []*Metrics
}

// Dimension defines the dimension.
type Dimension struct {
	Name  string
	Value string
}

// GetName returns the record name.
func (r *Record) GetName() string {
	return r.Name
}

// GetDimensions returns dimensions.
func (r *Record) GetDimensions() []*Dimension {
	if r == nil {
		return nil
	}
	return r.dimensions
}

// GetMetrics returns metrics.
func (r *Record) GetMetrics() []*Metrics {
	if r == nil {
		return nil
	}
	return r.metrics
}

// NewSingleDimensionMetrics creates a Record with no dimension and only one metric.
func NewSingleDimensionMetrics(name string, value float64, policy Policy) Record {
	return Record{
		dimensions: nil,
		metrics: []*Metrics{
			{name: name, value: value, policy: policy},
		},
	}
}

// ReportSingleDimensionMetrics creates and reports a Record with no dimension and only one metric.
func ReportSingleDimensionMetrics(name string, value float64, policy Policy, opts ...Option) error {
	return Report(Record{
		dimensions: nil,
		metrics: []*Metrics{
			{name: name, value: value, policy: policy},
		},
	}, opts...)
}

// NewMultiDimensionMetricsX creates a named Record with multiple dimensions and metrics.
func NewMultiDimensionMetricsX(name string, dimensions []*Dimension, metrics []*Metrics) Record {
	return Record{
		Name:       name,
		dimensions: dimensions,
		metrics:    metrics,
	}
}

// ReportMultiDimensionMetricsX creates and reports a named Record with multiple dimensions and
// metrics.
func ReportMultiDimensionMetricsX(
	name string,
	dimensions []*Dimension,
	metrics []*Metrics,
	opts ...Option,
) error {
	return Report(Record{
		Name:       name,
		dimensions: dimensions,
		metrics:    metrics,
	}, opts...)
}

// Metrics defines the metric.
type Metrics struct {
	name   string  // metric name
	value  float64 // metric value
	policy Policy  // aggregation policy
}

// NewMetrics creates a new Metrics.
func NewMetrics(name string, value float64, policy Policy) *Metrics {
	return &Metrics{name, value, policy}
}

// Name returns the metrics name.
func (m *Metrics) Name() string {
	if m == nil {
		return ""
	}
	return m.name
}

// Value returns the metrics value.
func (m *Metrics) Value() float64 {
	if m == nil {
		return 0
	}
	return m.value
}

// Policy returns the metrics policy.
func (m *Metrics) Policy() Policy {
	if m == nil {
		return PolicyNONE
	}
	return m.policy
}
