package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/metrics"
)

func Test_counter_Incr(t *testing.T) {
	metrics.RegisterMetricsSink(&metrics.ConsoleSink{})
	type fields struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"counter-1", fields{"counter-req.total"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := metrics.Counter(tt.fields.name)
			c.Incr()
			c.IncrBy(10)
			assert.NotNil(t, c)
		})
	}
}

func TestGetSameMetrics(t *testing.T) {
	c1 := metrics.Counter("x")
	c2 := metrics.Counter("x")
	c3 := metrics.Counter("y")

	assert.Equal(t, c1, c2)
	assert.NotEqual(t, c1, c3)

	g1 := metrics.Gauge("x")
	g2 := metrics.Gauge("x")
	g3 := metrics.Gauge("y")

	assert.Equal(t, g1, g2)
	assert.NotEqual(t, g1, g3)

	t1 := metrics.Timer("x")
	t2 := metrics.Timer("x")
	t3 := metrics.Timer("y")

	assert.Equal(t, t1, t2)
	assert.NotEqual(t, t1, t3)

	h1 := metrics.Histogram("x", metrics.NewDurationBounds())
	h2 := metrics.Histogram("x", metrics.NewDurationBounds())
	h3 := metrics.Histogram("y", metrics.NewDurationBounds())

	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
}

func BenchmarkReportCounter(b *testing.B) {

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := metrics.Counter("x")
			c.Incr()
		}
	})
}

func BenchmarkReportGauge(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := metrics.Gauge("x")
			c.Set(1)
		}
	})
}

func BenchmarkReportTimer(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := metrics.Timer("x")
			c.Record()
		}
	})
}

func BenchmarkReportHistogram(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := metrics.Histogram("x", metrics.NewValueBounds(10, 20, 50, 100))
			c.AddSample(1)
		}
	})
}
