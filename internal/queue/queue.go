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

// Package queue implements infinite queue, supporting blocking data acquisition.
package queue

import (
	"container/list"
	"sync"
)

// Queue uses list and channel to achieve blocking acquisition and infinite queue.
type Queue[T any] struct {
	list    *list.List
	notify  chan struct{}
	mu      sync.Mutex
	waiting bool
	done    <-chan struct{}
}

// New initializes a queue, dones is used to notify Queue.Get() from blocking.
func New[T any](done <-chan struct{}) *Queue[T] {
	q := &Queue[T]{
		list:   list.New(),
		notify: make(chan struct{}, 1),
		done:   done,
	}
	return q
}

// Put puts an element into the queue.
// Put and Get can be concurrent, multiple Put can be concurrent.
func (q *Queue[T]) Put(v T) {
	var wakeUp bool
	q.mu.Lock()
	if q.waiting {
		wakeUp = true
		q.waiting = false
	}
	q.list.PushBack(v)
	q.mu.Unlock()
	if wakeUp {
		select {
		case q.notify <- struct{}{}:
		default:
		}
	}
}

// Get gets an element from the queue, blocking if there is no content.
// Put and Get can be concurrent, but not concurrent Get.
// If done channel notify it from blocking, it will return false.
func (q *Queue[T]) Get() (T, bool) {
	for {
		q.mu.Lock()
		if e := q.list.Front(); e != nil {
			q.list.Remove(e)
			q.mu.Unlock()
			return e.Value.(T), true
		}
		q.waiting = true
		q.mu.Unlock()
		select {
		case <-q.notify:
			continue
		case <-q.done:
			var zero T
			return zero, false
		}
	}
}
