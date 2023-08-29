// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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
