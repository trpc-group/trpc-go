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

package circuitbreaker

import (
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/internal/lru"
)

// NewLRUCircuitBreakers creates a new LRUCircuitBreakers.
func NewLRUCircuitBreakers(opts ...Opt) *LRUCircuitBreakers {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}

	var newClosed func(*tfsw) *cbClosed
	var newOpened func(*tfsw) *cbOpened
	var newHalfOpened func(*tfsw) *cbHalfOpened
	newClosed = func(sw *tfsw) *cbClosed {
		return &cbClosed{
			sw:                      sw,
			minRequests:             o.minRequestsToOpen,
			errRate:                 o.errRateToOpen,
			continuousFailThreshold: o.continuousFailuresToOpen,
			newOpened:               func() *cbOpened { return newOpened(sw) },
		}
	}
	newOpened = func(sw *tfsw) *cbOpened {
		return &cbOpened{
			sw:            sw,
			until:         time.Now().Add(o.openDuration),
			newHalfOpened: func() *cbHalfOpened { return newHalfOpened(sw) },
		}
	}
	newHalfOpened = func(sw *tfsw) *cbHalfOpened {
		return &cbHalfOpened{
			sw: sw,
			// the Available request, which returns ok and converts from opened to halfOpened, should be counted.
			total:      1,
			maxTotal:   o.totalRequestsToClose,
			minSuccess: o.successRequestsToClose,
			newOpened:  func() *cbOpened { return newOpened(sw) },
			newClosed:  func() *cbClosed { return newClosed(sw) },
		}
	}
	return (*LRUCircuitBreakers)(lru.NewLRU(
		(o.openDuration+o.slidingWindowInterval)*2,
		func() *circuitBreaker {
			return &circuitBreaker{cb: newClosed(newSlidingWindow(
				func() totalAndFailures { return totalAndFailures{} },
				o.slidingWindowInterval,
				o.slidingWindowSize))}
		}))
}

// LRUCircuitBreakers is a group of circuitBreaker which is managed by LRU cache.
type LRUCircuitBreakers lru.LRU[*circuitBreaker]

// Available indicates whether the given addr is not affected by the circuit breaker.
func (cbs *LRUCircuitBreakers) Available(addr string) bool {
	return (*lru.LRU[*circuitBreaker])(cbs).Get(addr).Available()
}

// Report reports a calling status of addr.
func (cbs *LRUCircuitBreakers) Report(addr string, ok bool) {
	(*lru.LRU[*circuitBreaker])(cbs).Get(addr).Report(ok)
}

// circuitBreaker is an implementation of three phases circuit breaker.
type circuitBreaker struct {
	mu sync.Mutex
	cb cb
}

// Available returns whether it is available.
func (cb *circuitBreaker) Available() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	ok, nextCB := cb.cb.Available()
	cb.cb = nextCB
	return ok
}

// Report reports a calling status.
func (cb *circuitBreaker) Report(ok bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.cb.Report(ok)
}

// cb is an abstract interface which is implemented by each phase of circuit breaker.
// As Report is not a return value of Available, there may be instances in half opened
// phase where Report(s) is more than Available(s). Nevertheless, this can be mitigated
// with a sufficiently long opened phase.
type cb interface {
	Available() (bool, cb)
	Report(bool)
}

// cbClosed is the closed phase.
// The continuousFail counts longer than slidingWindow.
type cbClosed struct {
	sw             *tfsw
	continuousFail int

	minRequests             int
	errRate                 float64
	continuousFailThreshold int

	newOpened func() *cbOpened
}

// Available returns whether it is available and gives the next phase.
// The next phase can be closed itself or opened.
func (cb *cbClosed) Available() (bool, cb) {
	tf := cb.sw.Get()
	if tf.Total() < cb.minRequests {
		return true, cb
	}
	if float64(tf.Failures())/float64(tf.Total()) < cb.errRate &&
		cb.continuousFail < cb.continuousFailThreshold {
		return true, cb
	}
	return false, cb.newOpened()
}

