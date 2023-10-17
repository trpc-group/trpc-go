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

// Package stack provides a non-thread-safe stack.
package stack

// Stack is a non-thread-safe stack.
type Stack[T any] struct {
	top  *node[T]
	size int
}

type node[T any] struct {
	value T
	prev  *node[T]
}

// New creates a stack.
func New[T any]() *Stack[T] {
	return &Stack[T]{}
}

// Size returns the stack size.
func (st *Stack[T]) Size() int {
	return st.size
}

// Reset resets the stack.
func (st *Stack[T]) Reset() {
	st.top = nil
	st.size = 0
}

// Push pushes an element onto the stack.
func (st *Stack[T]) Push(value T) {
	newNode := &node[T]{
		value: value,
		prev:  st.top,
	}
	st.top = newNode
	st.size++
}

// Pop pops an element from the stack.
func (st *Stack[T]) Pop() (T, bool) {
	if st.size == 0 {
		var zero T
		return zero, false
	}
	topNode := st.top
	st.top = topNode.prev
	topNode.prev = nil
	st.size--
	return topNode.value, true
}

// Peek looks at the top element of the stack.
func (st *Stack[T]) Peek() (T, bool) {
	if st.size == 0 {
		var zero T
		return zero, false
	}
	return st.top.value, true
}
