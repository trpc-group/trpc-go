package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/metrics"
)

func Test_histogram_AddSample(t *testing.T) {

	buckets := metrics.NewDurationBounds(time.Second, time.Second*2, time.Second*5)
	h := metrics.Histogram("req.timecost", buckets)

	metrics.RegisterMetricsSink(&metrics.ConsoleSink{})

	type args struct {
		value float64
	}
	tests := []struct {
		name string
		args args
	}{
		{"histogram-sample-0.5s", args{float64(time.Millisecond * 500)}},
		{"histogram-sample-1.5s", args{float64(time.Millisecond * 1500)}},
		{"histogram-sample-2.5s", args{float64(time.Millisecond * 2500)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.AddSample(tt.args.value)
			b := h.GetBuckets()
			assert.NotEqual(t, 0, len(b))
		})
	}
}
