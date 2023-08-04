package metrics

// IGauge is the interface that emits gauge metrics.
type IGauge interface {
	// Set sets the gauges absolute value.
	Set(value float64)
}

// gauge defines the gauge. gauge is reported to each external Sink-able system.
type gauge struct {
	name string
}

// Set updates the gauge value.
func (g *gauge) Set(v float64) {
	r := NewSingleDimensionMetrics(g.name, v, PolicySET)
	for _, sink := range metricsSinks {
		sink.Report(r)
	}
}
