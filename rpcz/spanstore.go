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

import (
	"sync"
)

// spanStore stores spans that has ended its life.
type spanStore struct {
	sync.RWMutex // protects everything below.
	idSpans      map[SpanID]*span
	spans        *spanArray
}

// insert adds a span to spanStore.
func (ss *spanStore) insert(s *span) {
	ss.Lock()
	defer ss.Unlock()

	if ss.spans.full() {
		expiredSpan := ss.spans.front()
		ss.spans.dequeue()

		delete(ss.idSpans, expiredSpan.id)

		putSpanToPool(expiredSpan)
	}

	ss.spans.enqueue(s)
	ss.idSpans[s.ID()] = s
}

// query returns a ReadOnlySpan converted from span whose SpanID equals id.
func (ss *spanStore) query(id SpanID) (*ReadOnlySpan, bool) {
	ss.RLock()
	defer ss.RUnlock()
	s, ok := ss.idSpans[id]
	if !ok {
		return nil, false
	}
	return s.convertedToReadOnlySpan(), true
}

// batchQuery returns #num ReadOnlySpan converted from span that is newly inserted to spanStore.
func (ss *spanStore) batchQuery(num int) (spans []*ReadOnlySpan) {
	ss.RLock()
	ss.spans.doBackward(func(s *span) bool {
		if num <= 0 {
			return false
		}
		num--
		spans = append(spans, s.convertedToReadOnlySpan())
		return true
	})
	ss.RUnlock()
	return spans
}

func newSpanStore(capacity uint32) *spanStore {
	return &spanStore{
		idSpans: make(map[SpanID]*span),
		spans:   newSpanArray(capacity),
	}
}
