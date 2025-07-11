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

package rpcz

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_span_AddEvent(t *testing.T) {
	t.Run("add event sequentially", func(t *testing.T) {
		s := newSpan("cyk", 666, nil)
		events := []string{"sing", "dance", "rap"}
		for _, e := range events {
			time.Sleep(10 * time.Millisecond)
			s.AddEvent(e)
		}
		for i, e := range events {
			require.Equal(t, e, s.events[i].Name)
			if i+1 < len(events) {
				require.True(t, s.events[i].Time.Before(s.events[i+1].Time))
			}
		}
	})
}

func Test_span_SetAttribute(t *testing.T) {
	t.Run("set value of attribute as slice", func(t *testing.T) {
		sliceValue := []string{"filter1", "filter2"}
		s := &span{}
		s.SetAttribute(TRPCAttributeFilterNames, sliceValue)

		sliceValue[0] = "filter3"
		require.Equal(t, sliceValue, s.attributes[0].Value)

		sliceValue = sliceValue[1:]
		require.NotEqual(t, sliceValue, s.attributes[0].Value)
	})
	t.Run("set same attribute twice", func(t *testing.T) {
		s := &span{}
		s.SetAttribute("Decode", "success")
		s.SetAttribute("Decode", "failed")
		require.Len(t, s.attributes, 2)
		attribute, ok := s.Attribute("Decode")
		require.True(t, ok)
		require.Equal(t, "failed", attribute)
	})
	t.Run("set different attribute", func(t *testing.T) {
		s := &span{}
		s.SetAttribute("Decode", "success")
		require.Len(t, s.attributes, 1)
		require.Equal(t, "success", s.attributes[0].Value)

		s.SetAttribute("Encode", "failed")
		require.Len(t, s.attributes, 2)
		require.Equal(t, "failed", s.attributes[1].Value)
	})
}

func Test_span_Attribute(t *testing.T) {
	t.Run("modify return directly will affect related Span", func(t *testing.T) {
		s := &span{}
		s.SetAttribute("Decode", &[]int{1})
		aRaw1, ok := s.Attribute("Decode")
		require.True(t, ok)
		a1 := aRaw1.(*[]int)
		*a1 = append(*a1, 2)

		aRaw2, ok := s.Attribute("Decode")
		require.True(t, ok)
		a2 := aRaw2.(*[]int)
		require.Same(t, a1, a2)
	})
}

func Test_span_StartTime(t *testing.T) {
	ps := &span{}
	require.Zero(t, ps.StartTime())

	cs, ender := ps.NewChild("")
	defer ender.End()
	require.NotZero(t, cs.StartTime())
}

func Test_Span_EndTime(t *testing.T) {
	t.Run("*span new child", func(t *testing.T) {
		ps := &span{}
		require.Zero(t, ps.EndTime())

		cs, ender := ps.NewChild("")
		require.Zero(t, cs.EndTime())

		ender.End()
		require.NotZero(t, cs.EndTime())
	})
	t.Run("rpcz new child", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: 1})
		ps, psEnder := rpcz.NewChild("")
		require.Zero(t, ps.EndTime())

		cs, csEnder := ps.NewChild("")
		require.Zero(t, cs.EndTime())

		csEnder.End()
		require.NotZero(t, cs.EndTime())
		require.Zero(t, ps.EndTime())

		psEnder.End()
		require.NotZero(t, ps.EndTime())
	})

}

