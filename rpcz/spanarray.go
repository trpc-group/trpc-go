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

package rpcz

// spanArray is a non-empty circular array to store spans to avoid dynamic allocation of memory.
type spanArray struct {
	// assert(capacity > 0) should always be true.
	capacity uint32
	length   uint32
	data     []*span
	// head is the index for next dequeue.
	head uint32
	// tail is the index for next enqueue.
	tail uint32
}

// doBackward calls function f sequentially for each span present in the array
// in backward order (from tail to head) until f returns false.
func (a *spanArray) doBackward(f func(*span) bool) {
	iter := a.tail
	for i := uint32(0); i < a.length; i++ {
		iter = a.previousIndex(iter)
		if !f(a.data[iter]) {
			break
		}
	}
}

func (a *spanArray) enqueue(value *span) {
	if a.full() {
		a.dequeue()
	}
	a.data[a.tail] = value
	a.tail = a.nextIndex(a.tail)
	a.length++
}

func (a *spanArray) dequeue() {
	a.head = a.nextIndex(a.head)
	a.length--
}

func (a *spanArray) front() *span {
	return a.data[a.head]
}

func (a *spanArray) full() bool {
	return a.length >= a.capacity
}

func (a *spanArray) nextIndex(index uint32) uint32 {
	return (index + 1) % a.capacity
}

func (a *spanArray) previousIndex(index uint32) uint32 {
	return (index + a.capacity - 1) % a.capacity
}

func newSpanArray(capacity uint32) *spanArray {
	if capacity == 0 {
		panic("capacity should be greater than 0")
	}
	return &spanArray{
		capacity: capacity,
		data:     make([]*span, capacity),
	}
}
