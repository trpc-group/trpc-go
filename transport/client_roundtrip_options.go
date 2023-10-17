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
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

// RoundTripOptions is the options for one roundtrip.
type RoundTripOptions struct {
	Address               string // IP:Port. Note: address has been resolved from naming service.
	Password              string
	Network               string // tcp/udp
	LocalAddr             string // a random selected local address when accept a connection.
	DialTimeout           time.Duration
	Pool                  connpool.Pool // client connection pool
	ReqType               RequestType   // SendAndRecv, SendOnly
	FramerBuilder         codec.FramerBuilder
	ConnectionMode        ConnectionMode
	DisableConnectionPool bool // disable connection pool
	EnableMultiplexed     bool // enable multiplexed
	Multiplexed           multiplexed.Pool
	Msg                   codec.Msg
	Protocol              string // protocol type

	CACertFile    string // CA certificate file
	TLSCertFile   string // client certificate file
	TLSKeyFile    string // client key file
	TLSServerName string // the name when client verifies the server, default as HTTP hostname
}

// ConnectionMode is the connection mode, either Connected or NotConnected.
type ConnectionMode bool

// ConnectionMode of UDP.
const (
	Connected    = false // UDP which isolates packets from non-same path
	NotConnected = true  // UDP which allows returning packets from non-same path
)

// RequestType is the client request type, such as SendAndRecv or SendOnly.
type RequestType = codec.RequestType

// Request types.
const (
	SendAndRecv RequestType = codec.SendAndRecv // send and receive
	SendOnly    RequestType = codec.SendOnly    // send only
)

// RoundTripOption modifies the RoundTripOptions.
type RoundTripOption func(*RoundTripOptions)

// WithDialAddress returns a RoundTripOption which sets dial address.
func WithDialAddress(address string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Address = address
	}
}

// WithDialPassword returns a RoundTripOption which sets dial password.
func WithDialPassword(password string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Password = password
	}
}

// WithDialNetwork returns a RoundTripOption which sets dial network.
func WithDialNetwork(network string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Network = network
	}
}

// WithDialPool returns a RoundTripOption which sets dial pool.
func WithDialPool(pool connpool.Pool) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Pool = pool
	}
}

// WithClientFramerBuilder returns a RoundTripOption which sets FramerBuilder.
func WithClientFramerBuilder(builder codec.FramerBuilder) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.FramerBuilder = builder
	}
}

// WithReqType returns a RoundTripOption which sets request type.
func WithReqType(reqType RequestType) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.ReqType = reqType
	}
}

// WithConnectionMode returns a RoundTripOption which sets UDP connection mode.
func WithConnectionMode(connMode ConnectionMode) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.ConnectionMode = connMode
	}
}

// WithDialTLS returns a RoundTripOption which sets UDP TLS relatives.
func WithDialTLS(certFile, keyFile, caFile, serverName string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.TLSCertFile = certFile
		opts.TLSKeyFile = keyFile
		opts.CACertFile = caFile
		opts.TLSServerName = serverName
	}
}

// WithDisableConnectionPool returns a RoundTripOption which disables connection pool.
func WithDisableConnectionPool() RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.DisableConnectionPool = true
	}
}

// WithMultiplexed returns a RoundTripOption which enables multiplexed.
func WithMultiplexed(enable bool) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.EnableMultiplexed = enable
	}
}

// WithMultiplexedPool returns a RoundTripOption which sets multiplexed pool.
// This function also enables multiplexed.
func WithMultiplexedPool(p multiplexed.Pool) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.EnableMultiplexed = true
		opts.Multiplexed = p
	}
}

// WithMsg returns a RoundTripOption which sets msg.
func WithMsg(msg codec.Msg) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Msg = msg
	}
}

// WithLocalAddr returns a RoundTripOption which sets local address.
// Random selection by default when there are multiple NICs.
func WithLocalAddr(addr string) RoundTripOption {
	return func(o *RoundTripOptions) {
		o.LocalAddr = addr
	}
}

// WithDialTimeout returns a RoundTripOption which sets dial timeout.
func WithDialTimeout(dur time.Duration) RoundTripOption {
	return func(o *RoundTripOptions) {
		o.DialTimeout = dur
	}
}

// WithProtocol returns a RoundTripOption which sets protocol name, such as trpc.
func WithProtocol(s string) RoundTripOption {
	return func(o *RoundTripOptions) {
		o.Protocol = s
	}
}