func Test_span_End(t *testing.T) {
	t.Run("put span to pool directly if span's parent don't have this child span", func(t *testing.T) {
		s := &span{
			id:        1,
			name:      "server",
			parent:    &span{},
			startTime: time.Now(),
			childSpans: []*span{
				{name: "client1"},
				{name: "client2"},
			},
			events: []Event{
				{Name: "locking"},
				{Name: "locked"},
				{Name: "unlocked"},
			},
			attributes: []Attribute{
				{
					Name:  "size",
					Value: 67,
				},
			},
		}
		require.True(t, s.endTime.IsZero())

		s.End()
		require.Equal(t, &span{
			id:         nilSpanID,
			childSpans: []*span{},
			events:     []Event{},
			attributes: []Attribute{},
		}, s)
	})
	t.Run("put span to pool directly if span's parent is nil", func(t *testing.T) {
		s := &span{
			id:        1,
			name:      "server",
			parent:    nil,
			startTime: time.Now(),
			childSpans: []*span{
				{name: "client1"},
				{name: "client2"},
			},
			events: []Event{
				{Name: "locking"},
				{Name: "locked"},
				{Name: "unlocked"},
			},
			attributes: []Attribute{
				{
					Name:  "size",
					Value: 67,
				},
			},
		}
		require.True(t, s.endTime.IsZero())
		require.Panicsf(t, s.End, "should panic because of nil pointer dereference")
	})
	t.Run("calling End repeatedly won't reset endTime ", func(t *testing.T) {
		ps := newSpan("server", 1, NewRPCZ(&Config{Capacity: 100}))
		cs, end := ps.NewChild("client")
		require.Equal(t, cs, ps.childSpans[0])

		end.End()
		endTime := ps.childSpans[0].endTime
		require.False(t, endTime.IsZero(), "child span has been record to parent span")

		end.End()
		require.Equal(t, endTime, cs.(*span).endTime)
	})
	t.Run("record root span to rpcz", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Fraction: 1.0, Capacity: 10})
		const spanName = "server"
		s, end := rpcz.NewChild(spanName)
		id := s.ID()
		end.End()

		readOnlySpan, ok := rpcz.Query(id)
		require.True(t, ok)
		require.Equal(t, id, readOnlySpan.ID)
		require.Equal(t, spanName, readOnlySpan.Name)
	})
	t.Run("try record root span to rpcz more than once", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Fraction: 1, Capacity: 10})
		s, end := rpcz.NewChild("server")
		id := s.ID()
		require.Zero(t, rpcz.store.spans.length)

		end.End()
		require.Equal(t, uint32(1), rpcz.store.spans.length)
		readOnlySpan1, ok := rpcz.Query(id)
		require.True(t, ok)

		end.End()
		require.Equal(t, uint32(1), rpcz.store.spans.length)
		readOnlySpan2, ok := rpcz.Query(id)
		require.True(t, ok)
		require.Equal(t, readOnlySpan1, readOnlySpan2)
	})
	t.Run("record span to root span", func(t *testing.T) {
		ps := newSpan("server", 1, NewRPCZ(&Config{Capacity: 100}))
		cs, end := ps.NewChild("client")
		require.Equal(t, cs, ps.childSpans[0])

		end.End()
		require.False(t, ps.childSpans[0].endTime.IsZero(), "child span has been record to parent span")
	})
	t.Run("record span trigger reclaims memory of span pool, and parent span has been recycled", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Capacity: 1, Fraction: 1.0})
		ps1, endPS1 := rpcz.NewChild("server")
		{
			ps1 := ps1.(*span)
			cs1, endCS1 := ps1.NewChild("client")
			cs1Cast := cs1.(*span)
			readOnlySpan := cs1Cast.convertedToReadOnlySpan()
			require.Equal(t, cs1Cast, ps1.childSpans[0])

			endPS1.End()
			require.Equal(t, cs1, ps1.childSpans[0])

			_, endPS2 := rpcz.NewChild("server")
			endPS2.End()
			require.Empty(t, ps1.childSpans, "ps1 should been put in pool due limit of rpcz's capacity")
			require.Equal(t, readOnlySpan, cs1Cast.convertedToReadOnlySpan(), "cs shouldn't be put in pool before call end")

			endCS1.End()
			require.Equal(t, &span{id: nilSpanID}, cs1Cast, "cs should been put in pool if its parent has been recycled")
		}
	})
	t.Run("record span trigger reclaims memory of span pool, and parent span hasn't been recycled", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Capacity: 1, Fraction: 1.0})
		ps1, endPS1 := rpcz.NewChild("server")
		ps1Cast := ps1.(*span)
		cs1, endCS1 := ps1Cast.NewChild("client")
		cs1Cast := cs1.(*span)
		expectedReadOnlySpan := cs1Cast.convertedToReadOnlySpan()
		require.Equal(t, cs1, ps1Cast.childSpans[0])

		endPS1.End()
		require.Equal(t, cs1, ps1Cast.childSpans[0])

		_, endPS2 := rpcz.NewChild("server")

		endCS1.End()
		require.True(t, cs1Cast.isEnded())
		readOnlySpan := cs1Cast.convertedToReadOnlySpan()
		readOnlySpan.EndTime = expectedReadOnlySpan.EndTime
		require.Equal(
			t,
			expectedReadOnlySpan,
			readOnlySpan,
			"cs shouldn't been put in pool if its parent hasn't been recycled",
		)

		endPS2.End()
		require.Empty(t, ps1Cast.childSpans, "ps1 should been put in pool due limit of rpcz's capacity")
		require.Equal(t, &span{id: nilSpanID}, cs1, "parent span has put his child span to pool")
	})
}