// Report reports an error.
func (cb *cbClosed) Report(ok bool) {
	report(cb.sw, ok)
	if ok {
		cb.continuousFail = 0
	} else {
		cb.continuousFail++
	}
}

// cbOpened is the opened phase.
type cbOpened struct {
	sw    *tfsw
	until time.Time

	newHalfOpened func() *cbHalfOpened
}

// Available returns whether it is available and gives next phase.
// The next phase can be opened itself or half opened.
func (cb *cbOpened) Available() (bool, cb) {
	if time.Now().After(cb.until) {
		return true, cb.newHalfOpened()
	}
	return false, cb
}

// Report reports an error.
func (cb *cbOpened) Report(ok bool) {
	report(cb.sw, ok)
}

// cbHalfOpened is the half opened phase.
type cbHalfOpened struct {
	sw      *tfsw
	total   int
	success int
	failure int

	maxTotal   int
	minSuccess int

	newOpened func() *cbOpened
	newClosed func() *cbClosed
}

// Available returns whether it is available and gives the next phase.
// The next phase can be half opened itself, opened or closed.
// Unlike cbClosed and cbOpened, this function changes the status of cbHalfOpened.
// User should Report for each Available.
func (cb *cbHalfOpened) Available() (bool, cb) {
	if cb.success >= cb.minSuccess {
		return true, cb.newClosed()
	}
	if cb.failure > cb.maxTotal-cb.minSuccess {
		return false, cb.newOpened()
	}
	if cb.total < cb.maxTotal {
		cb.total++
		return true, cb
	}
	return false, cb
}

// Report reports an error.
func (cb *cbHalfOpened) Report(ok bool) {
	report(cb.sw, ok)
	if ok {
		cb.success++
	} else {
		cb.failure++
	}
}

func report(sw *tfsw, ok bool) {
	if ok {
		sw.Add(totalAndFailures{1, 0})
	} else {
		sw.Add(totalAndFailures{1, 1})
	}
}

type tfsw = slidingWindow[totalAndFailures]

func newSlidingWindow[G group[G]](
	newEmpty func() G,
	interval time.Duration,
	bucketSize int,
) *slidingWindow[G] {
	values := make([]G, 0, bucketSize)
	for i := 0; i < bucketSize; i++ {
		values = append(values, newEmpty())
	}
	return &slidingWindow[G]{
		total:     newEmpty(),
		values:    values,
		lastStart: time.Now(),
		interval:  interval / time.Duration(bucketSize),
	}
}

type slidingWindow[G group[G]] struct {
	total  G
	values []G

	idx       int
	lastStart time.Time
	interval  time.Duration
}

// group is mathematical term. The value of sliding windows forms a group.
type group[T any] interface {
	op(T) T
	empty() T
	inverse() T
}

func (sw *slidingWindow[G]) Add(val G) {
	for elapsed := time.Since(sw.lastStart); elapsed > sw.interval; elapsed -= sw.interval {
		sw.idx++
		if sw.idx >= len(sw.values) {
			sw.idx = 0
		}
		sw.lastStart = sw.lastStart.Add(sw.interval)
		sw.total = sw.total.op(sw.values[sw.idx].inverse())
		sw.values[sw.idx] = sw.values[sw.idx].empty()
	}
	sw.total = sw.total.op(val)
	sw.values[sw.idx] = sw.values[sw.idx].op(val)
}

func (sw *slidingWindow[G]) Get() G {
	return sw.total
}

type totalAndFailures struct {
	total    int
	failures int
}

func (tf totalAndFailures) op(g totalAndFailures) totalAndFailures {
	tf.total += g.total
	tf.failures += g.failures
	return tf
}

func (tf totalAndFailures) empty() totalAndFailures {
	return totalAndFailures{}
}

func (tf totalAndFailures) inverse() totalAndFailures {
	tf.total = -tf.total
	tf.failures = -tf.failures
	return tf
}

func (tf totalAndFailures) Total() int {
	return tf.total
}

func (tf totalAndFailures) Failures() int {
	return tf.failures
}
