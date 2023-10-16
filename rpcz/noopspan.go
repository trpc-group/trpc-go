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

import "time"

// noopSpan is an implementation of Span that preforms no operations.
type noopSpan struct{}

// AddEvent does nothing.
func (s noopSpan) AddEvent(_ string) {}

// Event returns nil.
func (s noopSpan) Event(string) (time.Time, bool) {
	return time.Time{}, false
}

// SetAttribute does nothing.
func (s noopSpan) SetAttribute(_ string, _ interface{}) {}

// Attribute always returns nil.
func (s noopSpan) Attribute(_ string) (interface{}, bool) {
	return nil, false
}

// StartTime returns the zero value of type Time.
func (s noopSpan) StartTime() time.Time {
	return time.Time{}
}

// EndTime returns the zero value of type Time.
func (s noopSpan) EndTime() time.Time {
	return time.Time{}
}

// ID return an invalid span ID.
func (s noopSpan) ID() SpanID { return nilSpanID }

// Name return an empty string.
func (s noopSpan) Name() string { return "" }

// NewChild returns a noopSpan and empty end function.
func (s noopSpan) NewChild(_ string) (Span, Ender) {
	ns := noopSpan{}
	return ns, ns
}

// Child always returns nil and false.
func (s noopSpan) Child(_ string) (Span, bool) {
	return nil, false
}

// End does nothing.
func (s noopSpan) End() {}
