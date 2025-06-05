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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet

import (
	"context"
	"fmt"
	"net"

	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/tnet/tls"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	intertls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
	tnetmultiplexed "trpc.group/trpc-go/trpc-go/transport/tnet/multiplex"
)

func init() {
	transport.RegisterClientTransport(transportName, DefaultClientTransport)
}

// DefaultClientTransport is the default implementation of tnet client transport.
var DefaultClientTransport = NewClientTransport()

// DefaultConnPool is the default connection pool used by tnet.
//
// The HealthChecker used here checks tnet.Conn.IsActive() to determine if the connection is healthy.
// But tnet's own idle timeout will still not be used, only the trpc-go's own connpool connection management
// mechanism will be taking effect.
var DefaultConnPool = connpool.NewConnectionPool(
	connpool.WithDialFunc(Dial),
	connpool.WithAdditionalHealthChecker(HealthChecker),
)

// DefaultMultiplexedPool is default multiplexd pool used by tnet.
var DefaultMultiplexedPool = tnetmultiplexed.NewPool(Dial)

// NewConnectionPool creates a new connection pool. Use it instead
// of connpool.NewConnectionPool when use tnet transport because
// it will dial tnet connection, otherwise error will occur.
func NewConnectionPool(opts ...connpool.Option) connpool.Pool {
	// Users are allowed to provide a custom dial function with higher priority than the default options.
	// Therefore, if users provide custom options, the default dial function should be overwritten.
	// To achieve this, we append the custom options after the default ones and create a new connection pool.
	// The HealthChecker used here checks tnet.Conn.IsActive() to determine if the connection is healthy.
	// But tnet's own idle timeout will still not be used, only the trpc-go's own connpool connection management
	// mechanism will be taking effect.
	return connpool.NewConnectionPool(append([]connpool.Option{
		connpool.WithDialFunc(Dial),
		connpool.WithAdditionalHealthChecker(HealthChecker),
	}, opts...)...)
}

// NewMultiplexdPool creates a new multiplexd pool. Use it instead
// of multiplexed.NewPool when use tnet transport because it will dial tnet connection.
func NewMultiplexdPool(opts ...tnetmultiplexed.OptPool) multiplexed.Pool {
	return tnetmultiplexed.NewPool(Dial, opts...)
}

type clientTransport struct {
	opts *ClientTransportOptions
}

// NewClientTransport creates a tnet client transport.
func NewClientTransport(opts ...ClientTransportOption) transport.ClientTransport {
	option := &ClientTransportOptions{}
	for _, o := range opts {
		o(option)
	}
	return &clientTransport{opts: option}
}

// RoundTrip begins an RPC roundtrip.
func (c *clientTransport) RoundTrip(
	ctx context.Context,
	req []byte,
	opts ...transport.RoundTripOption,
) ([]byte, error) {
	return c.switchNetworkToRoundTrip(ctx, req, opts...)
}

func (c *clientTransport) switchNetworkToRoundTrip(
	ctx context.Context,
	req []byte,
	opts ...transport.RoundTripOption,
) ([]byte, error) {
	option, err := buildRoundTripOptions(opts...)
	if err != nil {
		return nil, err
	}
	if err := canUseTnet(option); err != nil {
		log.Trace("switch to gonet default transport, ", err)
		return transport.DefaultClientTransport.RoundTrip(ctx, req, opts...)
	}
	log.Tracef("roundtrip to: %s is using tnet transport, current number of pollers: %d",
		option.Address, tnet.NumPollers())
	if option.EnableMultiplexed {
		return c.multiplexed(ctx, req, option)
	}
	switch option.Network {
	case protocol.TCP, protocol.TCP4, protocol.TCP6:
		return c.tcpRoundTrip(ctx, req, option)
	case protocol.UDP, protocol.UDP4, protocol.UDP6:
		return c.udpRoundTrip(ctx, req, option)
	default:
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			fmt.Sprintf("tnet client transport, doesn't support network [%s]", option.Network))
	}
}

func buildRoundTripOptions(opts ...transport.RoundTripOption) (*transport.RoundTripOptions, error) {
	rtOpts := &transport.RoundTripOptions{
		Pool:        DefaultConnPool,
		Multiplexed: DefaultMultiplexedPool,
	}
	for _, o := range opts {
		o(rtOpts)
	}
	if rtOpts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail, "client transport: framer builder empty")
	}
	return rtOpts, nil
}

// Dial connects to the address on the named network.
func Dial(opts *connpool.DialOptions) (net.Conn, error) {
	if opts.CACertFile == "" {
		conn, err := tnet.DialTCP(opts.Network, opts.Address, opts.Timeout)
		if err != nil {
			return nil, err
		}
		// We do not call conn.SetIdleTimeout(opts.IdleTimeout) here because the connection will be constantly
		// triggered by tnet, resulting in the connection being closed when it reaches the idle timeout, even if
		// it is obtained from the pool. Unlike tnet, connpool is not part of the tnet framework, so once a
		// connection is established, it will not be affected by the idle timeout. However, if tnet applies
		// its own idle timeout to the connection, this timeout will always be in effect, even if you refresh it
		// immediately after obtaining the connection. Therefore, we can rely on connpool's existing idle connection
		// management mechanism instead of tnet's.
		return conn, nil
	}
	if opts.TLSServerName == "" {
		opts.TLSServerName = opts.Address
	}
	tlsConf, err := intertls.GetClientConfig(opts.TLSServerName, opts.CACertFile, opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, errs.WrapFrameError(err, errs.RetClientDecodeFail, "client dial tnet tls fail")
	}
	return tls.Dial(opts.Network, opts.Address,
		tls.WithClientTLSConfig(tlsConf),
		tls.WithTimeout(opts.Timeout),
		tls.WithClientIdleTimeout(opts.IdleTimeout),
	)
}

// HealthChecker checks if connection healthy or not.
func HealthChecker(pc *connpool.PoolConn, _ bool) bool {
	c := pc.GetRawConn()
	tc, ok := c.(tnet.Conn)
	if !ok {
		return true
	}
	return tc.IsActive()
}

func validateTnetConn(conn net.Conn) bool {
	if _, ok := conn.(tnet.Conn); ok {
		return true
	}
	pc, ok := conn.(*connpool.PoolConn)
	if !ok {
		return false
	}
	_, ok = pc.GetRawConn().(tnet.Conn)
	return ok
}

func validateTnetTLSConn(conn net.Conn) bool {
	if _, ok := conn.(tls.Conn); ok {
		return true
	}
	pc, ok := conn.(*connpool.PoolConn)
	if !ok {
		return false
	}
	_, ok = pc.GetRawConn().(tls.Conn)
	return ok
}

func canUseTnet(opts *transport.RoundTripOptions) error {
	switch opts.Network {
	case protocol.TCP, protocol.TCP4, protocol.TCP6:
	case protocol.UDP, protocol.UDP4, protocol.UDP6:
	default:
		return fmt.Errorf("tnet client transport doesn't support network [%s]", opts.Network)
	}
	return nil
}
