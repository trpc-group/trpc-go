// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package transport

import (
	"context"
	"sync"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

const (
	defaultMaxConcurrentStreams = 1000
	defaultMaxIdleConnsPerHost  = 2
)

// DefaultClientStreamTransport is the default client stream transport.
var DefaultClientStreamTransport = NewClientStreamTransport()

// NewClientStreamTransport creates a new ClientStreamTransport.
func NewClientStreamTransport(opts ...ClientStreamTransportOption) ClientStreamTransport {
	options := &cstOptions{
		maxConcurrentStreams: defaultMaxConcurrentStreams,
		maxIdleConnsPerHost:  defaultMaxIdleConnsPerHost,
	}
	for _, opt := range opts {
		opt(options)
	}
	t := &clientStreamTransport{
		// Map streamID to connection. On the client side, ensure that the streamID is
		// incremented and unique, otherwise the map of addr must be added.
		streamIDToConn: make(map[uint32]multiplexed.MuxConn),
		m:              &sync.RWMutex{},
		multiplexedPool: multiplexed.New(
			multiplexed.WithMaxVirConnsPerConn(options.maxConcurrentStreams),
			multiplexed.WithMaxIdleConnsPerHost(options.maxIdleConnsPerHost),
		),
	}
	return t
}

// cstOptions is the client stream transport options.
type cstOptions struct {
	maxConcurrentStreams int
	maxIdleConnsPerHost  int
}

// ClientStreamTransportOption sets properties of ClientStreamTransport.
type ClientStreamTransportOption func(*cstOptions)

// WithMaxConcurrentStreams sets the maximum concurrent streams in each TCP connection.
func WithMaxConcurrentStreams(n int) ClientStreamTransportOption {
	return func(opts *cstOptions) {
		opts.maxConcurrentStreams = n
	}
}

// WithMaxIdleConnsPerHost sets the maximum idle connections per host.
func WithMaxIdleConnsPerHost(n int) ClientStreamTransportOption {
	return func(opts *cstOptions) {
		opts.maxIdleConnsPerHost = n
	}
}

// clientStreamTransport keeps compatibility with the original client transport.
type clientStreamTransport struct {
	streamIDToConn  map[uint32]multiplexed.MuxConn
	m               *sync.RWMutex
	multiplexedPool multiplexed.Pool
}

// Init inits clientStreamTransport. It gets a connection from the multiplexing pool. A stream is
// corresponding to a virtual connection, which provides the interface for the stream.
func (c *clientStreamTransport) Init(ctx context.Context, roundTripOpts ...RoundTripOption) error {
	opts, err := c.getOptions(ctx, roundTripOpts...)
	if err != nil {
		return err
	}
	// If ctx has been canceled or timeout, just return.
	if ctx.Err() == context.Canceled {
		return errs.NewFrameError(errs.RetClientCanceled,
			"client canceled before tcp dial: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return errs.NewFrameError(errs.RetClientTimeout,
			"client timeout before tcp dial: "+ctx.Err().Error())
	}
	msg := opts.Msg
	streamID := msg.StreamID()

	getOpts := multiplexed.NewGetOptions()
	getOpts.WithVID(streamID)
	fp, ok := opts.FramerBuilder.(multiplexed.FrameParser)
	if !ok {
		return errs.NewFrameError(errs.RetClientConnectFail,
			"frame builder does not implement multiplexed.FrameParser")
	}
	getOpts.WithFrameParser(fp)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithLocalAddr(opts.LocalAddr)
	conn, err := opts.Multiplexed.GetMuxConn(ctx, opts.Network, opts.Address, getOpts)
	if err != nil {
		return errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport multiplexd pool: "+err.Error())
	}
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())
	c.m.Lock()
	c.streamIDToConn[streamID] = conn
	c.m.Unlock()
	return nil
}

// Send sends stream data and provides interface for stream.
func (c *clientStreamTransport) Send(ctx context.Context, req []byte, roundTripOpts ...RoundTripOption) error {
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	// StreamID is uniquely generated by stream client.
	c.m.RLock()
	cc := c.streamIDToConn[streamID]
	c.m.RUnlock()
	if cc == nil {
		return errs.NewFrameError(errs.RetServerSystemErr, "Connection is Closed")
	}
	if err := cc.Write(req); err != nil {
		return err
	}
	return nil
}

// Recv receives stream data and provides interface for stream.
func (c *clientStreamTransport) Recv(ctx context.Context, roundTripOpts ...RoundTripOption) ([]byte, error) {
	cc, err := c.getConnect(ctx, roundTripOpts...)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp client transport canceled before Write: "+ctx.Err().Error())
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client transport timeout before Write: "+ctx.Err().Error())
		}
	default:
	}
	return cc.Read()
}

// Close closes connections and cleans up.
func (c *clientStreamTransport) Close(ctx context.Context) {
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	c.m.Lock()
	defer c.m.Unlock()
	if conn, ok := c.streamIDToConn[streamID]; ok {
		conn.Close()
		delete(c.streamIDToConn, streamID)
	}
}

// getOptions inits RoundTripOptions and does some basic check.
func (c *clientStreamTransport) getOptions(ctx context.Context,
	roundTripOpts ...RoundTripOption) (*RoundTripOptions, error) {
	opts := &RoundTripOptions{
		Multiplexed: c.multiplexedPool,
	}

	// use roundTripOpts to modify opts.
	for _, o := range roundTripOpts {
		o(opts)
	}

	if opts.Multiplexed == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: multiplexd pool empty")
	}

	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: framer builder empty")
	}

	if opts.Msg == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: message empty")
	}
	return opts, nil
}

func (c *clientStreamTransport) getConnect(ctx context.Context,
	roundTripOpts ...RoundTripOption) (multiplexed.MuxConn, error) {
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	c.m.RLock()
	cc := c.streamIDToConn[streamID]
	c.m.RUnlock()
	if cc == nil {
		return nil, errs.NewFrameError(errs.RetServerSystemErr, "Stream is not inited yet")
	}
	return cc, nil
}
