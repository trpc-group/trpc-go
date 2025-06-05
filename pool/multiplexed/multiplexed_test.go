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

package multiplexed

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"

	"github.com/stretchr/testify/assert"
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
	updateMsg   func(msg codec.Msg) error
	IsStream    bool
	reader      io.Reader
	decodeError bool
	safe        bool
}

func (f *lengthDelimitedFramer) New(reader io.Reader) codec.Framer {
	return &lengthDelimitedFramer{
		updateMsg:   f.updateMsg,
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

type delimitedResponse struct {
	RequestID uint32
	body      []byte
}

type DelimitedRequest struct {
	RequestID uint32
	body      []byte
}

func (d *delimitedResponse) GetRequestID() uint32 {
	return d.RequestID
}

func (d *delimitedResponse) GetResponseBuf() []byte {
	return d.body
}

func (f *lengthDelimitedFramer) UpdateMsg(rsp interface{}, msg codec.Msg) error {
	if f.updateMsg == nil {
		return nil
	}
	return f.updateMsg(msg)
}

func (f *lengthDelimitedFramer) Decode() (codec.TransportResponseFrame, error) {
	head := make([]byte, 8)
	num, err := io.ReadFull(f.reader, head)
	if err != nil {
		return nil, err
	}

	if f.decodeError {
		return nil, errDecodeDelimited
	}

	if num != 8 {
		return nil, errors.New("invalid read full num")
	}

	n := binary.BigEndian.Uint32(head[:4])
	requestID := binary.BigEndian.Uint32(head[4:8])
	body := make([]byte, int(n))

	num, err = io.ReadFull(f.reader, body)
	if err != nil {
		return nil, err
	}

	if num != int(n) {
		return nil, errors.New("invalid read full body")
	}

	if f.IsStream {
		return &delimitedResponse{
			RequestID: requestID,
			body:      append(head, body...),
		}, nil
	}

	return &delimitedResponse{
		RequestID: requestID,
		body:      body,
	}, nil
}

func (f *lengthDelimitedFramer) Encode(req *DelimitedRequest) ([]byte, error) {
	l := len(req.body)
	buf := bytes.NewBuffer(make([]byte, 0, 8+l))
	if err := binary.Write(buf, binary.BigEndian, uint32(l)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, req.RequestID); err != nil {
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
		msg := codec.Message(context.Background())
		requestID := atomic.AddUint32(&s.requestID, 1)
		msg.WithRequestID(requestID)
		ld := &lengthDelimitedFramer{
			decodeError: true,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		m := New()
		opts := NewGetOptions()
		opts.WithMsg(msg)
		opts.WithFramerBuilder(ld)
		vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
		assert.Nil(s.T(), err)
		body := []byte("hello world")
		buf, err := ld.Encode(&DelimitedRequest{
			body:      body,
			RequestID: requestID,
		})
		require.Nil(s.T(), err)
		require.Nil(s.T(), vc.Write(buf))
		_, err = vc.Read()
		assert.Equal(s.T(), err, tt.wantErr)
		cancel()
	}
}

func (s *msuite) TestMultiplexedGetConcurrent() {
	count := 10
	ld := &lengthDelimitedFramer{}
	m := New()
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
				msg := codec.Message(context.Background())
				requestID := atomic.AddUint32(&s.requestID, 1)
				msg.WithRequestID(requestID)
				opts := NewGetOptions()
				opts.WithMsg(msg)
				opts.WithFramerBuilder(ld)
				vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
				assert.Nil(s.T(), err)
				body := []byte("hello world" + strconv.Itoa(i))
				buf, err := ld.Encode(&DelimitedRequest{
					body:      body,
					RequestID: requestID,
				})
				assert.Nil(s.T(), err)
				assert.Nil(s.T(), vc.Write(buf))
				rsp, err := vc.Read()
				assert.Nil(s.T(), err)
				assert.Equal(s.T(), rsp, body)
				cancel()
			}(i)
		}
		wg.Wait()
	}
}

func (s *msuite) TestMultiplexedGet() {
	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	m := New(WithConnectNumber(4), WithDropFull(true), WithQueueSize(50000))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: requestID,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)
}

func (s *msuite) TestMultiplexedGetWithSafeFramer() {
	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{safe: true}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := New(WithConnectNumber(4), WithDropFull(true), WithQueueSize(50000))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: requestID,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)
}

func (s *msuite) TestNoFramerBuilder() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	_, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Equal(s.T(), err, ErrFrameBuilderNil)
}

func (s *msuite) TestNoDecoder() {
	tests := []struct {
		network string
		address string
	}{
		{s.network, s.address},
		{s.udpNetwork, s.udpAddr},
	}

	for _, tt := range tests {
		msg := codec.Message(context.Background())
		msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m := New()
		opts := NewGetOptions()
		opts.WithMsg(msg)
		opts.WithFramerBuilder(&emptyFramerBuilder{})
		vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
		assert.Nil(s.T(), err)
		_, err = vc.Read()
		assert.Equal(s.T(), err, ErrDecoderNil)
	}
}

