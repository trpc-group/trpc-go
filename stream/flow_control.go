// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package stream

import (
	"errors"
	"reflect"
	"sync/atomic"
)

// sendControl is the behavior control of the sender.
type sendControl struct {
	// The largest window is defined as uint32, but there may be negative numbers, so it is int64.
	window int64
	// When sending window to less than or equal to 0, ch will block, when receiving window update, ch will unblock.
	ch chan struct{}
	// waits wait for data to arrive or the stream to end.
	waits []reflect.SelectCase
}

// feedback is the feedback type.
type feedback func(uint32) error

func newSendControl(window uint32, dones ...<-chan struct{}) *sendControl {
	s := &sendControl{
		window: int64(window),
		ch:     make(chan struct{}, 1),
	}
	s.waits = []reflect.SelectCase{{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(s.ch)}}
	for _, done := range dones {
		s.waits = append(s.waits, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(done)})
	}
	return s
}

// GetWindow gets the sending window of a certain size, if it can't get it, it will block.
// precision is not guaranteed, may be negative.
func (s *sendControl) GetWindow(w uint32) error {
	for w := int64(w); ; {
		// First determine the currently available port, if the available window is <= 0, wait for the window to update.
		// If it is greater than 0, subtract this window and return.
		// Note that it may become a negative number after subtraction.
		if atomic.LoadInt64(&s.window) > 0 {
			atomic.AddInt64(&s.window, -w)
			return nil
		}
		if chosen, _, _ := reflect.Select(s.waits); chosen == 0 {
			// received data
			continue
		}
		// stream closed
		return errors.New(streamClosed)
	}
}

// UpdateWindow is used to update the window after receiving the feedback.
func (s *sendControl) UpdateWindow(increment uint32) {
	updatedWindow := atomic.AddInt64(&s.window, int64(increment))
	if !checkUpdate(updatedWindow, int64(increment)) {
		return
	}
	select {
	// Signal a blocked get window request
	case s.ch <- struct{}{}:
	default:
	}
}

func checkUpdate(updatedWindow, increment int64) bool {
	return (updatedWindow-increment <= 0) && (updatedWindow > 0)
}

// receiveControl represents the flow control statistics from the perspective of the receiving end.
type receiveControl struct {
	buffer    uint32   // upper limit.
	unUpdated uint32   // Consumed, no window update sent.
	left      uint32   // remaining available buffer.
	fb        feedback // function for feedback.
}

func newReceiveControl(buffer uint32, fb feedback) *receiveControl {
	return &receiveControl{
		buffer: buffer,
		fb:     fb,
		left:   buffer,
	}
}

// OnRecv application is called when data is received, and the window is updated.
func (r *receiveControl) OnRecv(n uint32) error {
	r.unUpdated += n
	if r.unUpdated >= r.buffer/4 {
		increment := r.unUpdated
		r.unUpdated = 0
		r.updateLeft()
		if r.fb != nil {
			return r.fb(increment)
		}
		return nil
	}
	r.updateLeft()
	return nil
}

// updateLeft updates the remaining available buffers.
func (r *receiveControl) updateLeft() {
	atomic.StoreUint32(&r.left, r.buffer-r.unUpdated)
}
