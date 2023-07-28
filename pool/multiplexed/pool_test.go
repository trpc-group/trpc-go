// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	. "trpc.group/trpc-go/trpc-go/pool/multiplexed"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestMultiplexedSuite(t *testing.T) {
	suite.Run(t, &msuite{})
}

type msuite struct {
	suite.Suite

	network    string
	udpNetwork string
	address    string
	udpAddr    string

	ts *tcpServer
	us *udpServer

	requestID uint32
}

func (s *msuite) SetupSuite() {
	s.ts = newTCPServer()
	s.us = newUDPServer()

	ctx := context.Background()
	s.ts.start(ctx)
	s.us.start(ctx)

	s.address = s.ts.ln.Addr().String()
	s.network = s.ts.ln.Addr().Network()

	s.udpAddr = s.us.conn.LocalAddr().String()
	s.udpNetwork = s.us.conn.LocalAddr().Network()

	s.requestID = 1
}

func (s *msuite) TearDownSuite() {
	s.ts.stop()
	s.us.stop()
}

func (s *msuite) TearDownTest() {
	// Close all the established tcp concreteConns after each test.
	s.ts.closeConnections()
}

var errDecodeDelimited = errors.New("decode error")

type lengthDelimitedFramer struct {
	IsStream    bool
	reader      io.Reader
	decodeError bool
	safe        bool
}

func (f *lengthDelimitedFramer) New(reader io.Reader) codec.Framer {
	return &lengthDelimitedFramer{
		IsStream:    f.IsStream,
		reader:      reader,
		decodeError: f.decodeError,
		safe:        f.safe,
	}
}

func (f *lengthDelimitedFramer) ReadFrame() ([]byte, error) {
	return nil, nil
}

func (f *lengthDelimitedFramer) IsSafe() bool {
	return f.safe
}

func (f *lengthDelimitedFramer) Parse(r io.Reader) (vid uint32, buf []byte, err error) {
	head := make([]byte, 8)
	num, err := io.ReadFull(r, head)
	if err != nil {
		return 0, nil, err
	}

	if f.decodeError {
		return 0, nil, errDecodeDelimited
	}

	if num != 8 {
		return 0, nil, errors.New("invalid read full num")
	}

	n := binary.BigEndian.Uint32(head[:4])
	requestID := binary.BigEndian.Uint32(head[4:8])
	body := make([]byte, int(n))

	num, err = io.ReadFull(r, body)
	if err != nil {
		return 0, nil, err
	}

	if num != int(n) {
		return 0, nil, errors.New("invalid read full body")
	}

	if f.IsStream {
		return requestID, append(head, body...), nil
	}
	return requestID, body, nil
}

type delimitedRequest struct {
	requestID uint32
	body      []byte
}

func encode(req *delimitedRequest) ([]byte, error) {
	l := len(req.body)
	buf := bytes.NewBuffer(make([]byte, 0, 8+l))
	if err := binary.Write(buf, binary.BigEndian, uint32(l)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, req.requestID); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, req.body); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *msuite) TestMultiplexedDecodeErr() {
	tests := []struct {
		network string
		address string
		wantErr error
	}{
		{s.network, s.address, errDecodeDelimited},
		{s.udpNetwork, s.udpAddr, context.DeadlineExceeded},
	}

	for _, tt := range tests {
		id := atomic.AddUint32(&s.requestID, 1)
		ld := &lengthDelimitedFramer{
			decodeError: true,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		m := NewPool(NewDialFunc())
		opts := NewGetOptions()
		opts.WithVID(id)
		opts.WithFrameParser(ld)
		vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
		require.Nil(s.T(), err)
		body := []byte("hello world")
		buf, err := encode(&delimitedRequest{
			body:      body,
			requestID: id,
		})
		require.Nil(s.T(), err)
		require.Nil(s.T(), vc.Write(buf))
		_, err = vc.Read()
		require.ErrorIs(s.T(), err, tt.wantErr)
		cancel()
	}
}

func (s *msuite) TestMultiplexedGetConcurrent() {
	count := 10
	ld := &lengthDelimitedFramer{}
	m := NewPool(NewDialFunc())
	tests := []struct {
		network string
		address string
	}{
		{s.network, s.address},
		{s.udpNetwork, s.udpAddr},
	}
	for _, tt := range tests {
		wg := sync.WaitGroup{}
		wg.Add(count)
		for i := 0; i < count; i++ {
			go func(i int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				id := atomic.AddUint32(&s.requestID, 1)
				opts := NewGetOptions()
				opts.WithVID(id)
				opts.WithFrameParser(ld)
				vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
				require.Nil(s.T(), err)
				body := []byte("hello world" + strconv.Itoa(i))
				buf, err := encode(&delimitedRequest{
					body:      body,
					requestID: id,
				})
				require.Nil(s.T(), err)
				require.Nil(s.T(), vc.Write(buf))
				rsp, err := vc.Read()
				require.Nil(s.T(), err)
				require.Equal(s.T(), rsp, body)
				cancel()
			}(i)
		}
		wg.Wait()
	}
}

func (s *msuite) TestMultiplexedGet() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	require.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	require.Nil(s.T(), err)
	require.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	require.Nil(s.T(), err)
	require.Equal(s.T(), rsp, body)
}

