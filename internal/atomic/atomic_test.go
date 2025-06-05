//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package atomic_test

import (
	"sync"
	"testing"
	"unsafe"

	iatomic "trpc.group/trpc-go/trpc-go/internal/atomic"
	"github.com/stretchr/testify/require"
)

func TestBool(t *testing.T) {
	var val iatomic.Bool

	// Test Store and Load.
	val.Store(true)
	require.True(t, val.Load(), "Load should return the value stored by store")

	val.Store(false)
	require.False(t, val.Load(), "Load should return the value stored by store")

	// Test Swap.
	oldVal := val.Swap(true)
	require.False(t, oldVal, "Swap should return the old value")
	require.True(t, val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	swapped := val.CompareAndSwap(true, false)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.False(t, val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(true, true)
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.False(t, val.Load(), "Load should return the same value after failed CompareAndSwap")
}

func TestAtomicPointer(t *testing.T) {
	type someStruct struct {
		field string
	}

	var val iatomic.Pointer[someStruct]

	// Initialize a value for the pointer
	initialValue := &someStruct{field: "initial"}
	val.Store(initialValue)

	// Test Store and Load.
	require.Equal(t, initialValue, val.Load(), "Load should return the value stored by store")

	// Test Swap.
	newValue := &someStruct{field: "new"}
	oldVal := val.Swap(newValue)
	require.Equal(t, initialValue, oldVal, "Swap should return the old value")
	require.Equal(t, newValue, val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	comparedValue := &someStruct{field: "compared"}
	swapped := val.CompareAndSwap(newValue, comparedValue)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.Equal(t, comparedValue, val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(newValue, &someStruct{field: "failed"})
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.Equal(t, comparedValue, val.Load(), "Load should return the same value after failed CompareAndSwap")
}

func TestInt32(t *testing.T) {
	var val iatomic.Int32

	// Test Store and Load.
	val.Store(32)
	require.Equal(t, int32(32), val.Load(), "Load should return the value stored by store")

	// Test Add.
	newVal := val.Add(10)
	require.Equal(t, int32(42), newVal, "Add should return the new value")
	require.Equal(t, int32(42), val.Load(), "Load should return the new value after Add")

	// Test Swap.
	oldVal := val.Swap(0)
	require.Equal(t, int32(42), oldVal, "Swap should return the old value")
	require.Equal(t, int32(0), val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	swapped := val.CompareAndSwap(0, 128)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.Equal(t, int32(128), val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(0, 256)
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.Equal(t, int32(128), val.Load(), "Load should return the same value after failed CompareAndSwap")
}

func TestAtomicInt64(t *testing.T) {
	var val iatomic.Int64

	// Test Store and Load.
	val.Store(42)
	require.Equal(t, int64(42), val.Load(), "Load should return the value stored by Store")

	// Test Swap.
	oldVal := val.Swap(100)
	require.Equal(t, int64(42), oldVal, "Swap should return the old value")
	require.Equal(t, int64(100), val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	swapped := val.CompareAndSwap(100, 200)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.Equal(t, int64(200), val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(100, 300)
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.Equal(t, int64(200), val.Load(), "Load should return the same value after failed CompareAndSwap")

	// Test Add.
	addedValue := val.Add(50)
	require.Equal(t, int64(250), addedValue, "Add should return the new value after addition")
	require.Equal(t, int64(250), val.Load(), "Load should return the new value after Add")
}

func TestAtomicUint32(t *testing.T) {
	var val iatomic.Uint32

	// Test Store and Load.
	val.Store(123)
	require.Equal(t, uint32(123), val.Load(), "Load should return the value stored by Store")

	// Test Swap.
	oldVal := val.Swap(456)
	require.Equal(t, uint32(123), oldVal, "Swap should return the old value")
	require.Equal(t, uint32(456), val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	swapped := val.CompareAndSwap(456, 789)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.Equal(t, uint32(789), val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(456, 101112)
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.Equal(t, uint32(789), val.Load(), "Load should return the same value after failed CompareAndSwap")

	// Test Add.
	addedValue := val.Add(10)
	require.Equal(t, uint32(799), addedValue, "Add should return the new value after addition")
	require.Equal(t, uint32(799), val.Load(), "Load should return the new value after Add")
}

func TestUint64(t *testing.T) {
	var wg sync.WaitGroup
	var val iatomic.Uint64

	// Test Store and Load.
	val.Store(64)
	require.Equal(t, uint64(64), val.Load(), "Load should return the value stored by store")

	// Test Add.
	newVal := val.Add(10)
	require.Equal(t, uint64(74), newVal, "Add should return the new value")
	require.Equal(t, uint64(74), val.Load(), "Load should return the new value after Add")

	// Test Swap.
	oldVal := val.Swap(0)
	require.Equal(t, uint64(74), oldVal, "Swap should return the old value")
	require.Equal(t, uint64(0), val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	swapped := val.CompareAndSwap(0, 128)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.Equal(t, uint64(128), val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(0, 256)
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.Equal(t, uint64(128), val.Load(), "Load should return the same value after failed CompareAndSwap")

	// Test concurrent Add.
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			val.Add(1)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			val.Add(1)
		}
	}()
	wg.Wait()
	require.Equal(t, uint64(2128), val.Load(), "Load should return the correct value after concurrent adds")
}

func TestAtomicUintptr(t *testing.T) {
	var val iatomic.Uintptr

	// Test Store and Load.
	initialValue := uintptr(unsafe.Pointer(new(int)))
	val.Store(initialValue)
	require.Equal(t, initialValue, val.Load(), "Load should return the value stored by Store")

	// Test Swap.
	newValue := uintptr(unsafe.Pointer(new(int)))
	oldVal := val.Swap(newValue)
	require.Equal(t, initialValue, oldVal, "Swap should return the old value")
	require.Equal(t, newValue, val.Load(), "Load should return the new value after Swap")

	// Test CompareAndSwap.
	comparedValue := uintptr(unsafe.Pointer(new(int)))
	swapped := val.CompareAndSwap(newValue, comparedValue)
	require.True(t, swapped, "CompareAndSwap should succeed when old value is correct")
	require.Equal(t, comparedValue, val.Load(), "Load should return the new value after CompareAndSwap")

	swapped = val.CompareAndSwap(newValue, uintptr(unsafe.Pointer(new(int))))
	require.False(t, swapped, "CompareAndSwap should fail when old value is incorrect")
	require.Equal(t, comparedValue, val.Load(), "Load should return the same value after failed CompareAndSwap")

	// Test Add.
	addedValue := val.Add(10) // Assuming we can Add an integer to a uintptr for this hypothetical case
	expectedValue := comparedValue + 10
	require.Equal(t, expectedValue, addedValue, "Add should return the new value after addition")
	require.Equal(t, expectedValue, val.Load(), "Load should return the new value after Add")
}