func (s *msuite) TestContextDeadline() {
	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	_, err = vc.Read()
	assert.Equal(s.T(), err, context.DeadlineExceeded)
	err = vc.Write([]byte("hello world"))
	assert.Equal(s.T(), err, context.DeadlineExceeded)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	vc, err = m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: requestID,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)
}

func (s *msuite) TestCloseConnection() {
	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	_, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	time.Sleep(500 * time.Millisecond)

	cs, ok := m.concreteConns[makeNodeKey(s.network, s.address)]
	assert.True(s.T(), ok)

	cs.conns[0].close(errors.New("fake error"), false)
	_, ok = m.concreteConns[makeNodeKey(s.network, s.address)]
	assert.False(s.T(), ok)
}

func (s *msuite) TestDuplicatedClose() {
	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: requestID,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)

	cs, ok := m.concreteConns[makeNodeKey(s.network, s.address)]
	assert.True(s.T(), ok)

	err1 := errors.New("error1")
	err2 := errors.New("error2")
	c := cs.conns[0]
	c.close(err1, false)
	c.close(err2, false)

	_, err = vc.Read()
	assert.Equal(s.T(), err, err1)
}

func (s *msuite) TestGetFail() {

	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	_, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	m.concreteConns[makeNodeKey(s.network, s.address)].expelled = true
	_, err = m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.NotNil(s.T(), err)
}

func (s *msuite) TestContextCancel() {
	msg := codec.Message(context.Background())
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	ld := &lengthDelimitedFramer{}

	// get with cancel.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	_, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.NotNil(s.T(), err)
}

// test when send fails.
func (s *msuite) TestSendFail() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithDropFull(true), WithQueueSize(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(&emptyFramerBuilder{})
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	err = vc.Write(body)
	assert.Nil(s.T(), err)
	err = vc.Write(body)
	assert.NotNil(s.T(), err)
}

func (s *msuite) TestWriteErrorCleanVirtualConnection() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithDropFull(true), WithQueueSize(0))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(&emptyFramerBuilder{})
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	err = vc.Write(body)
	assert.NotNil(s.T(), err)
	assert.Len(s.T(), vc.(*VirtualConnection).conn.virtualConns, 0)
}

func (s *msuite) TestReadErrorCleanVirtualConnection() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	m := New(WithDropFull(true), WithQueueSize(0))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(&lengthDelimitedFramer{})
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	time.Sleep(time.Millisecond * 100)
	_, err = vc.Read()
	assert.NotNil(s.T(), err)
	assert.Len(s.T(), vc.(*VirtualConnection).conn.virtualConns, 0)
}

func (s *msuite) TestUdpMultiplexedReadTimeout() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, "udp", s.udpAddr, opts)
	assert.Nil(s.T(), err)
	_, err = vc.Read()
	assert.Equal(s.T(), err, ctx.Err())
}

func (s *msuite) TestMultiplexedServerFail() {
	tests := []struct {
		network string
		address string
		exists  bool
	}{
		{s.network, "invalid address", false},
		{s.udpNetwork, "invalid address", false},
	}

	for _, tt := range tests {
		msg := codec.Message(context.Background())
		msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m := New(
			WithConnectNumber(1),
			// On windows, it will try to use up all the timeout to do the dialling.
			// So limit the dial timeout.
			WithDialTimeout(time.Millisecond),
		)
		opts := NewGetOptions()
		opts.WithMsg(msg)
		opts.WithFramerBuilder(&emptyFramerBuilder{})
		_, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
		s.T().Logf("m.GetVirtualConn err: %+v\n", err)
		// Because of possible out of order execution of goroutines,
		// the error may or may not be nil.
		if err != nil {
			// If it is non-nil, it must be an expelled error.
			require.True(s.T(), errors.Is(err, ErrConnectionsHaveBeenExpelled))
		}
		time.Sleep(10 * time.Millisecond)
		_, ok := m.concreteConns[makeNodeKey(tt.network, tt.address)]
		assert.Equal(s.T(), tt.exists, ok)
	}
}

func (s *msuite) TestMultiplexedConcurrentGetInvalidAddr() {
	const (
		network     = "tcp"
		invalidAddr = "invalid addr"
	)
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(&emptyFramerBuilder{})
	start := time.Now()
	for n := 16; ; n *= 2 {
		if time.Since(start) > time.Second*10 {
			require.FailNow(s.T(), "expected expelled error in 10s")
		}
		var eg errgroup.Group
		for i := 0; i < n; i++ {
			eg.Go(func() error {
				_, err := m.GetVirtualConn(ctx, network, invalidAddr, opts)
				return err
			})
		}
		if err := eg.Wait(); err != nil {
			s.T().Logf("ok, m.GetVirtualConn error: %+v\n", err)
			break
		}
	}
}