func (s *msuite) TestMultiplexedGetWithSafeFramer() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{safe: true}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	require.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	require.Nil(s.T(), err)
	require.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	require.Nil(s.T(), err)
	require.Equal(s.T(), rsp, body)
}

func (s *msuite) TestNoFramerParser() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	_, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	require.Equal(s.T(), ErrFrameParserNil, err)
}

func (s *msuite) TestContextDeadline() {
	opts := NewGetOptions()
	opts.WithFrameParser(&lengthDelimitedFramer{})
	m := NewPool(NewDialFunc())
	s.T().Run("deadline exceed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		opts.WithVID(atomic.AddUint32(&s.requestID, 1))
		vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
		require.Nil(s.T(), err)
		_, err = vc.Read()
		require.ErrorIs(s.T(), err, context.DeadlineExceeded)
	})

	s.T().Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		id := atomic.AddUint32(&s.requestID, 1)
		opts.WithVID(id)
		vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
		require.Nil(s.T(), err)
		body := []byte("hello world")
		buf, err := encode(&delimitedRequest{
			body:      body,
			requestID: id,
		})
		require.Nil(s.T(), err)
		require.Nil(s.T(), vc.Write(buf))

		rsp, err := vc.Read()
		require.Nil(s.T(), err)
		require.Equal(s.T(), rsp, body)
	})
}

func (s *msuite) TestContextCancel() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	// get with cancel.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	_, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	require.NotNil(s.T(), err)
}

func (s *msuite) TestUdpMultiplexedReadTimeout() {
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(ld)
	vc, err := m.GetVirtualConn(ctx, "udp", s.udpAddr, opts)
	require.Nil(s.T(), err)
	_, err = vc.Read()
	require.ErrorIs(s.T(), err, ctx.Err())
}

func (s *msuite) TestWithLocalAddr() {
	tests := []struct {
		network string
		address string
	}{
		{s.network, s.address},
		{s.udpNetwork, s.udpAddr},
	}
	localAddr := "127.0.0.1:18000"

	for _, tt := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m := NewPool(NewDialFunc())
		opts := NewGetOptions()
		opts.WithVID(atomic.AddUint32(&s.requestID, 1))
		opts.WithLocalAddr(localAddr)
		ld := &lengthDelimitedFramer{}
		opts.WithFrameParser(ld)
		vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
		require.Nil(s.T(), err)
		require.Equal(s.T(), localAddr, vc.LocalAddr().String())
	}
}

func (s *msuite) TestStreamMultiplexd() {
	id := atomic.AddUint32(&s.requestID, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(id)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), vc)

	body := []byte("hello world")
	buf, err := encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	require.Nil(s.T(), err)
	require.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	require.Nil(s.T(), err)
	require.Equal(s.T(), buf, rsp)
}

