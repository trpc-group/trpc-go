// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/rpcz"
)

func TestRPCZ_ShouldRecord(t *testing.T) {
	t.Run("checking ID", func(t *testing.T) {
		r := rpcz.NewRPCZ(&rpcz.Config{
			Fraction: 1.0,
			Capacity: 1,
			ShouldRecord: func(s rpcz.Span) bool {
				return s.ID()%2 == 0
			},
		})
		s, ender := r.NewChild("")
		id := s.ID()
		ender.End()
		readOnlySpan, ok := r.Query(id)
		if id%2 == 0 {
			require.True(t, ok)
			require.Equal(t, id, readOnlySpan.ID)
		} else {
			require.False(t, ok)
		}
	})
	t.Run("checking Attribute", func(t *testing.T) {
		const attributeName = "Error"
		attributeValue := errors.New("test error")
		r := rpcz.NewRPCZ(&rpcz.Config{
			Fraction: 1.0,
			Capacity: 1,
			ShouldRecord: func(s rpcz.Span) bool {
				if err, ok := s.Attribute(attributeName); ok {
					err, ok := err.(error)
					return ok && err != nil
				}
				return false
			},
		})
		s, ender := r.NewChild("")
		id := s.ID()
		s.SetAttribute(attributeName, attributeValue)
		ender.End()
		readOnlySpan, _ := r.Query(id)
		require.Contains(t, readOnlySpan.Attributes, rpcz.Attribute{Name: attributeName, Value: attributeValue})

		s, ender = r.NewChild("")
		id = s.ID()
		ender.End()
		_, ok := r.Query(id)
		require.False(t, ok)
	})
	t.Run("checking StartTime and EndTime", func(t *testing.T) {
		const maxDuration = 100 * time.Millisecond
		r := rpcz.NewRPCZ(&rpcz.Config{
			Fraction: 1.0,
			Capacity: 1,
			ShouldRecord: func(s rpcz.Span) bool {
				return s.EndTime().Sub(s.StartTime()) > maxDuration
			},
		})
		s, ender := r.NewChild("slow one")
		id := s.ID()
		// to mimic some time-consuming operation.
		time.Sleep(150 * time.Millisecond)
		ender.End()
		readOnlySpan, _ := r.Query(id)
		require.Contains(t, readOnlySpan.Name, "slow one")

		s, ender = r.NewChild("fast one")
		id = s.ID()
		ender.End()
		_, ok := r.Query(id)
		require.False(t, ok)
	})
	t.Run("checking Events", func(t *testing.T) {
		const specialEvent = "leave func"
		r := rpcz.NewRPCZ(&rpcz.Config{
			Fraction: 1.0,
			Capacity: 1,
			ShouldRecord: func(s rpcz.Span) bool {
				_, ok := s.Event(specialEvent)
				return ok
			},
		})
		s, ender := r.NewChild("doesn't contain special event")
		id := s.ID()
		s.AddEvent("enter func")
		ender.End()
		_, ok := r.Query(id)
		require.False(t, ok)

		s, ender = r.NewChild("contain special event")
		id = s.ID()
		s.AddEvent(specialEvent)
		ender.End()
		readOnlySpan, _ := r.Query(id)
		require.Equal(t, specialEvent, readOnlySpan.Events[0].Name)
	})
}