func (s *msuite) TestMultiplexedConcurrentGet() {
	const (
		network     = "tcp"
		invalidAddr = "invalid addr"
	)

	defer func() {
		if r := recover(); r != nil {
			require.FailNow(s.T(), "expected no panic")
		}
	}()

	m := New(WithConnectNumber(1))

	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(&emptyFramerBuilder{})
	start := time.Now()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		opts := NewGetOptions()
		opts.WithMsg(msg)
		opts.WithFramerBuilder(&emptyFramerBuilder{})
		if time.Since(start) > time.Second*5 {
			return
		}
		for i := 0; i < 10000; i++ {
			go m.GetVirtualConn(ctx, network, invalidAddr, opts)
		}
	}
}

func (s *msuite) TestWithLocalAddr() {
	tests := []struct {
		network string
		address string
	}{
		{s.network, s.address},
		{s.udpNetwork, s.udpAddr},
	}
	localAddr := "127.0.0.1"

	for _, tt := range tests {
		msg := codec.Message(context.Background())
		msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m := New()
		opts := NewGetOptions()
		opts.WithMsg(msg)
		opts.WithLocalAddr(localAddr + ":")
		ld := &lengthDelimitedFramer{}
		opts.WithFramerBuilder(ld)
		body := []byte("hello world")
		buf, err := ld.Encode(&DelimitedRequest{
			body:      body,
			RequestID: s.requestID,
		})
		assert.Nil(s.T(), err)
		vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
		assert.Nil(s.T(), err)
		assert.Nil(s.T(), vc.Write(buf))
		assert.Nil(s.T(), err)
		_, err = vc.Read()
		assert.Nil(s.T(), err)
		if tt.network == s.network {
			conn := vc.(*VirtualConnection).conn.getRawConn()
			realAddr := conn.LocalAddr().(*net.TCPAddr).IP.String()
			assert.Equal(s.T(), realAddr, localAddr)
		} else if tt.network == s.udpNetwork {
			realAddr := vc.(*VirtualConnection).conn.packetConn.LocalAddr().(*net.UDPAddr).IP.String()
			assert.Equal(s.T(), realAddr, localAddr)
		}
	}
}

func (s *msuite) TestShouldReconnect() {
	tests := []struct {
		name    string
		network string
		address string
		err     error
		want    bool
	}{
		{
			name:    "udp",
			network: s.udpNetwork,
			address: s.udpAddr,
			err:     nil,
			want:    false,
		},
		{
			name:    "tcpWithEOF",
			network: s.network,
			address: s.address,
			err:     io.EOF,
			want:    false,
		},
		{
			name:    "tcpWithOtherErr",
			network: s.network,
			address: s.address,
			err:     errors.New("other error"),
			want:    true,
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			msg := codec.Message(context.Background())
			msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

			m := New()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			opts := NewGetOptions()
			opts.WithMsg(msg)
			opts.WithFramerBuilder(&lengthDelimitedFramer{})
			_, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
			assert.Nil(s.T(), err)
			key := makeNodeKey(tt.network, tt.address)
			m.mu.Lock()
			val, ok := m.concreteConns[key]
			m.mu.Unlock()
			assert.True(t, ok)
			conn := val.conns[0]
			assert.Equal(t, tt.want, conn.shouldReconnect(tt.err))
		})
	}
}

func (s *msuite) TestTCPReconnect() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{}
	opts.WithFramerBuilder(ld)
	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: s.requestID,
	})
	assert.Nil(s.T(), err)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))
	_, err = vc.Read()
	assert.Nil(s.T(), err)

	// close conn

	cs, ok := m.concreteConns[makeNodeKey(s.network, s.address)]
	assert.True(s.T(), ok)
	c := cs.conns[0]
	conn := c.getRawConn()
	conn.Close()
	time.Sleep(100 * time.Millisecond)
	vc, err = m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))
	_, err = vc.Read()
	assert.Nil(s.T(), err)

	_, ok = m.concreteConns[makeNodeKey(s.network, s.address)]
	assert.True(s.T(), ok)

	// timeout after reconnected
	ctx, done := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer done()
	vc, err = m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	_, err = vc.Read()
	assert.ErrorIs(s.T(), err, context.DeadlineExceeded)
}

func (s *msuite) TestTCPReconnectMaxReconnectCount() {
	msg := codec.Message(context.Background())
	msg.WithRequestID(atomic.AddUint32(&s.requestID, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{}
	opts.WithFramerBuilder(ld)
	_, err := m.GetVirtualConn(ctx, s.network, "invalid address", opts)
	assert.Nil(s.T(), err)
	time.Sleep(time.Second)
	_, ok := m.concreteConns[makeNodeKey(s.network, "invalid address")]
	assert.False(s.T(), ok)
}

func (s *msuite) TestStreamMultiplexed() {
	msg := codec.Message(context.Background())
	streamID := 101
	msg.WithStreamID(uint32(streamID))
	msg.WithRequestID(uint32(streamID))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), vc)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: uint32(streamID),
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), buf, rsp)
}

