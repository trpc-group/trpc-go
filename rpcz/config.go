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

package rpcz

// Config stores the config of rpcz.
type Config struct {
	Fraction     float64
	Capacity     uint32
	ShouldRecord ShouldRecord
	Exporter     SpanExporter
}

func (c *Config) shouldRecord() ShouldRecord {
	if c.ShouldRecord == nil {
		return AlwaysRecord
	}
	return c.ShouldRecord
}

// ShouldRecord determines if the Span should be recorded.
type ShouldRecord = func(Span) bool

// AlwaysRecord always records span.
func AlwaysRecord(_ Span) bool { return true }
