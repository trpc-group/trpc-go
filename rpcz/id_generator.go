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
	"math/rand"
	"sync"
)

// randomIDGenerator generates random span ID.
type randomIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

// newSpanID returns a non-negative span ID randomly.
func (gen *randomIDGenerator) newSpanID() SpanID {
	gen.Lock()
	defer gen.Unlock()
	return SpanID(gen.randSource.Int63())
}

func newRandomIDGenerator(seed int64) *randomIDGenerator {
	return &randomIDGenerator{randSource: rand.New(rand.NewSource(seed))}
}