func (s *msuite) TestStreamMultiplexed_Addr() {
	msg := codec.Message(context.Background())
	streamID := 101
	msg.WithStreamID(uint32(streamID))
	msg.WithRequestID(uint32(streamID))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), vc)
	assert.Equal(s.T(), s.address, vc.RemoteAddr().String())
	time.Sleep(50 * time.Millisecond)

	la := vc.LocalAddr()
	assert.NotNil(s.T(), la)

	ra := vc.RemoteAddr()
	assert.Equal(s.T(), s.address, ra.String())
}

func (s *msuite) TestStreamMultiplexed_MaxVirConnPerConn() {
	msg := codec.Message(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New(WithMaxVirConnsPerConn(4))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFramerBuilder(ld)
	var cs *Connections
	for i := 0; i < 10; i++ {
		streamID := 99 + i
		msg.WithRequestID(uint32(streamID))
		vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), vc)

		var ok bool
		cs, ok = m.concreteConns[makeNodeKey(s.network, s.address)]
		require.True(s.T(), ok)

		body := []byte("hello world")
		buf, err := ld.Encode(&DelimitedRequest{
			body:      body,
			RequestID: uint32(streamID),
		})
		assert.Nil(s.T(), err)
		assert.Nil(s.T(), vc.Write(buf))

		rsp, err := vc.Read()
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), buf, rsp)
	}
	assert.Equal(s.T(), 3, len(cs.conns))
}

func (s *msuite) TestStreamMultiplexed_MaxIdleConnPerHost() {
	msg := codec.Message(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New(WithMaxVirConnsPerConn(2), WithMaxIdleConnsPerHost(3))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFramerBuilder(ld)

	vcs := make([]*VirtualConnection, 0)
	for i := 0; i < 10; i++ {
		streamID := 99 + i
		msg.WithRequestID(uint32(streamID))
		vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
		assert.Nil(s.T(), err)
		vcs = append(vcs, vc.(*VirtualConnection))
	}
	cs, ok := m.concreteConns[makeNodeKey(s.network, s.address)]
	require.True(s.T(), ok)
	assert.Equal(s.T(), 5, len(cs.conns))
	for i := 0; i < 10; i++ {
		vcs[i].Close()
	}
	assert.Equal(s.T(), 3, len(cs.conns))
}

func (s *msuite) TestMultiplexedGetConcurrent_MaxIdleConnPerHost() {
	count := 100
	ld := &lengthDelimitedFramer{}
	m := New(WithMaxVirConnsPerConn(20), WithMaxIdleConnsPerHost(2))
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
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				msg := codec.Message(context.Background())
				requestID := atomic.AddUint32(&s.requestID, 1)
				msg.WithRequestID(requestID)
				opts := NewGetOptions()
				opts.WithMsg(msg)
				opts.WithFramerBuilder(ld)
				vc, err := m.GetVirtualConn(ctx, tt.network, tt.address, opts)
				assert.Nil(s.T(), err)
				body := []byte("hello world" + strconv.Itoa(i))
				buf, err := ld.Encode(&DelimitedRequest{
					body:      body,
					RequestID: requestID,
				})
				assert.Nil(s.T(), err)
				assert.Nil(s.T(), vc.Write(buf))
				rsp, err := vc.Read()
				assert.Nil(s.T(), err)
				assert.Equal(s.T(), rsp, body)
				vc.Close()
				cancel()
			}(i)
			if i%50 == 0 {
				time.Sleep(50 * time.Millisecond)
			}
		}
		wg.Wait()
	}
}

func (s *msuite) TestMultiplexedReconnectOnConnectError() {
	ctx := context.Background()
	ts := newTCPServer()
	ts.start(ctx)
	defer ts.stop()
	m := New(
		WithConnectNumber(1),
		// On windows, it will try to use up all the timeout to do the dialling.
		// So limit the dial timeout.
		WithDialTimeout(time.Millisecond),
	)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	opts := NewGetOptions()
	msg := codec.Message(ctx)
	requestID := atomic.AddUint32(&s.requestID, 1)
	msg.WithRequestID(requestID)
	opts.WithMsg(msg)
	readTrigger := make(chan struct{})
	readErr := make(chan error)
	opts.WithFramerBuilder(&triggeredReadFramerBuilder{readTrigger: readTrigger, readErr: readErr})
	vc, err := m.GetVirtualConn(ctx, s.network, ts.ln.Addr().String(), opts)
	require.Nil(s.T(), err)
	<-readTrigger                     // Wait for the first read.
	require.Nil(s.T(), ts.ln.Close()) // Then close the server.
	readErr <- errAlwaysFail          // Fail the first read to trigger reconnection.
	log.Printf("%+v", vc.(*VirtualConnection).conn)
	log.Println(vc.(*VirtualConnection).conn.reconnectCount, vc.(*VirtualConnection).conn.maxReconnectCount)
	time.Sleep(10 * time.Millisecond)
	log.Println(vc.(*VirtualConnection).conn.reconnectCount, vc.(*VirtualConnection).conn.maxReconnectCount)
	require.Eventually(s.T(),
		func() bool {
			return vc.(*VirtualConnection).conn.maxReconnectCount+1 == vc.(*VirtualConnection).conn.reconnectCount
		},
		time.Second, defaultReconnectCountResetInterval/2)
}

