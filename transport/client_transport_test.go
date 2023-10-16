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

package transport_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
)

func TestTcpRoundTripPoolNIl(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(nil)
	_, err := st.RoundTrip(context.Background(), []byte("hello"), optNetwork, optPool)
	assert.NotNil(t, err)
}

func TestTcpRoundTripTCPErr(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	pool := connpool.NewConnectionPool()
	optPool := transport.WithDialPool(pool)
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	optDisabled := transport.WithDisableConnectionPool()
	newCtx := context.Background()
	newCtx.Done()
	newCtx.Deadline()
	_, err := st.RoundTrip(newCtx, []byte("hello"), optNetwork, optPool, optFramerBuilder, optDisabled)
	assert.NotNil(t, err)
}

func TestTcpRoundTripCTXErr(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	pool := connpool.NewConnectionPool()
	optPool := transport.WithDialPool(pool)
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	_, err := st.RoundTrip(context.Background(), []byte("hello"), optNetwork, optPool, optFramerBuilder)
	assert.NotNil(t, err)
}

type fakePool struct {
}

func (p *fakePool) Get(network string, address string, opts connpool.GetOptions) (net.Conn, error) {
	return &fakeConn{}, nil
}

type fakeConn struct {
}

func (c *fakeConn) Close() error {
	return nil
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

type netError struct {
	error
}

// Timeout() bool
// Temporary() bool
func (c *netError) Timeout() bool {
	return true
}
func (c *netError) Temporary() bool {
	return true
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
	if Count == 1 {
		return 0, errors.New("write failure")
	}
	if Count == 2 {
		return 0, netError{errors.New("net failure")}
	}
	return len(b), nil
}

func (c *fakeConn) LocalAddr() net.Addr {
	return nil
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8888}
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestTcpRoundTripReadFrameNil(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(&fakePool{})
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	optReqType := transport.WithReqType(transport.SendOnly)
	optAddress := transport.WithDialAddress(":8888")
	_, err := st.RoundTrip(context.Background(), []byte("hello"), optNetwork, optPool, optFramerBuilder,
		optReqType, optAddress)
	assert.NotNil(t, err)
}

func TestTCPRoundTripSetRemoteAddr(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(&fakePool{})
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	optAddress := transport.WithDialAddress("127.0.0.1:8888")
	ctx, msg := codec.WithNewMessage(context.Background())
	_, _ = st.RoundTrip(ctx, []byte("hello"), optNetwork, optPool, optFramerBuilder, optAddress)
	assert.NotNil(t, msg.RemoteAddr())
	assert.Equal(t, "127.0.0.1:8888", msg.RemoteAddr().String())
}

type newCtx struct {
}

var Count int64

func (c *newCtx) Deadline() (deadline time.Time, ok bool) {
	deadline = time.Now()
	return deadline, true
}
func (c *newCtx) Done() <-chan struct{} {
	return nil
}
func (c *newCtx) Err() error {
	if Count == 1 {
		return context.DeadlineExceeded
	}
	return context.Canceled
}
func (c *newCtx) Value(key interface{}) interface{} {
	return nil
}

func TestTcpRoundTripCanceled(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(&fakePool{})
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	optAddress := transport.WithDialAddress(":8888")
	_, err := st.RoundTrip(&newCtx{}, []byte("hello"), optNetwork, optPool, optFramerBuilder,
		optAddress)
	assert.NotNil(t, err)
}

func TestTcpRoundTripTimeout(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(&fakePool{})
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	optAddress := transport.WithDialAddress(":8888")
	Count = 1
	_, err := st.RoundTrip(&newCtx{}, []byte("hello"), optNetwork, optPool, optFramerBuilder,
		optAddress)
	assert.NotNil(t, err)
}

func TestTcpRoundTripConnWriteErr(t *testing.T) {
	st := transport.NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(&fakePool{})
	fb := &trpc.FramerBuilder{}
	optFramerBuilder := transport.WithClientFramerBuilder(fb)
	optAddress := transport.WithDialAddress(":8888")
	Count = 1
	_, err := st.RoundTrip(context.Background(), []byte("hello"), optNetwork, optPool, optFramerBuilder,
		optAddress)
	assert.NotNil(t, err)
	Count = 2
	_, err = st.RoundTrip(context.Background(), []byte("hello"), optNetwork, optPool, optFramerBuilder,
		optAddress)
	assert.NotNil(t, err)
}

type NewPacketConn struct {
}

func (c *NewPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	return 0, nil, nil
}
func (c *NewPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if Count == 1 {
		return len(p), errors.New("write failure")
	}
	return len(p), netError{errors.New("net failure")}
}
func (c *NewPacketConn) Close() error {
	return nil
}
func (c *NewPacketConn) LocalAddr() net.Addr {
	return nil
}
func (c *NewPacketConn) SetDeadline(t time.Time) error {
	return nil
}
func (c *NewPacketConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *NewPacketConn) SetWriteDeadline(t time.Time) error {
	return nil
}
func (c *NewPacketConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	return len(b), nil, netError{errors.New("net failure")}
}

func TestNewClientTransport(t *testing.T) {
	st := transport.NewClientTransport()
	assert.NotNil(t, st)
}

func TestWithDialPool(t *testing.T) {
	opt := transport.WithDialPool(nil)
	opts := &transport.RoundTripOptions{}
	opt(opts)
	assert.Equal(t, nil, opts.Pool)
}

func TestWithReqType(t *testing.T) {
	opt := transport.WithReqType(transport.SendOnly)
	opts := &transport.RoundTripOptions{}
	opt(opts)
	assert.Equal(t, transport.SendOnly, opts.ReqType)
}

type emptyPool struct {
}

func (p *emptyPool) Get(network string, address string, opts connpool.GetOptions) (net.Conn, error) {
	return nil, errors.New("empty")
}

var testReqByte = []byte{'a', 'b'}

func TestWithDialPoolError(t *testing.T) {
	ctx, f := context.WithTimeout(context.Background(), 3*time.Second)
	defer f()
	_, err := transport.RoundTrip(ctx, testReqByte,
		transport.WithDialPool(&emptyPool{}),
		transport.WithDialNetwork("tcp"))
	// fmt.Printf("err: %v", err)
	assert.NotNil(t, err)
}

func TestContextTimeout(t *testing.T) {
	ctx, f := context.WithTimeout(context.Background(), time.Millisecond)
	defer f()
	<-ctx.Done()
	fb := &trpc.FramerBuilder{}
	_, err := transport.RoundTrip(ctx, testReqByte,
		transport.WithDialNetwork("tcp"),
		transport.WithDialAddress(":8888"),
		transport.WithClientFramerBuilder(fb))
	assert.NotNil(t, err)
}

func TestContextTimeout_Multiplexed(t *testing.T) {
	ctx, f := context.WithTimeout(context.Background(), time.Millisecond)
	defer f()
	<-ctx.Done()
	fb := &trpc.FramerBuilder{}
	_, err := transport.RoundTrip(ctx, testReqByte,
		transport.WithDialNetwork("tcp"),
		transport.WithDialAddress(":8888"),
		transport.WithMultiplexed(true),
		transport.WithMsg(codec.Message(ctx)),
		transport.WithClientFramerBuilder(fb))
	assert.NotNil(t, err)
}

func TestContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cancel()
	fb := &trpc.FramerBuilder{}
	_, err := transport.RoundTrip(ctx, testReqByte,
		transport.WithDialNetwork("tcp"),
		transport.WithDialAddress(":8888"),
		transport.WithClientFramerBuilder(fb))
	assert.NotNil(t, err)
}

