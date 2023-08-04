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

func (f *lengthDelimitedFramer) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	head := make([]byte, 8)
	num, err := io.ReadFull(rc, head)
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

	num, err = io.ReadFull(rc, body)
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

func (f *lengthDelimitedFramer) Encode(req *delimitedRequest) ([]byte, error) {
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
		m := New()
		opts := NewGetOptions()
		opts.WithVID(id)
		opts.WithFrameParser(ld)
		vc, err := m.GetMuxConn(ctx, tt.network, tt.address, opts)
		assert.Nil(s.T(), err)
		body := []byte("hello world")
		buf, err := ld.Encode(&delimitedRequest{
			body:      body,
			requestID: id,
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
				id := atomic.AddUint32(&s.requestID, 1)
				opts := NewGetOptions()
				opts.WithVID(id)
				opts.WithFrameParser(ld)
				vc, err := m.GetMuxConn(ctx, tt.network, tt.address, opts)
				assert.Nil(s.T(), err)
				body := []byte("hello world" + strconv.Itoa(i))
				buf, err := ld.Encode(&delimitedRequest{
					body:      body,
					requestID: id,
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
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	m := New(WithConnectNumber(4), WithDropFull(true), WithQueueSize(50000))
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)
}

func (s *msuite) TestMultiplexedGetWithSafeFramer() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{safe: true}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := New(WithConnectNumber(4), WithDropFull(true), WithQueueSize(50000))
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)
}

func (s *msuite) TestNoFramerParser() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New()
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	_, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Equal(s.T(), err, ErrFrameParserNil)
}

func (s *msuite) TestContextDeadline() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	_, err = vc.Read()
	assert.Equal(s.T(), err, context.DeadlineExceeded)
	err = vc.Write([]byte("hello world"))
	assert.Equal(s.T(), err, context.DeadlineExceeded)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	vc, err = m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)
}

func (s *msuite) TestCloseConnection() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	_, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	time.Sleep(500 * time.Millisecond)
	v, ok := m.concreteConns.Load(makeNodeKey(s.network, s.address))
	assert.True(s.T(), ok)
	cs := v.(*Connections)
	cs.conns[0].close(errors.New("fake error"), false)
	_, ok = m.concreteConns.Load(makeNodeKey(s.network, s.address))
	assert.False(s.T(), ok)
}

func (s *msuite) TestDuplicatedClose() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	buf, err := ld.Encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rsp, body)

	v, ok := m.concreteConns.Load(makeNodeKey(s.network, s.address))
	assert.True(s.T(), ok)
	cs := v.(*Connections)
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	c := cs.conns[0]
	c.close(err1, false)
	c.close(err2, false)

	_, err = vc.Read()
	assert.Equal(s.T(), err, err1)
}

func (s *msuite) TestGetFail() {
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(ld)
	_, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	m.concreteConns.Store(makeNodeKey(s.network, s.address), &Connection{})
	_, err = m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.NotNil(s.T(), err)
}

func (s *msuite) TestContextCancel() {
	id := atomic.AddUint32(&s.requestID, 1)
	ld := &lengthDelimitedFramer{}

	// get with cancel.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := New()
	opts := NewGetOptions()
	opts.WithVID(id)
	opts.WithFrameParser(ld)
	_, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.NotNil(s.T(), err)
}

// test when send fails.
func (s *msuite) TestSendFail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithDropFull(true), WithQueueSize(1))
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(&emptyFrameParser{})
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)

	body := []byte("hello world")
	err = vc.Write(body)
	assert.Nil(s.T(), err)
	err = vc.Write(body)
	assert.NotNil(s.T(), err)
}

func (s *msuite) TestWriteErrorCleanVirtualConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithDropFull(true), WithQueueSize(0))
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(&emptyFrameParser{})
	mc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	vc, ok := mc.(*VirtualConnection)
	assert.True(s.T(), ok)

	body := []byte("hello world")
	err = vc.Write(body)
	assert.NotNil(s.T(), err)
	assert.Len(s.T(), vc.conn.virConns, 0)
}

func (s *msuite) TestReadErrorCleanVirtualConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	m := New(WithDropFull(true), WithQueueSize(0))
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(&lengthDelimitedFramer{})
	mc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	vc, ok := mc.(*VirtualConnection)
	assert.True(s.T(), ok)

	time.Sleep(time.Millisecond * 100)
	_, err = vc.Read()
	assert.NotNil(s.T(), err)
	assert.Len(s.T(), vc.conn.virConns, 0)
}

