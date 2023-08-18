// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package rpcz is a tool that monitors the running state of RPC, recording various things that happen in a rpc,
// such as serialization/deserialization, compression/decompression, and the execution of filter,
// which can be applied to debug and performance optimization.
package rpcz

import (
	"crypto/rand"
	"encoding/binary"
)

// GlobalRPCZ to collect span, config by admin module.
// This global variable is unavoidable.
var GlobalRPCZ = NewRPCZ(&Config{Fraction: 0.0, Capacity: 1})

// RPCZ generates, samples and stores spans.
type RPCZ struct {
	shouldRecord ShouldRecord
	noopSpan
	idGenerator *randomIDGenerator
	sampler     *spanIDRatioSampler
	store       *spanStore
	exporter    SpanExporter
	enabled     bool
}

var _ recorder = (*RPCZ)(nil)

// NewRPCZ create a new RPCZ.
func NewRPCZ(cfg *Config) *RPCZ {
	var rngSeed int64
	_ = binary.Read(rand.Reader, binary.LittleEndian, &rngSeed)
	return &RPCZ{
		shouldRecord: cfg.shouldRecord(),
		idGenerator:  newRandomIDGenerator(rngSeed),
		sampler:      newSpanIDRatioSampler(cfg.Fraction),
		store:        newSpanStore(cfg.Capacity),
		exporter:     cfg.Exporter,
		enabled:      cfg.Fraction > 0.0,
	}
}

// NewChild creates a span, and returns RPCZ itself if rpcz isn't enabled.
// End this span if related operation is completed.
func (r *RPCZ) NewChild(name string) (Span, Ender) {
	if !r.enabled {
		return r, r
	}

	id := r.idGenerator.newSpanID()
	if !r.sampler.shouldSample(id) {
		s := noopSpan{}
		return s, s
	}
	s := newSpan(name, id, r)
	return s, s
}

// Query returns a span by ID.
func (r *RPCZ) Query(id SpanID) (*ReadOnlySpan, bool) {
	return r.store.query(id)
}

// BatchQuery return #num newly inserted span.
func (r *RPCZ) BatchQuery(num int) []*ReadOnlySpan {
	return r.store.batchQuery(num)
}

func (r *RPCZ) record(s *span) {
	if !r.shouldRecord(s) {
		return
	}

	if r.exporter != nil {
		r.exporter.Export(s.convertedToReadOnlySpan())
	}

	r.store.insert(s)
}
