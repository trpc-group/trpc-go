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
	intertls "trpc.group/trpc-go/trpc-go/internal/tls"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/tnet/multiplex"
)

func init() {
	transport.RegisterClientTransport(transportName, DefaultClientTransport)
}

// DefaultClientTransport is the default implementation of tnet client transport.
var DefaultClientTransport = NewClientTransport()

// DefaultConnPool is default connection pool used by tnet.
var DefaultConnPool = connpool.NewConnectionPool(
	connpool.WithDialFunc(Dial),
	connpool.WithHealthChecker(HealthChecker),
)

// DefaultMuxPool is default muxtiplex pool used by tnet.
var DefaultMuxPool = multiplex.NewPool(Dial)

// NewConnectionPool creates a new connection pool. Use it instead
// of connpool.NewConnectionPool when use tnet transport because
// it will dial tnet connection, otherwise error will occur.
func NewConnectionPool(opts ...connpool.Option) connpool.Pool {
	opts = append(opts,
		connpool.WithDialFunc(Dial),
		connpool.WithHealthChecker(HealthChecker))
	return connpool.NewConnectionPool(opts...)
}

// NewMuxPool creates a new multiplexing pool. Use it instead
// of mux.NewPool when use tnet transport because it will dial tnet connection.
func NewMuxPool(opts ...multiplex.OptPool) multiplexed.Pool {
	return multiplex.NewPool(Dial, opts...)
}

type clientTransport struct{}

// NewClientTransport creates a tnet client transport.
func NewClientTransport() transport.ClientTransport {
	return &clientTransport{}
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
		log.Error("switch to gonet default transport, ", err)
		return transport.DefaultClientTransport.RoundTrip(ctx, req, opts...)
	}
	log.Tracef("roundtrip to:%s is using tnet transport, current number of pollers: %d",
		option.Address, tnet.NumPollers())
	if option.EnableMultiplexed {
		return c.multiplex(ctx, req, option)
	}
	switch option.Network {
	case "tcp", "tcp4", "tcp6":
		return c.tcpRoundTrip(ctx, req, option)
	default:
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			fmt.Sprintf("tnet client transport, doesn't support network [%s]", option.Network))
	}
}

func buildRoundTripOptions(opts ...transport.RoundTripOption) (*transport.RoundTripOptions, error) {
	rtOpts := &transport.RoundTripOptions{
		Pool:        DefaultConnPool,
		Multiplexed: DefaultMuxPool,
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
		if err := conn.SetIdleTimeout(opts.IdleTimeout); err != nil {
			return nil, err
		}
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
	case "tcp", "tcp4", "tcp6":
	default:
		return fmt.Errorf("tnet doesn't support network [%s]", opts.Network)
	}
	return nil
}