func (s *msuite) TestUdpMultiplexedReadTimeout() {
	ld := &lengthDelimitedFramer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New()
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, "udp", s.udpAddr, opts)
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m := New(
			WithConnectNumber(1),
			// On windows, it will try to use up all the timeout to do the dialling.
			// So limit the dial timeout.
			WithDialTimeout(time.Millisecond),
		)
		opts := NewGetOptions()
		opts.WithVID(atomic.AddUint32(&s.requestID, 1))
		opts.WithFrameParser(&emptyFrameParser{})
		_, err := m.GetMuxConn(ctx, tt.network, tt.address, opts)
		s.T().Logf("m.GetMuxConn err: %+v\n", err)
		// Because of possible out of order execution of goroutines,
		// the error may or may not be nil.
		if err != nil {
			// If it is non-nil, it must be an expelled error.
			require.True(s.T(), errors.Is(err, ErrConnectionsHaveBeenExpelled))
		}
		time.Sleep(10 * time.Millisecond)
		_, ok := m.concreteConns.Load(makeNodeKey(tt.network, tt.address))
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
	opts.WithFrameParser(&emptyFrameParser{})
	start := time.Now()
	for n := 1; ; n++ {
		if time.Since(start) > time.Second*10 {
			require.FailNow(s.T(), "expected expelled error in 10s")
		}
		var eg errgroup.Group
		for i := 0; i < n; i++ {
			eg.Go(func() error {
				_, err := m.GetMuxConn(ctx, network, invalidAddr, opts)
				return err
			})
		}
		if err := eg.Wait(); err != nil {
			s.T().Logf("ok, m.GetMuxConn error: %+v\n", err)
			break
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		m := New()
		opts := NewGetOptions()
		opts.WithVID(atomic.AddUint32(&s.requestID, 1))
		opts.WithLocalAddr(localAddr + ":")
		ld := &lengthDelimitedFramer{}
		opts.WithFrameParser(ld)
		body := []byte("hello world")
		buf, err := ld.Encode(&delimitedRequest{
			body:      body,
			requestID: s.requestID,
		})
		assert.Nil(s.T(), err)
		mc, err := m.GetMuxConn(ctx, tt.network, tt.address, opts)
		assert.Nil(s.T(), err)
		vc, ok := mc.(*VirtualConnection)
		assert.True(s.T(), ok)
		assert.Nil(s.T(), vc.Write(buf))
		assert.Nil(s.T(), err)
		_, err = vc.Read()
		assert.Nil(s.T(), err)
		if tt.network == s.network {
			conn := vc.conn.getRawConn()
			realAddr := conn.LocalAddr().(*net.TCPAddr).IP.String()
			assert.Equal(s.T(), realAddr, localAddr)
		} else if tt.network == s.udpNetwork {
			realAddr := vc.conn.packetConn.LocalAddr().(*net.UDPAddr).IP.String()
			assert.Equal(s.T(), realAddr, localAddr)
		}
	}
}

func (s *msuite) TestTCPReconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	ld := &lengthDelimitedFramer{}
	opts.WithFrameParser(ld)
	body := []byte("hello world")
	buf, err := ld.Encode(&delimitedRequest{
		body:      body,
		requestID: s.requestID,
	})
	assert.Nil(s.T(), err)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))
	_, err = vc.Read()
	assert.Nil(s.T(), err)

	// close conn
	val, ok := m.concreteConns.Load(makeNodeKey(s.network, s.address))
	assert.True(s.T(), ok)
	c := val.(*Connections).conns[0]
	conn := c.getRawConn()
	conn.Close()
	time.Sleep(100 * time.Millisecond)
	vc, err = m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))
	_, err = vc.Read()
	assert.Nil(s.T(), err)
	_, ok = m.concreteConns.Load(makeNodeKey(s.network, s.address))
	assert.True(s.T(), ok)

	// timeout after reconnected
	ctx, done := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer done()
	vc, err = m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	_, err = vc.Read()
	assert.ErrorIs(s.T(), err, context.DeadlineExceeded)
}

func (s *msuite) TestTCPReconnectMaxReconnectCount() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := New(WithConnectNumber(1))
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	ld := &lengthDelimitedFramer{}
	opts.WithFrameParser(ld)
	_, err := m.GetMuxConn(ctx, s.network, "invalid address", opts)
	assert.Nil(s.T(), err)
	time.Sleep(time.Second)
	_, ok := m.concreteConns.Load(makeNodeKey(s.network, "invalid address"))
	assert.False(s.T(), ok)
}

func (s *msuite) TestStreamMultiplexd() {
	id := atomic.AddUint32(&s.requestID, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithVID(id)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), vc)

	body := []byte("hello world")
	buf, err := ld.Encode(&delimitedRequest{
		body:      body,
		requestID: id,
	})
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), vc.Write(buf))

	rsp, err := vc.Read()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), buf, rsp)
}