func TestMultiplexedReconnectOnReadError(t *testing.T) {
	ts := newTCPServer()
	require.Nil(t, ts.start(context.Background()))
	defer ts.stop()

	m := New(
		WithConnectNumber(1),
		// On windows, it will try to use up all the timeout to do the dialling.
		// So limit the dial timeout.
		WithDialTimeout(time.Millisecond),
		WithMaxReconnectCount(5),
		WithInitialBackoff(time.Microsecond),
	)

	// Just for test, maxBackoff and reconnectCountResetInterval should be calculated by reconnect strategy.
	m.opts.maxBackoff = 50 * time.Microsecond
	m.opts.reconnectCountResetInterval = time.Hour

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts := NewGetOptions()
	calledAt := make([]time.Time, 0, m.opts.maxReconnectCount)
	msg := codec.Message(ctx)
	msg.WithRequestID(1)
	opts.WithMsg(msg)
	opts.WithFramerBuilder(&errFramerBuilder{readFrameCalledAt: &calledAt})
	vc, err := m.GetVirtualConn(ctx, ts.ln.Addr().Network(), ts.ln.Addr().String(), opts)
	require.Nil(t, err)
	require.Eventually(t,
		func() bool {
			return vc.(*VirtualConnection).conn.maxReconnectCount+1 == vc.(*VirtualConnection).conn.reconnectCount
		},
		3*time.Second, time.Second,
		fmt.Sprintf("final status: maxReconnectCount+1=%d, vc.conn.reconnectCount=%d",
			vc.(*VirtualConnection).conn.maxReconnectCount+1, vc.(*VirtualConnection).conn.reconnectCount))
	require.Eventually(t,
		func() bool { return vc.(*VirtualConnection).conn.maxReconnectCount+1 == len(calledAt) },
		3*time.Second, 50*time.Millisecond,
		fmt.Sprintf("final status: maxReconnectCount+1=%d, len(calledAt)=%d",
			vc.(*VirtualConnection).conn.maxReconnectCount+1, len(calledAt)))
	var differences []float64
	for i := 1; i < len(calledAt); i++ {
		delay := calledAt[i].Sub(calledAt[i-1])
		expectedBackoff := vc.(*VirtualConnection).conn.initialBackoff * time.Duration(i)
		t.Logf("calledAt delay: %2dms, expect: %2dms (between %d and %d)\n",
			delay.Milliseconds(), expectedBackoff.Milliseconds(), i-1, i)
		differences = append(differences, float64(delay-expectedBackoff))
	}
	require.Equal(t, vc.(*VirtualConnection).conn.maxReconnectCount+1, len(calledAt),
		"the actual times called is %d, expect %d", len(calledAt), vc.(*VirtualConnection).conn.maxReconnectCount+1)
	t.Logf("differences: %+v", differences)
	t.Logf("mean of differences between real retry delay and the calculated backoff: %vns", mean(differences))
	ss := std(differences)
	t.Logf("std of differences between real retry delay and the calculated backoff: %vns", ss)
	const expectedStdLimit = time.Second
	require.Less(t, ss, float64(expectedStdLimit),
		"standard deviation of differences between real retry delay and calculated backoff is expected to be within %s",
		expectedStdLimit)
}

