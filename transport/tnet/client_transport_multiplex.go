//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet

import (
	"fmt"

	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/tnet/tls"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

// DefaultMultiplexPool is default multiplexed pool used by tnet.
var DefaultMultiplexPool = multiplexed.NewPool(DialMultiplexConn)

// NewMultiplexPool creates a new multiplexed pool. Use it instead
// of multiplexed.NewPool when use tnet transport because it will
// dial tnet connection.
func NewMultiplexPool(opts ...multiplexed.OptPool) multiplexed.Pool {
	return multiplexed.NewPool(DialMultiplexConn, opts...)
}

// DialMultiplexConn creates a Conn for the multiplexed pool using the tnet as the underlying implementation.
func DialMultiplexConn(fp multiplexed.FrameParser, opts *connpool.DialOptions) (multiplexed.Conn, error) {
	conn, err := Dial(opts)
	if err != nil {
		return nil, err
	}
	switch c := conn.(type) {
	case tnet.Conn:
		return &tnetConn{Conn: c, frameParser: fp}, nil
	case tls.Conn:
		return &tlsConn{Conn: c, frameParser: fp}, nil
	}
	return nil, fmt.Errorf("dialed connection type %T does't implements tnet.Conn or tnet/tls.Conn", conn)
}

type tnetConn struct {
	tnet.Conn
	frameParser multiplexed.FrameParser
}

// Start starts background reading processes.
func (tc *tnetConn) Start(n multiplexed.Notifier) error {
	tc.Conn.SetOnRequest(func(conn tnet.Conn) error {
		vid, buf, err := tc.frameParser.Parse(conn)
		if err != nil {
			n.Close(err)
			return err
		}
		n.Dispatch(vid, buf)
		return nil
	})
	tc.Conn.SetOnClosed(func(_ tnet.Conn) error {
		n.Close(multiplexed.ErrConnClosed)
		return nil
	})
	return nil
}

type tlsConn struct {
	tls.Conn
	frameParser multiplexed.FrameParser
}

// Start starts background reading processes.
func (tc *tlsConn) Start(n multiplexed.Notifier) error {
	tc.Conn.SetOnRequest(func(conn tls.Conn) error {
		vid, buf, err := tc.frameParser.Parse(conn)
		if err != nil {
			n.Close(err)
			return err
		}
		n.Dispatch(vid, buf)
		return nil
	})
	tc.Conn.SetOnClosed(func(_ tls.Conn) error {
		n.Close(multiplexed.ErrConnClosed)
		return nil
	})
	return nil
}