func (s *msuite) TestStreamMultiplexd_Addr() {
	streamID := atomic.AddUint32(&s.requestID, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New()
	opts := NewGetOptions()
	opts.WithVID(streamID)
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)
	vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), vc)
	time.Sleep(50 * time.Millisecond)

	la := vc.LocalAddr()
	assert.NotNil(s.T(), la)

	ra := vc.RemoteAddr()
	assert.Equal(s.T(), s.address, ra.String())
}

func (s *msuite) TestStreamMultiplexd_MaxVirConnPerConn() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New(WithMaxVirConnsPerConn(4))
	opts := NewGetOptions()
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)
	var cs *Connections
	for i := 0; i < 10; i++ {
		id := atomic.AddUint32(&s.requestID, 1)
		opts.WithVID(id)
		vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), vc)
		conns, ok := m.concreteConns.Load(makeNodeKey(s.network, s.address))
		require.True(s.T(), ok)
		cs, ok = conns.(*Connections)
		require.True(s.T(), ok)

		body := []byte("hello world")
		buf, err := ld.Encode(&delimitedRequest{
			body:      body,
			requestID: uint32(id),
		})
		assert.Nil(s.T(), err)
		assert.Nil(s.T(), vc.Write(buf))

		rsp, err := vc.Read()
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), buf, rsp)
	}
	assert.Equal(s.T(), 3, len(cs.conns))
}

func (s *msuite) TestStreamMultiplexd_MaxIdleConnPerHost() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	m := New(WithMaxVirConnsPerConn(2), WithMaxIdleConnsPerHost(3))
	opts := NewGetOptions()
	ld := &lengthDelimitedFramer{IsStream: true}
	opts.WithFrameParser(ld)

	vcs := make([]MuxConn, 0)
	for i := 0; i < 10; i++ {
		id := atomic.AddUint32(&s.requestID, 1)
		opts.WithVID(id)
		vc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
		assert.Nil(s.T(), err)
		vcs = append(vcs, vc)
	}
	conns, ok := m.concreteConns.Load(makeNodeKey(s.network, s.address))
	require.True(s.T(), ok)
	cs, ok := conns.(*Connections)
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
				id := atomic.AddUint32(&s.requestID, 1)
				opts := NewGetOptions()
				opts.WithVID(id)
				opts.WithFrameParser(ld)
				vc, err := m.GetMuxConn(ctx, tt.network, tt.address, opts)
				assert.Nil(s.T(), err)
				body := []byte("hello world" + strconv.Itoa(i))
				buf, err := ld.Encode(&delimitedRequest{
					body:      body,
					requestID: id,
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
		WithDialTimeout(time.Millisecond*10),
	)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	readTrigger := make(chan struct{})
	readErr := make(chan error)
	opts.WithFrameParser(&triggeredReadFramerBuilder{readTrigger: readTrigger, readErr: readErr})
	mc, err := m.GetMuxConn(ctx, s.network, ts.ln.Addr().String(), opts)
	require.Nil(s.T(), err)
	vc, ok := mc.(*VirtualConnection)
	assert.True(s.T(), ok)
	<-readTrigger                     // Wait for the first read.
	require.Nil(s.T(), ts.ln.Close()) // Then close the server.
	readErr <- errAlwaysFail          // Fail the first read to trigger reconnection.
	require.Eventually(s.T(),
		func() bool { return maxReconnectCount+1 == vc.conn.reconnectCount },
		time.Second, 10*time.Millisecond)
}