func TestMultiplexedReconnectOnWriteError(t *testing.T) {
	ctx := context.Background()
	ts := newTCPServer()
	require.Nil(t, ts.start(ctx))
	defer ts.stop()
	m := New(
		WithConnectNumber(1),
		// On windows, it will try to use up all the timeout to do the dialling.
		// So limit the dial timeout.
		WithDialTimeout(time.Millisecond),
		WithMaxReconnectCount(5),
		WithInitialBackoff(time.Microsecond),
	)

	// Just for test, maxBackoff and reconnectCountResetInterval should be calculated by reconnect strategy.
	m.opts.maxBackoff = 50 * time.Microsecond
	m.opts.reconnectCountResetInterval = time.Hour

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	opts := NewGetOptions()
	msg := codec.Message(ctx)
	msg.WithRequestID(1)
	opts.WithMsg(msg)
	const readTriggerChanSize = 10
	readTrigger := make(chan struct{}, readTriggerChanSize)
	readErr := make(chan error)
	opts.WithFramerBuilder(&triggeredReadFramerBuilder{readTrigger: readTrigger, readErr: readErr})
	var (
		vc  VirtualConn
		err error
	)
	require.Eventually(t, func() bool {
		vc, err = m.GetVirtualConn(ctx, ts.ln.Addr().Network(), ts.ln.Addr().String(), opts)
		require.Nil(t, err)
		bs := []byte("hello")
		err = vc.Write(bs)
		return err == nil
	}, time.Second, 300*time.Millisecond,
		fmt.Sprintf("multiplex get connection failed: %+v", err))

	timeout := 5 * time.Second
	ctx1, cancel1 := context.WithTimeout(context.Background(), timeout)
	defer cancel1()

	select {
	case <-readTrigger:
	case <-ctx1.Done():
		t.Fatalf("Timed out waiting for readTrigger after %v", timeout)
	}
	require.Nil(t, vc.(*VirtualConnection).conn.getRawConn().Close()) // Now close the underlying connection.

	require.Nil(t, vc.Write([]byte("hello"))) // Then this write will trigger a reconnection on write error.
	// Now we are cool to check that a reconnection is triggered.
	require.Eventually(t,
		func() bool { return vc.(*VirtualConnection).conn.reconnectCount == 1 },
		3*time.Second, 20*time.Millisecond,
		fmt.Sprintf("final status: vc.conn.reconnectCount=%d, want 1",
			vc.(*VirtualConnection).conn.reconnectCount))
}

func TestMultiplexedSetReconnectParamsPanic(t *testing.T) {
	tests := []struct {
		name                        string
		maxReconnectCount           int
		initialBackoff              time.Duration
		reconnectCountResetInterval time.Duration
	}{
		{
			name:              "maxReconnectCount is less than 0",
			maxReconnectCount: -100,
			initialBackoff:    100,
		},
		{
			name:              "initialBackoff is less than or equal to 0",
			maxReconnectCount: defaultMaxReconnectCount,
			initialBackoff:    -100,
		},
		{
			name:              "maxReconnectCount and initialBackoff are both invalid",
			maxReconnectCount: -100,
			initialBackoff:    -100,
		},
		{
			name:                        "reconnectionResetInterval is too small",
			maxReconnectCount:           100,
			initialBackoff:              100,
			reconnectCountResetInterval: time.Nanosecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("FunctionThatPanics did not panic")
				}
			}()
			New(
				WithMaxReconnectCount(tt.maxReconnectCount),
				WithInitialBackoff(tt.initialBackoff),
				WithReconnectCountResetInterval(tt.reconnectCountResetInterval),
			)
		})
	}
}

func TestMultiplexedSetReconnectParamsSuccess(t *testing.T) {
	tests := []struct {
		name              string
		dialTimeout       time.Duration
		maxReconnectCount int
		initialBackoff    time.Duration

		expectedErr                         bool
		expectedMaxReconnectCount           int
		expectedInitialBackoff              time.Duration
		expectedMaxBackoff                  time.Duration
		expectedReconnectCountResetInterval time.Duration
	}{
		{
			name:                                "valid1",
			dialTimeout:                         defaultDialTimeout,
			maxReconnectCount:                   20,
			initialBackoff:                      10,
			expectedErr:                         false,
			expectedMaxReconnectCount:           20,
			expectedInitialBackoff:              10,
			expectedMaxBackoff:                  20 * time.Duration(10),
			expectedReconnectCountResetInterval: defaultDialTimeout*40 + 10*time.Duration((1+20)*20),
		},
		{
			name:                                "valid2",
			dialTimeout:                         defaultDialTimeout,
			maxReconnectCount:                   10,
			initialBackoff:                      5,
			expectedErr:                         false,
			expectedMaxReconnectCount:           10,
			expectedInitialBackoff:              5,
			expectedMaxBackoff:                  10 * time.Duration(5),
			expectedReconnectCountResetInterval: defaultDialTimeout*20 + 5*time.Duration((1+10)*10),
		},
		{
			name:                                "valid3",
			dialTimeout:                         defaultDialTimeout * 2,
			maxReconnectCount:                   10,
			initialBackoff:                      5,
			expectedErr:                         false,
			expectedMaxReconnectCount:           10,
			expectedInitialBackoff:              5,
			expectedMaxBackoff:                  10 * time.Duration(5),
			expectedReconnectCountResetInterval: defaultDialTimeout*40 + 5*time.Duration((1+10)*10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(
				WithDialTimeout(tt.dialTimeout),
				WithMaxReconnectCount(tt.maxReconnectCount),
				WithInitialBackoff(tt.initialBackoff),
			)

			require.Equal(t, tt.expectedMaxReconnectCount, m.opts.maxReconnectCount,
				"expected maxReconnectCount to be %v, got %v", tt.expectedMaxReconnectCount, m.opts.maxReconnectCount)

			require.Equal(t, tt.expectedInitialBackoff, m.opts.initialBackoff,
				"expected initialBackoff to be %v, got %v", tt.expectedInitialBackoff, m.opts.initialBackoff)

			require.Equal(t, tt.expectedMaxBackoff, m.opts.maxBackoff,
				"expected maxBackoff to be %v, got %v", tt.expectedMaxBackoff, m.opts.maxBackoff)

			require.Equal(t, tt.expectedReconnectCountResetInterval, m.opts.reconnectCountResetInterval,
				"expected reconnectCountResetInterval to be %v, got %v",
				tt.expectedReconnectCountResetInterval, m.opts.reconnectCountResetInterval)
		})
	}
}

