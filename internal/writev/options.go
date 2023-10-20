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

package writev

// Options is the Buffer configuration.
type Options struct {
	handler    QuitHandler // Set the goroutine to exit the cleanup function.
	bufferSize int         // Set the length of each connection request queue.
	dropFull   bool        // Whether the queue is full or not.
}

// Option optional parameter.
type Option func(*Options)

// WithQuitHandler returns an Option which sets the Buffer goroutine exit handler.
func WithQuitHandler(handler QuitHandler) Option {
	return func(o *Options) {
		o.handler = handler
	}
}

// WithBufferSize returns an Option which sets the length of each connection request queue.
func WithBufferSize(size int) Option {
	return func(opts *Options) {
		opts.bufferSize = size
	}
}

// WithDropFull returns an Option which sets whether to drop the request when the queue is full.
func WithDropFull(drop bool) Option {
	return func(opts *Options) {
		opts.dropFull = drop
	}
}
