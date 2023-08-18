// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package allocator_test

import (
	"testing"

	. "trpc.group/trpc-go/trpc-go/internal/allocator"

	"github.com/stretchr/testify/require"
)

func TestDefaultAllocator(t *testing.T) {
	bs, free := Malloc(10)
	require.Equal(t, 10, len(bs))
	Free(free)
}

func TestClassAllocator(t *testing.T) {
	a := NewClassAllocator()
	bs, free := a.Malloc(10)
	require.Equal(t, 10, len(bs))
	a.Free(free)
}

func TestClassAllocator_InvalidMalloc(t *testing.T) {
	a := NewClassAllocator()
	defer func() {
		require.NotEmpty(t, recover())
	}()
	a.Malloc(-1)
}

func TestClassAllocator_InvalidFree(t *testing.T) {
	a := NewClassAllocator()
	t.Run("free empty slice", func(t *testing.T) {
		defer func() {
			require.NotEmpty(t, recover())
		}()
		a.Free(nil)
	})
	t.Run("invalid slice size", func(t *testing.T) {
		defer func() {
			require.NotEmpty(t, recover())
		}()
		a.Free(make([]byte, 9))
	})
}