func (s *msuite) TestNoConcurrentModifyMessage() {
	requestID := atomic.AddUint32(&s.requestID, 1)
	meta := make(codec.MetaData)
	msg := codec.Message(context.Background())
	msg.WithRequestID(requestID)
	msg.WithClientMetaData(meta)
	ld := &lengthDelimitedFramer{
		// multiplexed update message
		updateMsg: func(msg codec.Msg) error {
			meta := msg.ClientMetaData()
			meta["key"] = []byte("value")
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: requestID,
	})
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), vc.Write(buf))
	time.Sleep(200 * time.Millisecond)

	// multiplexed reader routine shouldn't modify message, otherwise cocurrenty
	// read/write will happen.
	_, ok := meta["key"]
	assert.False(s.T(), ok)
}

func (s *msuite) TestUpdateMessageFail() {
	requestID := atomic.AddUint32(&s.requestID, 1)
	meta := make(codec.MetaData)
	msg := codec.Message(context.Background())
	msg.WithRequestID(requestID)
	msg.WithClientMetaData(meta)
	ld := &lengthDelimitedFramer{
		// multiplexed update message
		updateMsg: func(msg codec.Msg) error {
			return errs.New(599, "update message failed")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithMsg(msg)
	opts.WithFramerBuilder(ld)
	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&DelimitedRequest{
		body:      body,
		RequestID: requestID,
	})
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), vc.Write(buf))
	_, err = vc.Read()
	assert.Equal(s.T(), 599, errs.Code(err))
}

func (s *msuite) TestTriggerReadOnConnectionClose() {
	msg := codec.Message(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithMsg(msg)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFramerBuilder(ld)

	vc, err := m.GetVirtualConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	vc.Close()

	finished := make(chan struct{}, 1)
	go func() {
		_, err := vc.Read()
		assert.NotNil(s.T(), err)
		finished <- struct{}{}
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		assert.FailNow(s.T(),
			"When the connection is closed, the read operation is not triggered to return an error.")
	}
}

func TestMultiplexedDestroyMayCauseGoroutineLeak(t *testing.T) {
	l, err := net.Listen("tcp", ":")
	require.Nil(t, err)
	const connNum = 2
	acceptedConns, acceptErrs := make(chan net.Conn, connNum*2), make(chan error)
	var closedConns uint32
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				acceptErrs <- err
				return
			}
			acceptedConns <- c
			go func() {
				_, _ = io.Copy(c, c)
				atomic.AddUint32(&closedConns, 1)
			}()
		}
	}()

	fb := fixedLenFrameBuilder{packetLen: 2}
	dialTimeout := time.Millisecond * 50
	m := New(
		WithConnectNumber(connNum),
		// replace the too long default 1s dial timeout.
		WithDialTimeout(dialTimeout))
	getVirtualConn := func(requestID uint32) (VirtualConn, error) {
		getOptions := NewGetOptions()
		msg := codec.Message(context.Background())
		msg.WithRequestID(requestID)
		getOptions.WithMsg(msg)
		getOptions.WithFramerBuilder(&fb)
		return m.GetVirtualConn(context.Background(), l.Addr().Network(), l.Addr().String(), getOptions)
	}

	vc, err := getVirtualConn(1)
	require.Nil(t, err)
	require.Nil(t, vc.Write(fb.EncodeWithRequestID(1, []byte("1a"))))
	read, err := vc.Read()
	require.Nil(t, err)
	require.Equal(t, []byte("1a"), read)
	vc.Close()

	var (
		c1 net.Conn
		c2 net.Conn
	)
	select {
	case c1 = <-acceptedConns:
	case <-time.After(time.Second):
		require.FailNow(t, "should accept a connection")
	}
	select {
	case c2 = <-acceptedConns:
	case <-time.After(time.Second):
		require.FailNow(t, "multiplexed should establish two concreteConns")
	}

	require.Nil(t, l.Close())
	<-acceptErrs
	require.Nil(t, c1.Close())
	// on windows, connecting to closed listener returns an error until dial timeout, not immediately.
	// we should sleep additional dialTimeout * maxReconnectCount to wait all retry finished.
	time.Sleep((defaultMaxBackoff + dialTimeout) * time.Duration(defaultMaxReconnectCount))
	require.Equal(t, uint32(1), atomic.LoadUint32(&closedConns))

	vc, err = getVirtualConn(2)
	require.Nil(t, err)
	require.Nil(t, vc.Write(fb.EncodeWithRequestID(2, []byte("2a"))))
	require.EqualValues(t, 1, atomic.LoadUint32(&closedConns))
	read, err = vc.Read()
	require.Nil(t, err)
	require.Equal(t, []byte("2a"), read)
	require.Nil(t, err)
	require.Nil(t, c2.Close())
}

