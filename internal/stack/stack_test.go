// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package stack_test

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/stack"

	"github.com/stretchr/testify/require"
)

func TestStack(t *testing.T) {
	st := stack.New[struct{}]()
	st.Push(struct{}{})
	require.Equal(t, 1, st.Size())

	st.Reset()
	require.Equal(t, 0, st.Size())

	v, ok := st.Peek()
	require.False(t, ok)
	require.Equal(t, struct{}{}, v)

	v, ok = st.Pop()
	require.False(t, ok)
	require.Equal(t, struct{}{}, v)

	{
		type foo struct {
			bar string
		}

		st := stack.New[foo]()
		st.Push(foo{bar: "baz"})

		v, ok := st.Peek()
		require.True(t, ok)
		require.Equal(t, foo{bar: "baz"}, v)

		v, ok = st.Pop()
		require.True(t, ok)
		require.Equal(t, foo{bar: "baz"}, v)

		require.Zero(t, st.Size())
	}
}