func TestWithReqTypeSendOnly(t *testing.T) {
	ctx, f := context.WithTimeout(context.Background(), 3*time.Second)
	defer f()
	_, err := transport.RoundTrip(ctx, []byte{},
		transport.WithReqType(transport.SendOnly),
		transport.WithDialNetwork("tcp"))
	// fmt.Printf("err: %v", err)
	assert.NotNil(t, err)
}

func TestClientTransport_RoundTrip(t *testing.T) {
	fb := &lengthDelimitedBuilder{}
	go func() {
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp"),
			transport.WithListenAddress("localhost:9998"),
			transport.WithHandler(&lengthDelimitedHandler{}),
			transport.WithServerFramerBuilder(fb),
		)
		assert.Nil(t, err)
	}()
	time.Sleep(20 * time.Millisecond)

	var err error
	_, err = transport.RoundTrip(context.Background(), encodeLengthDelimited("helloworld"))
	assert.NotNil(t, err)

	tc := transport.NewClientTransport()
	_, err = tc.RoundTrip(context.Background(), encodeLengthDelimited("helloworld"))
	assert.NotNil(t, err)

	// Test address invalid.
	_, err = tc.RoundTrip(context.Background(), encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("invalidaddress"),
		transport.WithReqType(transport.SendOnly))
	assert.NotNil(t, err)

	// Test send only.
	rsp, err := tc.RoundTrip(context.Background(), encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("localhost:9998"),
		transport.WithClientFramerBuilder(fb),
		transport.WithReqType(transport.SendOnly),
		transport.WithConnectionMode(transport.NotConnected))
	assert.NotNil(t, err)
	assert.Equal(t, errs.ErrClientNoResponse, err)
	assert.Nil(t, rsp)

	// Test multiplexed send only.
	ctx, msg := codec.WithNewMessage(context.Background())
	rsp, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithMultiplexed(true),
		transport.WithDialAddress("localhost:9998"),
		transport.WithReqType(transport.SendOnly),
		transport.WithClientFramerBuilder(fb),
		transport.WithMsg(msg),
	)
	assert.NotNil(t, err)
	assert.Equal(t, errs.ErrClientNoResponse, err)
	assert.Nil(t, rsp)

	// Test context canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithClientFramerBuilder(fb),
		transport.WithDialAddress("localhost:9998"))
	assert.EqualValues(t, err.(*errs.Error).Code, int32(errs.RetClientCanceled))

	// Test context timeout.
	ctx, timeout := context.WithTimeout(context.Background(), time.Millisecond)
	defer timeout()
	<-ctx.Done()
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithClientFramerBuilder(fb),
		transport.WithDialAddress("localhost:9998"))
	assert.EqualValues(t, err.(*errs.Error).Code, int32(errs.RetClientTimeout))

	// Test roundtrip.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rsp, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("localhost:9998"),
		transport.WithConnectionMode(transport.NotConnected),
		transport.WithClientFramerBuilder(fb),
	)
	assert.NotNil(t, rsp)
	assert.Nil(t, err)

	// Test setting RemoteAddr of UDP RoundTrip.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("127.0.0.1:9998"),
		transport.WithConnectionMode(transport.Connected),
		transport.WithClientFramerBuilder(fb),
	)
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:9998", msg.RemoteAddr().String())

	// Test local addr.
	localAddr := "127.0.0.1:"
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("127.0.0.1:9998"),
		transport.WithConnectionMode(transport.Connected),
		transport.WithClientFramerBuilder(fb),
		transport.WithLocalAddr(localAddr),
	)
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1", msg.LocalAddr().(*net.UDPAddr).IP.String())

	// Test local addr error.
	localAddr = "invalid address"
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx, msg = codec.WithNewMessage(ctx)
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("127.0.0.1:9998"),
		transport.WithConnectionMode(transport.Connected),
		transport.WithClientFramerBuilder(fb),
		transport.WithLocalAddr(localAddr),
	)
	assert.NotNil(t, err)
	assert.Nil(t, msg.LocalAddr())

	// Test readframer error.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("127.0.0.1:9998"),
		transport.WithConnectionMode(transport.Connected),
		transport.WithClientFramerBuilder(&lengthDelimitedBuilder{
			readError: true,
		}),
	)
	assert.Contains(t, err.Error(), readFrameError.Error())

	// Test readframe bytes remaining error.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = tc.RoundTrip(ctx, encodeLengthDelimited("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("127.0.0.1:9998"),
		transport.WithConnectionMode(transport.Connected),
		transport.WithClientFramerBuilder(&lengthDelimitedBuilder{
			remainingBytes: true,
		}),
	)
	assert.Contains(t, err.Error(), remainingBytesError.Error())
}

