package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// NewConsoleSink creates a new console sink.
func NewConsoleSink() Sink {
	return &ConsoleSink{
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		timers:     make(map[string]timer),
		histograms: make(map[string]histogram),
	}
}

// ConsoleSink defines the console sink.
type ConsoleSink struct {
	countersMu sync.RWMutex
	counters   map[string]float64

	gaugesMu sync.RWMutex
	gauges   map[string]float64

	timersMu sync.RWMutex
	timers   map[string]timer

	histogramsMu sync.RWMutex
	histograms   map[string]histogram
}

// Name returns console sink name.
func (c *ConsoleSink) Name() string {
	return "console"
}

// Report reports a record.
func (c *ConsoleSink) Report(rec Record, opts ...Option) error {
	if len(rec.dimensions) <= 0 {
		return c.reportSingleDimensionMetrics(rec, opts...)
	}
	return c.reportMultiDimensionMetrics(rec, opts...)
}

func (c *ConsoleSink) reportSingleDimensionMetrics(rec Record, _ ...Option) error {
	// almost all monitor systems support cumulant and instant.
	for _, m := range rec.metrics {
		switch m.policy {
		case PolicySUM:
			c.incrCounter(m.name, m.value)
		case PolicySET:
			c.setGauge(m.name, m.value)
		case PolicyTimer:
			c.recordTimer(m.name, time.Duration(m.value))
		case PolicyHistogram:
			c.addSample(m.name, m.value)
		default:
			// not supported policies
		}
	}
	return nil
}

func (c *ConsoleSink) reportMultiDimensionMetrics(rec Record, opts ...Option) error {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	type metric struct {
		Name   string  `json:"name"`
		Value  float64 `json:"value"`
		Policy Policy  `json:"policy"`
	}
	metrics := make([]*metric, 0, len(rec.GetMetrics()))
	for _, m := range rec.GetMetrics() {
		metrics = append(metrics, &metric{Name: m.Name(), Value: m.Value(), Policy: m.Policy()})
	}

	buf, err := json.Marshal(struct {
		Name       string       `json:"name"`
		Dimensions []*Dimension `json:"dimensions"`
		Metrics    []*metric    `json:"metrics"`
	}{
		Name:       rec.GetName(),
		Dimensions: rec.GetDimensions(),
		Metrics:    metrics,
	})
	if err != nil {
		return err
	}

	// a common multiple dimension report.
	fmt.Printf("metrics multi-dimension = %s\n", string(buf))
	return nil
}

// incrCounter increases counter.
func (c *ConsoleSink) incrCounter(key string, value float64) {
	if c.counters == nil {
		return
	}
	c.countersMu.Lock()
	c.counters[key] += value
	c.countersMu.Unlock()
	fmt.Printf("metrics counter[key] = %s val = %v\n", key, value)
}

// setGauge sets gauge.
func (c *ConsoleSink) setGauge(key string, value float64) {
	if c.gauges == nil {
		return
	}
	c.countersMu.Lock()
	c.gauges[key] = value
	c.countersMu.Unlock()
	fmt.Printf("metrics gauge[key] = %s val = %v\n", key, value)
}

// recordTimer records timer.
func (c *ConsoleSink) recordTimer(key string, duration time.Duration) {
	fmt.Printf("metrics timer[key] = %s val = %v\n", key, duration)
}

// addSample adds a sample.
func (c *ConsoleSink) addSample(key string, value float64) {
	histogramsMutex.RLock()
	h := histograms[key]
	histogramsMutex.RUnlock()

	v, ok := h.(*histogram)
	if !ok {
		return
	}
	hist := *v

	histogramsMutex.Lock()
	idx := sort.SearchFloat64s(hist.LookupByValue, value)
	upperBound := hist.Buckets[idx].ValueUpperBound
	hist.Buckets[idx].samples += value
	histogramsMutex.Unlock()

	fmt.Printf("metrics histogram[%s.%v] = %v\n", hist.Name, upperBound, hist.Buckets[idx].samples)
}