func (s *msuite) TestMultiplexedReconnectOnReadError() {
	preInitialBackoff := initialBackoff
	preMaxBackoff := maxBackoff
	preMaxReconnectCount := maxReconnectCount
	preResetInterval := reconnectCountResetInterval
	defer func() {
		initialBackoff = preInitialBackoff
		maxBackoff = preMaxBackoff
		maxReconnectCount = preMaxReconnectCount
		reconnectCountResetInterval = preResetInterval
	}()
	initialBackoff = time.Microsecond
	maxBackoff = 50 * time.Microsecond
	maxReconnectCount = 5
	reconnectCountResetInterval = time.Hour

	m := New(
		WithConnectNumber(1),
		// On windows, it will try to use up all the timeout to do the dialling.
		// So limit the dial timeout.
		WithDialTimeout(time.Millisecond*10),
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts := NewGetOptions()
	calledAt := make([]time.Time, 0, maxReconnectCount)
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	opts.WithFrameParser(&errFramerBuilder{readFrameCalledAt: &calledAt})
	mc, err := m.GetMuxConn(ctx, s.network, s.address, opts)
	require.Nil(s.T(), err)
	vc, ok := mc.(*VirtualConnection)
	assert.True(s.T(), ok)
	require.Eventually(s.T(),
		func() bool { return maxReconnectCount+1 == vc.conn.reconnectCount },
		3*time.Second, time.Second,
		fmt.Sprintf("final status: maxReconnectCount+1=%d, vc.conn.reconnectCount=%d",
			maxReconnectCount+1, vc.conn.reconnectCount))
	require.Eventually(s.T(),
		func() bool { return maxReconnectCount+1 == len(calledAt) },
		3*time.Second, 50*time.Millisecond,
		fmt.Sprintf("final status: maxReconnectCount+1=%d, len(calledAt)=%d",
			maxReconnectCount+1, len(calledAt)))
	var differences []float64
	for i := 1; i < len(calledAt); i++ {
		delay := calledAt[i].Sub(calledAt[i-1])
		expectedBackoff := (initialBackoff * time.Duration(i))
		s.T().Logf("calledAt delay: %2dms, expect: %2dms (between %d and %d)\n",
			delay.Milliseconds(), expectedBackoff.Milliseconds(), i-1, i)
		differences = append(differences, float64(delay-expectedBackoff))
	}
	require.Equal(s.T(), maxReconnectCount+1, len(calledAt),
		"the actual times called is %d, expect %d", len(calledAt), maxReconnectCount+1)
	s.T().Logf("differences: %+v", differences)
	s.T().Logf("mean of differences between real retry delay and the calculated backoff: %vns", mean(differences))
	ss := std(differences)
	s.T().Logf("std of differences between real retry delay and the calculated backoff: %vns", ss)
	const expectedStdLimit = time.Second
	require.Less(s.T(), ss, float64(expectedStdLimit),
		"standard deviation of differences between real retry delay and calculated backoff is expected to be within %s",
		expectedStdLimit)
}

func (s *msuite) TestMultiplexedReconnectOnWriteError() {
	ctx := context.Background()
	ts := newTCPServer()
	ts.start(ctx)
	defer ts.stop()
	m := New(
		WithConnectNumber(1),
		// On windows, it will try to use up all the timeout to do the dialling.
		// So limit the dial timeout.
		WithDialTimeout(time.Millisecond*10),
	)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	opts := NewGetOptions()
	opts.WithVID(atomic.AddUint32(&s.requestID, 1))
	readTrigger := make(chan struct{})
	readErr := make(chan error)
	opts.WithFrameParser(&triggeredReadFramerBuilder{readTrigger: readTrigger, readErr: readErr})
	mc, err := m.GetMuxConn(ctx, s.network, ts.ln.Addr().String(), opts)
	require.Nil(s.T(), err)
	vc, ok := mc.(*VirtualConnection)
	assert.True(s.T(), ok)
	<-readTrigger                                    // Wait for the first read.
	require.Nil(s.T(), vc.conn.getRawConn().Close()) // Now close the underlying connection.
	require.Nil(s.T(), vc.Write([]byte("hello")))    // Then this write will trigger a reconnection on write error.
	// Now we are cool to check that a reconnection is triggered.
	require.Eventually(s.T(),
		func() bool { return 1 == vc.conn.reconnectCount },
		time.Second, 10*time.Millisecond)
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
		// replace the too long default 1s dail timeout.
		WithDialTimeout(dialTimeout))
	getVirtualConn := func(requestID uint32) (MuxConn, error) {
		getOptions := NewGetOptions()
		getOptions.WithVID(requestID)
		getOptions.WithFrameParser(&fb)
		return m.GetMuxConn(context.Background(), l.Addr().Network(), l.Addr().String(), getOptions)
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
	time.Sleep((maxBackoff + dialTimeout) * time.Duration(maxReconnectCount))
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

func (fb *errFramerBuilder) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	*fb.readFrameCalledAt = append(*fb.readFrameCalledAt, time.Now())
	buf, err = fb.New(rc).ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	return 0, buf, nil
}

var errAlwaysFail = errors.New("always fail")

type errFramer struct {
	calledAt *[]time.Time
}

// ReadFrame implements codec.Framer.
func (f *errFramer) ReadFrame() ([]byte, error) {
	return nil, errAlwaysFail
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

func (fb *triggeredReadFramerBuilder) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	buf, err = fb.New(rc).ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	return 0, buf, nil
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

func (fb *fixedLenFrameBuilder) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	buf = make([]byte, 4+fb.packetLen)
	n, err := rc.Read(buf)
	if err != nil {
		return 0, nil, err
	}
	id, bts, err := fb.Decode(buf[:n])
	if err != nil {
		return 0, nil, err
	}
	return id, bts, nil
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