func Test_span_ID(t *testing.T) {
	s := &span{}
	require.Equal(t, SpanID(0), s.ID())

	s = newSpan("", 1, nil)
	require.Equal(t, SpanID(1), s.ID())
}

func Test_span_Name(t *testing.T) {
	rpcz := NewRPCZ(&Config{Fraction: 1, Capacity: 10})
	s, end := rpcz.NewChild("server")
	defer end.End()
	require.Equal(t, "server", s.Name())
}

func Test_span_attribute(t *testing.T) {
	s := span{}

	t.Run("span doesn't have specific attribute", func(t *testing.T) {
		v, ok := s.Attribute("non-exist-attribute")
		require.False(t, ok)
		require.Nil(t, v)
	})
	t.Run("span has specific attribute", func(t *testing.T) {
		const (
			name  = "ResponseSize"
			value = 101
		)
		s.SetAttribute(name, value)
		v, ok := s.Attribute(name)
		require.True(t, ok)
		require.Equal(t, value, v)
	})
}

func Test_span_NewChild(t *testing.T) {
	t.Run("parent span doesn't have TRPCAttributeFilterNames", func(t *testing.T) {
		ps := newSpan("server", 1, NewRPCZ(&Config{Capacity: 10}))
		cs, _ := ps.NewChild("client")
		{
			cs := cs.(*span)
			require.Equal(t, ps, cs.parent)
			v, ok := cs.Attribute(TRPCAttributeFilterNames)
			require.False(t, ok)
			require.Nil(t, v)
		}
	})
}

func Test_span_Child(t *testing.T) {
	t.Run("query child ok", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Capacity: 1, Fraction: 1.0})
		s, ender := rpcz.NewChild("CoV")
		_, csEnder := s.NewChild("alpha")
		_, ok := s.Child("alpha")
		require.True(t, ok)

		csEnder.End()
		_, ok = s.Child("alpha")
		require.True(t, ok)

		ender.End()
		_, ok = s.Child("alpha")
		require.True(t, ok)
	})
	t.Run("modify *span.Child return result", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Capacity: 1, Fraction: 1.0})
		s, ender := rpcz.NewChild("CoV")
		cs, csEnder := s.NewChild("beta")
		{
			cs, _ := s.Child("beta")
			cs.SetAttribute("ATK", "9999+ points")
		}
		_, ok := cs.Attribute("ATK")
		require.True(t, ok)
		csEnder.End()
		ender.End()
	})
	t.Run("query child failed", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Capacity: 1, Fraction: 1.0})
		s, ender := rpcz.NewChild("CoV")
		defer ender.End()
		_, csEnder := s.NewChild("gamma")
		defer csEnder.End()
		c, ok := s.Child("delta")
		require.False(t, ok)
		require.Nil(t, c)
	})
}
