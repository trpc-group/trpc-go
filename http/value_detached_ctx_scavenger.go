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

package http

import (
	"reflect"

	"go.uber.org/atomic"
)

var globalScavenger *scavenger

func init() {
	globalScavenger = newScavenger()
	go globalScavenger.scavengeValueDetachedCtx()
}

type scavenger struct {
	inprocess atomic.Uint32
	incoming  chan *valueDetachedCtx
	cases     []reflect.SelectCase
	ctxs      []*valueDetachedCtx
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
		ctxs: []*valueDetachedCtx{{}},
	}
}

func (s *scavenger) collect(c *valueDetachedCtx) bool {
	if s.inprocess.Inc() >= limitedCases {
		s.inprocess.Dec()
		return false
	}
	s.incoming <- c
	return true
}

func (s *scavenger) scavengeValueDetachedCtx() {
	for {
		chosen, recv, recvOK := reflect.Select(s.cases)
		if chosen == indexOfIncomingChan {
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

		for i := chosen; i < len(s.ctxs)-1; i++ {
			s.cases[i] = s.cases[i+1]
			s.ctxs[i] = s.ctxs[i+1]
		}
		s.cases[len(s.cases)-1] = reflect.SelectCase{}
		s.ctxs[len(s.ctxs)-1] = nil
		s.cases = s.cases[:len(s.cases)-1]
		s.ctxs = s.ctxs[:len(s.ctxs)-1]

		const minSelectCasesCap = 1024
		currentCap := cap(s.cases)
		currentLen := len(s.cases)
		if currentLen < currentCap/2 && currentCap/2 >= minSelectCasesCap {
			newCap := currentCap / 2
			newCases := make([]reflect.SelectCase, currentLen, newCap)
			newCtxs := make([]*valueDetachedCtx, currentLen, newCap)
			copy(newCases, s.cases)
			copy(newCtxs, s.ctxs)
			s.cases = newCases
			s.ctxs = newCtxs
		}
		s.inprocess.Dec()
	}
}
