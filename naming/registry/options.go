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

package registry

// Options is the node register options.
type Options struct {
	Address string
	Event   EventType
}

// Option modifies the Options.
type Option func(*Options)

// WithAddress returns an Option which sets the server address. The format of address is "IP:Port" or
// just ":Port".
func WithAddress(s string) Option {
	return func(opts *Options) {
		opts.Address = s
	}
}

// EventType defines the event types.
type EventType int

// GracefulRestart represents the hot restart event.
const GracefulRestart = EventType(iota)

// WithEvent returns an Option which sets the event type.
func WithEvent(e EventType) Option {
	return func(opts *Options) {
		opts.Event = e
	}
}
