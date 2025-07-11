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

package metrics

// NoopSink defines the noop Sink.
type NoopSink struct{}

// Name returns noop.
func (n *NoopSink) Name() string {
	return "noop"
}

// Report does nothing.
func (n *NoopSink) Report(rec Record, opts ...Option) error {
	return nil
}