// Frame a stream of bytes based on a length prefix
// +------------+--------------------------------+
// | len: uint8 |          frame payload         |
// +------------+--------------------------------+
type lengthDelimitedBuilder struct {
	remainingBytes bool
	readError      bool
}

func (fb *lengthDelimitedBuilder) New(reader io.Reader) codec.Framer {
	return &lengthDelimited{
		readError:      fb.readError,
		remainingBytes: fb.remainingBytes,
		reader:         reader,
	}
}

func (fb *lengthDelimitedBuilder) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	buf, err = fb.New(rc).ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	return 0, buf, nil
}

type lengthDelimited struct {
	reader         io.Reader
	readError      bool
	remainingBytes bool
}

func encodeLengthDelimited(data string) []byte {
	result := []byte{byte(len(data))}
	result = append(result, []byte(data)...)
	return result
}

var (
	readFrameError      = errors.New("read framer error")
	remainingBytesError = fmt.Errorf(
		"packet data is not drained, the remaining %d will be dropped",
		remainingBytes,
	)
	remainingBytes = 1
)

func (f *lengthDelimited) ReadFrame() ([]byte, error) {
	if f.readError {
		return nil, readFrameError
	}
	head := make([]byte, 1)
	if _, err := io.ReadFull(f.reader, head); err != nil {
		return nil, err
	}
	bodyLen := int(head[0])
	if f.remainingBytes {
		bodyLen = bodyLen - remainingBytes
	}
	body := make([]byte, bodyLen)
	if _, err := io.ReadFull(f.reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

type lengthDelimitedHandler struct{}

func (h *lengthDelimitedHandler) Handle(ctx context.Context, req []byte) ([]byte, error) {
	rsp := make([]byte, len(req)+1)
	rsp[0] = byte(len(req))
	copy(rsp[1:], req)
	return rsp, nil
}

func TestClientTransport_MultiplexedErr(t *testing.T) {
	listener, err := net.Listen("tcp", ":")
	require.Nil(t, err)
	defer listener.Close()
	go func() {
		transport.ListenAndServe(
			transport.WithListener(listener),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(transport.GetFramerBuilder("trpc")),
		)
	}()
	time.Sleep(20 * time.Millisecond)

	tc := transport.NewClientTransport()
	fb := &trpc.FramerBuilder{}

	// Test multiplexed context timeout.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx, msg := codec.WithNewMessage(ctx)
	_, err = tc.RoundTrip(ctx, []byte("helloworld"),
		transport.WithDialNetwork(listener.Addr().Network()),
		transport.WithDialAddress(listener.Addr().String()),
		transport.WithMultiplexed(true),
		transport.WithClientFramerBuilder(fb),
		transport.WithMsg(msg),
	)
	assert.EqualValues(t, err.(*errs.Error).Code, int32(errs.RetClientTimeout))

	// Test multiplexed context canceled.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	go func() {
		time.Sleep(time.Millisecond * 200)
		cancel()
	}()
	_, err = tc.RoundTrip(ctx, []byte("helloworld"),
		transport.WithDialNetwork(listener.Addr().Network()),
		transport.WithDialAddress(listener.Addr().String()),
		transport.WithMultiplexed(true),
		transport.WithClientFramerBuilder(fb),
		transport.WithMsg(msg),
	)
	assert.EqualValues(t, err.(*errs.Error).Code, int32(errs.RetClientCanceled))
}

func TestClientTransport_RoundTrip_PreConnected(t *testing.T) {

	go func() {
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp"),
			transport.WithListenAddress("localhost:9999"),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(transport.GetFramerBuilder("trpc")),
		)
		assert.Nil(t, err)
	}()
	time.Sleep(20 * time.Millisecond)

	var err error
	_, err = transport.RoundTrip(context.Background(), []byte("helloworld"))
	assert.NotNil(t, err)

	tc := transport.NewClientTransport()

	// Test connected UDPConn.
	rsp, err := tc.RoundTrip(context.Background(), []byte("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("localhost:9999"),
		transport.WithDialPassword("passwd"),
		transport.WithClientFramerBuilder(&trpc.FramerBuilder{}),
		transport.WithReqType(transport.SendOnly),
		transport.WithConnectionMode(transport.Connected))
	assert.NotNil(t, err)
	assert.Equal(t, errs.ErrClientNoResponse, err)
	assert.Nil(t, rsp)

	// Test context done.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = tc.RoundTrip(ctx, []byte("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("localhost:9999"),
		transport.WithConnectionMode(transport.Connected))
	assert.NotNil(t, err)

	// Test RoundTrip.
	ctx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	rsp, err = tc.RoundTrip(ctx, []byte("helloworld"),
		transport.WithDialNetwork("udp"),
		transport.WithDialAddress("localhost:9999"),
		transport.WithConnectionMode(transport.Connected))
	assert.NotNil(t, err)
	assert.Nil(t, rsp)
}

