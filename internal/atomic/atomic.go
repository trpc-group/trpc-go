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

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package atomic provides atomic data structures.
// This package exists due to the challenges encountered when upgrading the go directive from go1.18 to go1.20.
//
// Reference:
//
//	https://github.com/golang/go/blob/6bfaafd3c34325515e8ffbe7446b9beda3f49698/src/sync/atomic/type.go#L1
package atomic

import (
	"sync/atomic"
	"unsafe"
)

// A Bool is an atomic boolean value.
// The zero value is false.
type Bool struct {
	_ noCopy
	v uint32
}

// Load atomically loads and returns the value stored in x.
func (x *Bool) Load() bool { return atomic.LoadUint32(&x.v) != 0 }

// Store atomically stores val into x.
func (x *Bool) Store(val bool) { atomic.StoreUint32(&x.v, b32(val)) }

// Swap atomically stores new into x and returns the previous value.
func (x *Bool) Swap(new bool) (old bool) { return atomic.SwapUint32(&x.v, b32(new)) != 0 }

// CompareAndSwap executes the compare-and-swap operation for the boolean value x.
func (x *Bool) CompareAndSwap(old, new bool) (swapped bool) {
	return atomic.CompareAndSwapUint32(&x.v, b32(old), b32(new))
}

// b32 returns a uint32 0 or 1 representing b.
func b32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// For testing *Pointer[T]'s methods can be inlined.
// Keep in sync with cmd/compile/internal/test/inl_test.go:TestIntendedInlining.
var _ = &Pointer[int]{}

// A Pointer is an atomic pointer of type *T. The zero value is a nil *T.
type Pointer[T any] struct {
	// Mention *T in a field to disallow conversion between Pointer types.
	// See go.dev/issue/56603 for more details.
	// Use *T, not T, to avoid spurious recursive type definition errors.
	_ [0]*T

	_ noCopy
	v unsafe.Pointer
}

// Load atomically loads and returns the value stored in x.
func (x *Pointer[T]) Load() *T { return (*T)(atomic.LoadPointer(&x.v)) }

// Store atomically stores val into x.
func (x *Pointer[T]) Store(val *T) { atomic.StorePointer(&x.v, unsafe.Pointer(val)) }

// Swap atomically stores new into x and returns the previous value.
func (x *Pointer[T]) Swap(new *T) (old *T) {
	return (*T)(atomic.SwapPointer(&x.v, unsafe.Pointer(new)))
}

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *Pointer[T]) CompareAndSwap(old, new *T) (swapped bool) {
	return atomic.CompareAndSwapPointer(&x.v, unsafe.Pointer(old), unsafe.Pointer(new))
}

// An Int32 is an atomic int32. The zero value is zero.
type Int32 struct {
	_ noCopy
	v int32
}

// Load atomically loads and returns the value stored in x.
func (x *Int32) Load() int32 { return atomic.LoadInt32(&x.v) }

// Store atomically stores val into x.
func (x *Int32) Store(val int32) { atomic.StoreInt32(&x.v, val) }

// Swap atomically stores new into x and returns the previous value.
func (x *Int32) Swap(new int32) (old int32) { return atomic.SwapInt32(&x.v, new) }

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *Int32) CompareAndSwap(old, new int32) (swapped bool) {
	return atomic.CompareAndSwapInt32(&x.v, old, new)
}

// Add atomically adds delta to x and returns the new value.
func (x *Int32) Add(delta int32) (new int32) { return atomic.AddInt32(&x.v, delta) }

// An Int64 is an atomic int64. The zero value is zero.
type Int64 struct {
	_ noCopy
	_ align64
	v int64
}

// Load atomically loads and returns the value stored in x.
func (x *Int64) Load() int64 { return atomic.LoadInt64(&x.v) }

// Store atomically stores val into x.
func (x *Int64) Store(val int64) { atomic.StoreInt64(&x.v, val) }

// Swap atomically stores new into x and returns the previous value.
func (x *Int64) Swap(new int64) (old int64) { return atomic.SwapInt64(&x.v, new) }

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *Int64) CompareAndSwap(old, new int64) (swapped bool) {
	return atomic.CompareAndSwapInt64(&x.v, old, new)
}

// Add atomically adds delta to x and returns the new value.
func (x *Int64) Add(delta int64) (new int64) { return atomic.AddInt64(&x.v, delta) }

// A Uint32 is an atomic uint32. The zero value is zero.
type Uint32 struct {
	_ noCopy
	v uint32
}

// Load atomically loads and returns the value stored in x.
func (x *Uint32) Load() uint32 { return atomic.LoadUint32(&x.v) }

// Store atomically stores val into x.
func (x *Uint32) Store(val uint32) { atomic.StoreUint32(&x.v, val) }

// Swap atomically stores new into x and returns the previous value.
func (x *Uint32) Swap(new uint32) (old uint32) { return atomic.SwapUint32(&x.v, new) }

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *Uint32) CompareAndSwap(old, new uint32) (swapped bool) {
	return atomic.CompareAndSwapUint32(&x.v, old, new)
}

// Add atomically adds delta to x and returns the new value.
func (x *Uint32) Add(delta uint32) (new uint32) { return atomic.AddUint32(&x.v, delta) }

// A Uint64 is an atomic uint64. The zero value is zero.
type Uint64 struct {
	_ noCopy
	_ align64
	v uint64
}

// Load atomically loads and returns the value stored in x.
func (x *Uint64) Load() uint64 { return atomic.LoadUint64(&x.v) }

// Store atomically stores val into x.
func (x *Uint64) Store(val uint64) { atomic.StoreUint64(&x.v, val) }

// Swap atomically stores new into x and returns the previous value.
func (x *Uint64) Swap(new uint64) (old uint64) { return atomic.SwapUint64(&x.v, new) }

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *Uint64) CompareAndSwap(old, new uint64) (swapped bool) {
	return atomic.CompareAndSwapUint64(&x.v, old, new)
}

// Add atomically adds delta to x and returns the new value.
func (x *Uint64) Add(delta uint64) (new uint64) { return atomic.AddUint64(&x.v, delta) }

// A Uintptr is an atomic uintptr. The zero value is zero.
type Uintptr struct {
	_ noCopy
	v uintptr
}

// Load atomically loads and returns the value stored in x.
func (x *Uintptr) Load() uintptr { return atomic.LoadUintptr(&x.v) }

// Store atomically stores val into x.
func (x *Uintptr) Store(val uintptr) { atomic.StoreUintptr(&x.v, val) }

// Swap atomically stores new into x and returns the previous value.
func (x *Uintptr) Swap(new uintptr) (old uintptr) { return atomic.SwapUintptr(&x.v, new) }

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *Uintptr) CompareAndSwap(old, new uintptr) (swapped bool) {
	return atomic.CompareAndSwapUintptr(&x.v, old, new)
}

// Add atomically adds delta to x and returns the new value.
func (x *Uintptr) Add(delta uintptr) (new uintptr) { return atomic.AddUintptr(&x.v, delta) }

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// align64 may be added to structs that must be 64-bit aligned.
// This struct is recognized by a special case in the compiler
// and will not work if copied to any other package.
type align64 struct{}
