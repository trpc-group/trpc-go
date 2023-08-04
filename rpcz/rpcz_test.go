package rpcz

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/log"
)

func TestRPCZ_NewChildSpan(t *testing.T) {
	t.Run("disable rpcz", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Fraction: 0.0, Capacity: 10})
		s, _ := rpcz.NewChild("server")
		require.Equal(t, rpcz, s)
	})
	t.Run("New span", func(t *testing.T) {
		ctx := context.Background()
		ctx, canceled := context.WithCancel(ctx)
		canceled()

		rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: 10})
		s, _ := rpcz.NewChild("server")
		sp := s.(*span)
		require.Equal(t, rpcz, sp.parent)
		require.Equal(t, "server", sp.name)
	})
}

func TestRPCZ_Query(t *testing.T) {
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: 10})
	t.Run("span is not in rpcz", func(t *testing.T) {
		_, ok := rpcz.Query(0)
		require.False(t, ok)
	})
	t.Run("span is in rpcz", func(t *testing.T) {
		s := newSpan("server", 1, rpcz)
		s.End()
		readOnlySpan, ok := rpcz.Query(1)
		require.True(t, ok)
		require.Equal(t, "server", readOnlySpan.Name)
		require.Equal(t, SpanID(1), readOnlySpan.ID)
	})
	t.Run("span that contains child span is in rpcz", func(t *testing.T) {
		s := newSpan("server", 2, rpcz)
		_, end := s.NewChild("client")
		end.End()
		s.End()
		readOnlySpan, ok := rpcz.Query(2)
		require.True(t, ok)
		childSpan := readOnlySpan.ChildSpans[0]
		require.Equal(t, "client", childSpan.Name)
		require.Equal(t, SpanID(2), childSpan.ID, "child span id is equal parent span")
	})
}

func TestRPCZ_BatchQuery(t *testing.T) {
	const insertedNum = 10
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: insertedNum})

	t.Run("query num less than zero", func(t *testing.T) {
		require.Empty(t, rpcz.BatchQuery(-1))
	})
	t.Run("query num equals zero", func(t *testing.T) {
		require.Empty(t, rpcz.BatchQuery(0))
	})

	recordSpansToRPCZ(insertedNum, rpcz)
	t.Run("returns newly inserted span", func(t *testing.T) {
		readOnlySpans := rpcz.BatchQuery(5)
		require.Len(t, readOnlySpans, 5)
		for i, s := range readOnlySpans {
			require.Equal(t, fmt.Sprintf("server-%d", insertedNum-i), s.Name)
		}
	})
	t.Run("query num greater than insertedNum of stored span", func(t *testing.T) {
		readOnlySpans := rpcz.BatchQuery(insertedNum + 1)
		require.Len(t, readOnlySpans, insertedNum)
		for i, s := range readOnlySpans {
			require.Equal(t, fmt.Sprintf("server-%d", insertedNum-i), s.Name)
		}
	})
}

func TestRPCZ_RecordManySpans(t *testing.T) {
	t.Run("RPCZ has small capacity", func(t *testing.T) {
		const num = 1000000
		rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: 1})
		recordSpansToRPCZ(num, rpcz)
		readOnlySpans := rpcz.BatchQuery(num)
		require.Len(t, readOnlySpans, 1)
		require.Equal(t, fmt.Sprintf("server-%d", num), readOnlySpans[0].Name)
	})
	t.Run("RPCZ has large capacity", func(t *testing.T) {
		const largeCapacity = 100000
		rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: largeCapacity})
		recordSpansToRPCZ(10*largeCapacity, rpcz)
		readOnlySpans := rpcz.BatchQuery(10 * largeCapacity)
		require.Len(t, readOnlySpans, largeCapacity)
	})
	t.Run("RPCZ is configured with exporter", func(t *testing.T) {
		const maxCapacity = 100000
		exporter := newSliceSpanExporter(maxCapacity)
		rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: maxCapacity + 1, Exporter: exporter})

		recordSpansToRPCZ(maxCapacity+1, rpcz)
		readOnlySpans := rpcz.BatchQuery(10 * maxCapacity)

		require.Len(t, readOnlySpans, maxCapacity+1)
		require.Len(t, exporter.spans, maxCapacity, "the rpcz still stores a copy of the exported span")
	})
}

type sliceSpanExporter struct {
	spans       []ReadOnlySpan
	maxCapacity uint64
}

func newSliceSpanExporter(maxCapacity uint64) *sliceSpanExporter {
	return &sliceSpanExporter{maxCapacity: maxCapacity}
}

func (e *sliceSpanExporter) Export(span *ReadOnlySpan) {
	if uint64(len(e.spans)) >= e.maxCapacity {
		log.Info("exporter has been filled, and no more spans can be received")
		return
	}
	e.spans = append(e.spans, *span)
}

func recordSpansToRPCZ(num int, rpcz *RPCZ) {
	for i := 1; i <= num; i++ {
		s := newSpan(fmt.Sprintf("server-%d", i), SpanID(i), rpcz)
		s.End()
	}
}

const smallCapacity = 1

func BenchmarkRPCZSmallCapacity_NewSpanEndZero(b *testing.B) {
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: smallCapacity})
	for i := 0; i < b.N; i++ {
		_, _ = rpcz.NewChild("")
	}
}

func BenchmarkRPCZSmallCapacity_NewSpanAndEndHalf(b *testing.B) {
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: smallCapacity})
	for i := 0; i < b.N; i++ {
		_, end := rpcz.NewChild("")
		if i%2 == 0 {
			end.End()
		}
	}
}

func BenchmarkRPCZSmallCapacity_NewSpanAndEndAll(b *testing.B) {
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: smallCapacity})
	for i := 0; i < b.N; i++ {
		_, end := rpcz.NewChild("")
		end.End()
	}
}

const largeCapacity = 1000000

func BenchmarkRPCZLargeCapacity_NewSpanEndZero(b *testing.B) {
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: largeCapacity})
	for i := 0; i < largeCapacity; i++ {
		_, _ = rpcz.NewChild("")
	}
}

func BenchmarkRPCZLargeCapacity_NewSpanAndEndHalf(b *testing.B) {
	rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: largeCapacity})
	for i := 0; i < largeCapacity; i++ {
		_, end := rpcz.NewChild("")
		if i%2 == 0 {
			end.End()
		}
	}
}

func BenchmarkRPCZLargeCapacity_NewSpanAndEndAll(b *testing.B) {
	var rpcz Span = NewRPCZ(&Config{Fraction: 1.0, Capacity: largeCapacity})
	for i := 0; i < largeCapacity; i++ {
		_, end := rpcz.NewChild("")
		end.End()
	}
}