func TestOptions(t *testing.T) {

	opts := &transport.RoundTripOptions{}

	o := transport.WithDialTLS("client.cert", "client.key", "ca.pem", "servername")
	o(opts)
	assert.Equal(t, "client.cert", opts.TLSCertFile)
	assert.Equal(t, "client.key", opts.TLSKeyFile)
	assert.Equal(t, "ca.pem", opts.CACertFile)
	assert.Equal(t, "servername", opts.TLSServerName)

	o = transport.WithDisableConnectionPool()
	o(opts)

	assert.True(t, opts.DisableConnectionPool)
}

// TestWithMultiplexedPool tests connection pool multiplexing.
func TestWithMultiplexedPool(t *testing.T) {
	opts := &transport.RoundTripOptions{}
	m := multiplexed.New(multiplexed.WithConnectNumber(10))
	o := transport.WithMultiplexedPool(m)
	o(opts)
	assert.True(t, opts.EnableMultiplexed)
	assert.Equal(t, opts.Multiplexed, m)
}

// TestUDPTransportFramerBuilderErr tests nil FramerBuilder error.
func TestUDPTransportFramerBuilderErr(t *testing.T) {
	opts := []transport.RoundTripOption{
		transport.WithDialNetwork("udp"),
	}
	ts := transport.NewClientTransport()
	_, err := ts.RoundTrip(context.Background(), nil, opts...)
	assert.EqualValues(t, err.(*errs.Error).Code, int32(errs.RetClientConnectFail))
}

// TestWithLocalAddr tests local addr.
func TestWithLocalAddr(t *testing.T) {
	opts := &transport.RoundTripOptions{}
	localAddr := "127.0.0.1:8080"
	o := transport.WithLocalAddr(localAddr)
	o(opts)
	assert.Equal(t, opts.LocalAddr, localAddr)
}

func TestWithDialTimeout(t *testing.T) {
	opts := &transport.RoundTripOptions{}
	timeout := time.Second
	o := transport.WithDialTimeout(timeout)
	o(opts)
	assert.Equal(t, opts.DialTimeout, timeout)
}

func TestWithProtocol(t *testing.T) {
	opts := &transport.RoundTripOptions{}
	protocol := "xxx-protocol"
	o := transport.WithProtocol(protocol)
	o(opts)
	assert.Equal(t, protocol, opts.Protocol)
}

func TestWithDisableEncodeTransInfoBase64(t *testing.T) {
	opts := &transport.ClientTransportOptions{}
	transport.WithDisableEncodeTransInfoBase64()(opts)
	assert.Equal(t, true, opts.DisableHTTPEncodeTransInfoBase64)
}