func (s *msuite) TestStreamMultiplexd_Addr() {
	streamID := atomic.AddUint32(&s.requestID, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := NewPool(NewDialFunc())
	opts := NewGetOptions()
	opts.WithVID(streamID)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), vc)
	time.Sleep(50 * time.Millisecond)

	la := vc.LocalAddr()
	require.NotNil(s.T(), la)

	ra := vc.RemoteAddr()
	require.Equal(s.T(), s.address, ra.String())
}

func (s *msuite) TestStreamMultiplexd_MaxVirtualConnPerConn() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := NewPool(NewDialFunc(), WithMaxConcurrentVirtualConnsPerConn(2))
	opts := NewGetOptions()
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)
	var vcs []VirtualConn
	for i := 0; i < 6; i++ {
		id := atomic.AddUint32(&s.requestID, 1)
		opts.WithVID(id)
		vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
		require.Nil(s.T(), err)
		vcs = append(vcs, vc)
	}
	require.Equal(s.T(), vcs[0].LocalAddr(), vcs[1].LocalAddr())
	require.Equal(s.T(), vcs[2].LocalAddr(), vcs[3].LocalAddr())
	require.Equal(s.T(), vcs[4].LocalAddr(), vcs[5].LocalAddr())
	require.NotEqual(s.T(), vcs[0].LocalAddr(), vcs[2].LocalAddr())
	require.NotEqual(s.T(), vcs[0].LocalAddr(), vcs[4].LocalAddr())
	require.NotEqual(s.T(), vcs[2].LocalAddr(), vcs[4].LocalAddr())
}

func newTCPServer() *tcpServer {
	return &tcpServer{}
}

type tcpServer struct {
	cancel        context.CancelFunc
	ln            net.Listener
	concreteConns []net.Conn
}

func (s *tcpServer) start(ctx context.Context) error {
	var err error
	s.ln, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			conn, err := s.ln.Accept()
			if err != nil {
				log.Println("l.Accept err: ", err)
				return
			}
			s.concreteConns = append(s.concreteConns, conn)

			go func() {
				select {
				case <-ctx.Done():
					return
				default:
				}
				io.Copy(conn, conn)
			}()
		}
	}()
	return nil
}

func (s *tcpServer) stop() {
	s.cancel()
	s.closeConnections()
	s.ln.Close()
}

func (s *tcpServer) closeConnections() {
	for i := range s.concreteConns {
		s.concreteConns[i].Close()
	}
	s.concreteConns = s.concreteConns[:0]
}

func newUDPServer() *udpServer {
	return &udpServer{}
}

type udpServer struct {
	cancel context.CancelFunc
	conn   net.PacketConn
}

func (s *udpServer) start(ctx context.Context) error {
	var err error
	s.conn, err = net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, addr, err := s.conn.ReadFrom(buf)
			if err != nil {
				log.Println("l.ReadFrom err: ", err)
				return
			}

			s.conn.WriteTo(buf[:n], addr)
		}
	}()
	return nil
}

func (s *udpServer) stop() {
	s.cancel()
	s.conn.Close()
}

func TestUDPReadFromErr(t *testing.T) {
	testcases := []struct {
		name       string
		begin      func(*testing.T, func(net.Conn), *tls.Config) (net.Addr, context.CancelFunc)
		dialFunc   DialFunc
		wantErrMsg string
	}{
		{name: "UDP_Gonet", begin: beginUDPServer, dialFunc: NewDialFunc(), wantErrMsg: context.DeadlineExceeded.Error()},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			addr, cancel := tt.begin(t, echo, nil)
			t.Cleanup(cancel)

			pool := NewPool(tt.dialFunc)
			getOpts := func() (uint32, GetOptions) {
				id := getReqID()
				opts := NewGetOptions()
				opts.WithFrameParser(&simpleFrameParser{})
				opts.WithVID(id)
				return id, opts
			}
			// return error when decode fail.
			id, opts := getOpts()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			conn, err := pool.GetVirtualConn(ctx, addr.Network(), addr.String(), opts)
			require.Nil(t, err)

			err = conn.Write(encodeFrame(id, helloworld))
			require.Nil(t, err)
			_, err = conn.Read()
			require.Nil(t, err)
			conn.Close()
			cancel()
			time.Sleep(time.Second)
		})
	}
}
