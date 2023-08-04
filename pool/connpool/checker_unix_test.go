package connpool

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/codec"
)

const network = "tcp"

func TestRemoteEOF(t *testing.T) {
	var s server
	require.Nil(t, s.init())

	p := NewConnectionPool(
		WithDialFunc(func(opts *DialOptions) (net.Conn, error) {
			return net.Dial(opts.Network, opts.Address)
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(network, s.addr, GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)

	clientConn := pc.(*PoolConn).GetRawConn()
	serverConn := <-s.serverConns

	require.Nil(t, serverConn.Close())
	buf := make([]byte, 1)
	require.Eventually(t, func() bool {
		return errors.Is(checkConnErr(clientConn, buf), io.EOF)
	}, time.Second, time.Millisecond)
	require.Nil(t, pc.Close())
}

func TestUnexceptedRead(t *testing.T) {
	var s server
	require.Nil(t, s.init())

	p := NewConnectionPool(
		WithDialFunc(func(opts *DialOptions) (net.Conn, error) {
			return net.Dial(opts.Network, opts.Address)
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pc, err := p.Get(network, s.addr, GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)

	clientConn := pc.(*PoolConn).GetRawConn()
	serverConn := <-s.serverConns

	require.Nil(t, pc.Close())
	data := []byte("test")
	n, err := serverConn.Write(data)
	require.Nil(t, err)
	require.Equal(t, len(data), n)

	buf := make([]byte, 1)
	require.Eventually(t, func() bool {
		return strings.Contains(
			checkConnErr(clientConn, buf).Error(),
			ErrUnexpectedRead.Error())
	}, time.Second, time.Millisecond)
	require.Nil(t, serverConn.Close())
}

func TestEAGAIN(t *testing.T) {
	var s server
	require.Nil(t, s.init())

	p := NewConnectionPool(
		WithDialFunc(func(opts *DialOptions) (net.Conn, error) {
			return net.Dial(opts.Network, opts.Address)
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(network, s.addr, GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)

	clientConn := pc.(*PoolConn).GetRawConn()

	buf := make([]byte, 100)
	err2 := checkConnErr(clientConn, buf)
	require.Nil(t, err2)

	require.Nil(t, pc.Close())
	require.Nil(t, (<-s.serverConns).Close())
}

type server struct {
	serverConns chan net.Conn
	addr        string
}

func (s *server) init() error {
	s.serverConns = make(chan net.Conn)

	l, err := net.Listen(network, ":0")
	if err != nil {
		return err
	}
	s.addr = l.Addr().String()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}
			s.serverConns <- conn
		}
	}()

	return nil
}
