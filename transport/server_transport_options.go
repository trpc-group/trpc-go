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

package transport

import (
	"runtime"
	"time"
)

const (
	defaultRecvMsgChannelSize      = 100
	defaultSendMsgChannelSize      = 100
	defaultRecvUDPPacketBufferSize = 65536
	defaultIdleTimeout             = time.Minute
)

// ServerTransportOptions is options of the server transport.
type ServerTransportOptions struct {
	RecvMsgChannelSize      int
	SendMsgChannelSize      int
	RecvUDPPacketBufferSize int
	RecvUDPRawSocketBufSize int
	IdleTimeout             time.Duration
	KeepAlivePeriod         time.Duration
	ReusePort               bool
}

// ServerTransportOption modifies the ServerTransportOptions.
type ServerTransportOption func(*ServerTransportOptions)

// WithRecvMsgChannelSize returns a ServerTransportOption which sets the size of receive buf of
// ServerTransport TCP.
func WithRecvMsgChannelSize(size int) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.RecvMsgChannelSize = size
	}
}

// WithReusePort returns a ServerTransportOption which enable reuse port or not.
func WithReusePort(reuse bool) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.ReusePort = reuse
		if runtime.GOOS == "windows" {
			options.ReusePort = false
		}
	}
}

// WithSendMsgChannelSize returns a ServerTransportOption which sets the size of sendCh of
// ServerTransport TCP.
func WithSendMsgChannelSize(size int) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.SendMsgChannelSize = size
	}
}

// WithRecvUDPPacketBufferSize returns a ServerTransportOption which sets the pre-allocated buffer
// size of ServerTransport UDP.
func WithRecvUDPPacketBufferSize(size int) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.RecvUDPPacketBufferSize = size
	}
}

// WithRecvUDPRawSocketBufSize returns a ServerTransportOption which sets the size of the operating
// system's receive buffer associated with the UDP connection.
func WithRecvUDPRawSocketBufSize(size int) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.RecvUDPRawSocketBufSize = size
	}
}

// WithIdleTimeout returns a ServerTransportOption which sets the server connection idle timeout.
func WithIdleTimeout(timeout time.Duration) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.IdleTimeout = timeout
	}
}

// WithKeepAlivePeriod returns a ServerTransportOption which sets the period to keep TCP connection
// alive.
// It's not available for TLS, since TLS neither use net.TCPConn nor net.Conn.
func WithKeepAlivePeriod(d time.Duration) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.KeepAlivePeriod = d
	}
}

func defaultServerTransportOptions() *ServerTransportOptions {
	return &ServerTransportOptions{
		RecvMsgChannelSize:      defaultRecvMsgChannelSize,
		SendMsgChannelSize:      defaultSendMsgChannelSize,
		RecvUDPPacketBufferSize: defaultRecvUDPPacketBufferSize,
	}
}
