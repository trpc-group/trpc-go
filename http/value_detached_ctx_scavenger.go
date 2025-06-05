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

package http

import (
	"reflect"

	iatomic "trpc.group/trpc-go/trpc-go/internal/atomic"
)

var globalScavenger *scavenger

func init() {
	globalScavenger = newScavenger()
	go globalScavenger.scavengeValueDetachedCtx()
}

type scavenger struct {
	// inprocess is the number of contexts that are being processed.
	inprocess iatomic.Uint32
	incoming  chan *valueDetachedCtx
	// The first select case is incoming channel.
	// Others are the ctx.Done() channels.
	cases []reflect.SelectCase
	// The first element is a nil pointer which serves as a placeholder.
	// Others are the value detached contexts which holds the original contexts.
	// Each of these ctxs corresponds to one of the above select cases.
	ctxs []*valueDetachedCtx
}

const (
	incomingChanSize    = 1024
	indexOfIncomingChan = 0
)

var limitedCases uint32 = 50000

func newScavenger() *scavenger {
	incoming := make(chan *valueDetachedCtx, incomingChanSize)
	return &scavenger{
		incoming: incoming,
		cases: []reflect.SelectCase{
			{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(incoming),
			},
		},
		ctxs: []*valueDetachedCtx{{}}, // Initialize with a nil to reserve an index.
	}
}

func (s *scavenger) collect(c *valueDetachedCtx) bool {
	if s.inprocess.Add(1) >= limitedCases {
		// Decrease the inprocess count since it has reached the limit.
		s.inprocess.Add(^uint32(0))
		return false
	}
	s.incoming <- c
	return true
}

func (s *scavenger) scavengeValueDetachedCtx() {
	for {
		chosen, recv, recvOK := reflect.Select(s.cases)
		if chosen == indexOfIncomingChan { // New context is added.
			if !recvOK {
				continue
			}
			in := recv.Interface().(*valueDetachedCtx)
			s.cases = append(s.cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(in.ctx.Done()),
			})
			s.ctxs = append(s.ctxs, in)
			continue
		}
		// One of the old context's context done is reached.
		c := s.ctxs[chosen]
		ctx := c.ctx
		deadline, ok := ctx.Deadline()
		c.mu.Lock()
		c.ctx = &ctxRemnant{
			deadline:    deadline,
			hasDeadline: ok,
			err:         ctx.Err(),
			done:        ctx.Done(),
		}
		c.mu.Unlock()
		// Remove the context that has triggered context done.
		for i := chosen; i < len(s.ctxs)-1; i++ {
			s.cases[i] = s.cases[i+1]
			s.ctxs[i] = s.ctxs[i+1]
		}
		// The following detaches are necessary, or else the slices would still
		// hold references to the underlying data.
		s.cases[len(s.cases)-1] = reflect.SelectCase{}
		s.ctxs[len(s.ctxs)-1] = nil
		s.cases = s.cases[:len(s.cases)-1]
		s.ctxs = s.ctxs[:len(s.ctxs)-1]

		// Shrink capacity if length is less than half of capacity.
		// But don't shrink below minSelectCasesCap to avoid frequent reallocations.
		const minSelectCasesCap = 1024
		currentCap := cap(s.cases)
		currentLen := len(s.cases)
		if currentLen < currentCap/2 && currentCap/2 >= minSelectCasesCap {
			// Create new slices with reduced capacity but not less than minSelectCasesCap.
			newCap := currentCap / 2
			newCases := make([]reflect.SelectCase, currentLen, newCap)
			newCtxs := make([]*valueDetachedCtx, currentLen, newCap)
			copy(newCases, s.cases)
			copy(newCtxs, s.ctxs)
			s.cases = newCases
			s.ctxs = newCtxs
		}

		// Decrease the inprocess count.
		s.inprocess.Add(^uint32(0))
	}
}
