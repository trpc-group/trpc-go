// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_noopSpan(t *testing.T) {
	s := noopSpan{}
	t.Run("AddEvent", func(t *testing.T) {
		s.AddEvent("happy birthday, tencent 24th anniversary")
		require.Equal(t, noopSpan{}, s)
	})
	t.Run("ID", func(t *testing.T) {
		require.Equal(t, nilSpanID, s.ID())
		require.Equal(t, noopSpan{}, s)
	})
	t.Run("Name", func(t *testing.T) {
		require.Empty(t, s.Name())
	})
	t.Run("SetAttribute", func(t *testing.T) {
		s.SetAttribute("Friday", "go home early :)")
		require.Equal(t, noopSpan{}, s)
	})
	t.Run("NewChild", func(t *testing.T) {
		s, _ := s.NewChild("child")
		require.Equal(t, noopSpan{}, s)
	})
	t.Run("NoopSpan Comparison", func(t *testing.T) {
		s1 := noopSpan{}
		s2 := noopSpan{}
		require.True(t, s1 == s2)
		require.Same(t, &noopSpan{}, &s1)
	})
	t.Run("Attribute", func(t *testing.T) {
		s := noopSpan{}
		const attributeName = "attribute"
		s.SetAttribute(attributeName, "value")
		attribute, ok := s.Attribute(attributeName)
		require.False(t, ok)
		require.Nil(t, attribute)
	})
	t.Run("StartTime", func(t *testing.T) {
		s := noopSpan{}
		require.Zero(t, s.StartTime())
	})
	t.Run("EndTime", func(t *testing.T) {
		s := noopSpan{}
		require.Zero(t, s.EndTime())
	})
	t.Run("Events", func(t *testing.T) {
		s := noopSpan{}
		s.AddEvent("event")
		time, ok := s.Event("event")
		require.False(t, ok)
		require.Zero(t, time)
	})
	t.Run("Child", func(t *testing.T) {
		s := noopSpan{}
		_, ender := s.NewChild("child")
		ender.End()
		_, ok := s.Child("child")
		require.False(t, ok)
	})
}