func mean(v []float64) float64 {
	n := len(v)
	if n == 0 {
		return 0
	}
	var res float64
	for i := 0; i < n; i++ {
		res += v[i]
	}
	return res / float64(n)
}

func variance(v []float64) float64 {
	n := len(v)
	if n <= 1 {
		return 0
	}
	var res float64
	m := mean(v)
	for i := 0; i < n; i++ {
		res += (v[i] - m) * (v[i] - m)
	}
	return res / float64(n-1)
}

func std(v []float64) float64 {
	return math.Sqrt(variance(v))
}

type errFramerBuilder struct {
	readFrameCalledAt *[]time.Time
}

func (fb *errFramerBuilder) New(io.Reader) codec.Framer {
	return &errFramer{
		calledAt: fb.readFrameCalledAt,
	}
}

var errAlwaysFail = errors.New("always fail")

type errFramer struct {
	calledAt *[]time.Time
}

// ReadFrame implements codec.Framer.
func (f *errFramer) ReadFrame() ([]byte, error) {
	return nil, errAlwaysFail
}

// Decode parse frame head, package head and package body from response.
func (f *errFramer) Decode() (codec.TransportResponseFrame, error) {
	*(f.calledAt) = append(*(f.calledAt), time.Now())
	return nil, errAlwaysFail
}

// UpdateMsg update Msg content, the first input param is parsed response data.
func (f *errFramer) UpdateMsg(interface{}, codec.Msg) error {
	return nil
}

type triggeredReadFramerBuilder struct {
	readTrigger chan struct{}
	readErr     chan error
}

func (fb *triggeredReadFramerBuilder) New(io.Reader) codec.Framer {
	return &triggeredReadFramer{
		readTrigger: fb.readTrigger,
		readErr:     fb.readErr,
	}
}

type triggeredReadFramer struct {
	readTrigger chan struct{}
	readErr     chan error
}

// ReadFrame implements codec.Framer.
func (f *triggeredReadFramer) ReadFrame() ([]byte, error) {
	f.readTrigger <- struct{}{}
	err := <-f.readErr
	return nil, err
}

// Decode parse frame head, package head and package body from response.
func (f *triggeredReadFramer) Decode() (codec.TransportResponseFrame, error) {
	f.readTrigger <- struct{}{}
	err := <-f.readErr
	return nil, err
}

// UpdateMsg update Msg content, the first input param is parsed response data.
func (f *triggeredReadFramer) UpdateMsg(interface{}, codec.Msg) error {
	return nil
}

type fixedLenFrameBuilder struct {
	packetLen int
}

func (fb *fixedLenFrameBuilder) New(r io.Reader) codec.Framer {
	return &fixedLenFramer{
		decode: fb.Decode,
		buf:    make([]byte, 4+fb.packetLen), // uint64 request id + packet len
		r:      r,
	}
}

func (*fixedLenFrameBuilder) EncodeWithRequestID(id uint32, buf []byte) []byte {
	bts := make([]byte, 4+len(buf))
	binary.BigEndian.PutUint32(bts[:4], id)
	copy(bts[4:], buf)
	return bts
}

func (*fixedLenFrameBuilder) Decode(bts []byte) (uint32, []byte, error) {
	if l := len(bts); l < 4 {
		return 0, nil, fmt.Errorf("bts len %d must not be lesser than 8, content: %q", l, bts)
	}
	return binary.BigEndian.Uint32(bts), bts[4:], nil
}

type fixedLenFramer struct {
	decode func([]byte) (uint32, []byte, error)
	buf    []byte
	r      io.Reader
}

func (f *fixedLenFramer) ReadFrame() ([]byte, error) {
	return nil, errors.New("should not be used by multiplexed")
}

func (f *fixedLenFramer) Decode() (codec.TransportResponseFrame, error) {
	n, err := f.r.Read(f.buf)
	if err != nil {
		return nil, err
	}
	id, bts, err := f.decode(f.buf[:n])
	if err != nil {
		return nil, err
	}
	return &delimitedResponse{
		RequestID: id,
		body:      bts,
	}, nil
}

func (f *fixedLenFramer) UpdateMsg(interface{}, codec.Msg) error {
	return nil
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
