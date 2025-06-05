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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"trpc.group/trpc-go/trpc-go"
	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/internal/rpczenable"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestNewServerTransport(t *testing.T) {
	st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
	assert.NotNil(t, st)
}

func TestTCPListenAndServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServiceName("test name"),
		)

		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()

	// Round trip.
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{}))
	assert.NotNil(t, err)
}

func TestTCPTLSListenAndServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	// Wait until server transport ready.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport()
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.pem"),
		)

		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()

	// Round trip.
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{}),
		transport.WithDialTLS("../testdata/client.crt", "../testdata/client.key", "../testdata/ca.pem", "localhost"))
	assert.Nil(t, err)

	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{}),
		transport.WithDialTLS("../testdata/client.crt", "../testdata/client.key", "none", ""))
	assert.Nil(t, err)
}

func TestHandleError(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	// Wait until server transport is ready.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp"),
			transport.WithUDPListener(ln),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
		)

		if err != nil {
			t.Logf("test fail: %v", err)
		}
	}()
	wg.Wait()

	// Round trip.
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("test fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()
	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.LocalAddr().Network()),
		transport.WithDialAddress(ln.LocalAddr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{}))
	assert.NotNil(t, err)
}

func TestNewServerTransport_NotSupport(t *testing.T) {
	st := transport.NewServerTransport()
	err := st.ListenAndServe(context.Background(), transport.WithListenNetwork("unix"))
	assert.NotNil(t, err)

	err = st.ListenAndServe(context.Background(), transport.WithListenNetwork("xxx"))
	assert.NotNil(t, err)
}

func TestServerTransport_ListenAndServeUDP(t *testing.T) {
	// NoReusePort
	st := transport.NewServerTransport(transport.WithReusePort(false),
		transport.WithKeepAlivePeriod(time.Minute))
	err := st.ListenAndServe(
		context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithServerFramerBuilder(&framerBuilder{}),
	)
	assert.Nil(t, err)

	st = transport.NewServerTransport(transport.WithReusePort(true))
	err = st.ListenAndServe(
		context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithServerFramerBuilder(&framerBuilder{}),
	)
	assert.Nil(t, err)

	st = transport.NewServerTransport(transport.WithReusePort(true))
	err = st.ListenAndServe(
		context.Background(),
		transport.WithListenNetwork("ip"),
		transport.WithServerFramerBuilder(&framerBuilder{}),
	)
	assert.NotNil(t, err)
}

func TestServerTransport_ListenAndServe(t *testing.T) {
	// NoFramerBuilder
	st := transport.NewServerTransport(transport.WithReusePort(false))
	err := st.ListenAndServe(context.Background(), transport.WithListenNetwork("tcp"))
	assert.NotNil(t, err)

	fb := transport.GetFramerBuilder("trpc")
	// NoReusePort
	st = transport.NewServerTransport(transport.WithReusePort(false))
	err = st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("tcp"),
		transport.WithServerFramerBuilder(fb))
	assert.Nil(t, err)

	// ReusePort
	st = transport.NewServerTransport(transport.WithReusePort(true))
	err = st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("tcp"),
		transport.WithServerFramerBuilder(fb))
	assert.Nil(t, err)

	// Listener
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	st = transport.NewServerTransport()
	err = st.ListenAndServe(context.Background(),
		transport.WithListener(lis),
		transport.WithServerFramerBuilder(fb))
	assert.Nil(t, err)
	lis.Close()

	// ReusePort + Listen Error
	st = transport.NewServerTransport(transport.WithReusePort(true))
	err = st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("tcperror"),
		transport.WithServerFramerBuilder(fb))
	assert.NotNil(t, err)

	// context cancel
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	st = transport.NewServerTransport(transport.WithReusePort(true))
	err = st.ListenAndServe(ctx, transport.WithListenNetwork("tcp"), transport.WithServerFramerBuilder(fb))
	assert.Nil(t, err)
}

func TestServerTransport_ListenAndServeBothUDPAndTCP(t *testing.T) {
	fb := transport.GetFramerBuilder("trpc")
	// Empty network.
	network := ""
	st := transport.NewServerTransport()
	err := st.ListenAndServe(context.Background(), transport.WithListenNetwork(network))
	assert.EqualError(t, err, "server transport: not support network type "+network)

	// Another unknown wrong input.
	network = "wrong_type"
	st = transport.NewServerTransport()
	err = st.ListenAndServe(context.Background(), transport.WithListenNetwork(network))
	assert.EqualError(t, err, "server transport: not support network type "+network)

	// Right input.
	network = "tcp,udp"
	// No reuse.
	st = transport.NewServerTransport(transport.WithReusePort(false))
	err = st.ListenAndServe(context.Background(),
		transport.WithListenNetwork(network),
		transport.WithServerFramerBuilder(fb))
	assert.Nil(t, err)
}

// TestTCPListenAndServeAsync tests asynchronous server process.
func TestTCPListenAndServeAsync(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
			transport.WithWritev(true),
		)

		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()

	// round trip
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{}))
	assert.NotNil(t, err)
}

// TestTCPListenAndServerRoutinePool tests serving with goroutine pool.
func TestTCPListenAndServerRoutinePool(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListenAddress(ln.Addr().String()),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
			transport.WithMaxRoutines(100),
		)

		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()

	// round trip
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{}))
	assert.NotNil(t, err)
}

func TestWithReusePort(t *testing.T) {
	opts := &transport.ServerTransportOptions{}
	require.False(t, opts.ReusePort)

	opt := transport.WithReusePort(true)
	require.NotNil(t, opt)
	opt(opts)
	if runtime.GOOS != "windows" {
		require.True(t, opts.ReusePort)
	} else {
		require.False(t, opts.ReusePort)
	}

	opt = transport.WithReusePort(false)
	require.NotNil(t, opt)
	opt(opts)
	require.False(t, opts.ReusePort)
}

func TestWithRecvMsgChannelSize(t *testing.T) {
	opt := transport.WithRecvMsgChannelSize(1000)
	assert.NotNil(t, opt)
	opts := &transport.ServerTransportOptions{}
	opt(opts)
	assert.Equal(t, 1000, opts.RecvMsgChannelSize)
}

func TestWithSendMsgChannelSize(t *testing.T) {
	opt := transport.WithSendMsgChannelSize(1000)
	assert.NotNil(t, opt)
	opts := &transport.ServerTransportOptions{}
	opt(opts)
	assert.Equal(t, 1000, opts.SendMsgChannelSize)
}

func TestWithRecvUDPPacketBufferSize(t *testing.T) {
	opt := transport.WithRecvUDPPacketBufferSize(1000)
	assert.NotNil(t, opt)
	opts := &transport.ServerTransportOptions{}
	opt(opts)
	assert.Equal(t, 1000, opts.RecvUDPPacketBufferSize)
}

func TestWithRecvUDPRawSocketBufSize(t *testing.T) {
	opt := transport.WithRecvUDPRawSocketBufSize(1000)
	assert.NotNil(t, opt)
	opts := &transport.ServerTransportOptions{}
	opt(opts)
	assert.Equal(t, 1000, opts.RecvUDPRawSocketBufSize)
}

func TestWithIdleTimeout(t *testing.T) {
	opt := transport.WithIdleTimeout(time.Second)
	assert.NotNil(t, opt)
	opts := &transport.ServerTransportOptions{}
	opt(opts)
	assert.Equal(t, time.Second, opts.IdleTimeout)
}

func TestWithKeepAlivePeriod(t *testing.T) {
	opt := transport.WithKeepAlivePeriod(time.Minute)
	assert.NotNil(t, opt)
	opts := &transport.ServerTransportOptions{}
	opt(opts)
	assert.Equal(t, time.Minute, opts.KeepAlivePeriod)
}

func TestWithServeTLS(t *testing.T) {
	opt := transport.WithServeTLS("certfile", "keyfile", "")
	assert.NotNil(t, opt)
	opts := &transport.ListenServeOptions{}
	opt(opts)
	assert.Equal(t, "certfile", opts.TLSCertFile)
	assert.Equal(t, "keyfile", opts.TLSKeyFile)
}

// TestWithServeAsync tests setting server async.
func TestWithServeAsync(t *testing.T) {
	opt := transport.WithServerAsync(true)
	assert.NotNil(t, opt)
	opts := &transport.ListenServeOptions{}
	opt(opts)
	assert.Equal(t, true, opts.ServerAsync)
}

// TestWithWritev tests setting writev.
func TestWithWritev(t *testing.T) {
	opt := transport.WithWritev(true)
	assert.NotNil(t, opt)
	opts := &transport.ListenServeOptions{}
	opt(opts)
	assert.Equal(t, true, opts.Writev)
}

// TestWithMaxRoutine tests setting max number of goroutines.
func TestWithMaxRoutine(t *testing.T) {
	opt := transport.WithMaxRoutines(100)
	assert.NotNil(t, opt)
	opts := &transport.ListenServeOptions{}
	opt(opts)
	assert.Equal(t, 100, opts.Routines)
}

// TestTCPServerClosed tests if TCP listener can be closed immediately.
func TestTCPListenerClosed(t *testing.T) {
	err := tryCloseTCPListener(false)
	if err != nil {
		t.Errorf("close tcp listener err: %v", err)
	}
}

// TestTCPListenerClosed_WithReuseport tests if TCP listener can be closed immediately.
func TestTCPListenerClosed_WithReuseport(t *testing.T) {
	err := tryCloseTCPListener(true)
	if err != nil {
		t.Errorf("close tcp listener (with reuseport) err: %v", err)
	}
}

func tryCloseTCPListener(reuseport bool) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	var prepareErr error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithReusePort(reuseport))
		if err := st.ListenAndServe(ctx,
			transport.WithListener(ln),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
		); err != nil {
			prepareErr = err
		}
	}()
	wg.Wait()

	if prepareErr != nil {
		cancel()
		return fmt.Errorf("prepare listener error: %v", prepareErr)
	}

	// First time dial, should work.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		cancel()
		return fmt.Errorf("tcp dial error: %v", err)
	}
	conn.Close()

	// Notify and wait server close.
	cancel()
	time.Sleep(5 * time.Millisecond)

	// Second time dial, must fail.
	_, err = net.DialTimeout("tcp", ln.Addr().String(), 10*time.Millisecond)
	if err == nil {
		return fmt.Errorf("tcp dial (2nd time) want error")
	}
	return nil
}

func TestGetListenersFds(t *testing.T) {
	ListenFds := transport.GetListenersFds()
	assert.NotNil(t, ListenFds)
}

var savedListenerPort int

func TestSaveListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	require.Nil(t, transport.SaveListener(&NewPacketConn{}))
	require.Nil(t, transport.SaveListener(ln))
}

func TestTCPSeverErr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	st := transport.NewServerTransport()
	require.Nil(t, st.ListenAndServe(context.Background(),
		transport.WithListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{})))
}

func TestUDPServerErr(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()
	st := transport.NewServerTransport()
	require.Nil(t, st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{})))
}

type fakeListen struct {
}

func (c *fakeListen) Accept() (net.Conn, error) {
	return nil, &netError{errors.New("network failure")}
}
func (c *fakeListen) Close() error {
	return nil
}

func (c *fakeListen) Addr() net.Addr {
	return nil
}

func TestTCPServerConErr(t *testing.T) {
	go func() {
		fb := transport.GetFramerBuilder("trpc")
		st := transport.NewServerTransport()
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(&fakeListen{}),
			transport.WithServerFramerBuilder(fb))
		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
}

func TestUDPServerConErr(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()
	fb := transport.GetFramerBuilder("trpc")
	st := transport.NewServerTransport()
	require.Nil(t, st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithServerFramerBuilder(fb)))
}

func TestTCPWriteToClosedConn(t *testing.T) {
	l, err := net.Listen("tcp4", "localhost:0")
	require.Nil(t, err)
	defer l.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(l),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
		)
		assert.Nil(t, err)
	}()
	wg.Wait()
	conn, err := net.Dial("tcp4", l.Addr().String())
	require.Nil(t, err)
	require.Nil(t, conn.Close())
	_, err = conn.Write([]byte("data"))
	require.Contains(t, errs.Msg(err), "use of closed network connection")
}

func TestTCPServerHandleErrAndClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
		)
		assert.Nil(t, err)
	}()
	wg.Wait()

	// First time dial, should work.
	conn, err := net.Dial("tcp", ln.Addr().String())
	assert.Nil(t, err)
	time.Sleep(time.Millisecond * 5)
	data := []byte("hello world")
	req := make([]byte, 4)
	binary.BigEndian.PutUint32(req, uint32(len(data)))
	req = append(req, data...)
	_, err = conn.Write(req)
	assert.Nil(t, err)

	// Check the connection is closed by server.
	time.Sleep(time.Millisecond * 5)
	out := make([]byte, 8)
	_, err = conn.Read(out)
	assert.NotNil(t, err)
}

func getFreeAddr(network string) string {
	p, err := getFreePort(network)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(":%d", p)
}

func getFreePort(network string) (int, error) {
	if network == "tcp" || network == "tcp4" || network == "tcp6" {
		addr, err := net.ResolveTCPAddr(network, "localhost:0")
		if err != nil {
			return -1, err
		}

		l, err := net.ListenTCP(network, addr)
		if err != nil {
			return -1, err
		}
		defer l.Close()

		return l.Addr().(*net.TCPAddr).Port, nil
	}

	if network == "udp" || network == "udp4" || network == "udp6" {
		addr, err := net.ResolveUDPAddr(network, "localhost:0")
		if err != nil {
			return -1, err
		}

		l, err := net.ListenUDP(network, addr)
		if err != nil {
			return -1, err
		}
		defer l.Close()

		return l.LocalAddr().(*net.UDPAddr).Port, nil
	}

	return -1, errors.New("invalid network")
}

// TestTCPListenAndServeWithSafeFramer tests that we support safe framer without copying packages.
func TestUDPListenAndServeWithSafeFramer(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp"),
			transport.WithUDPListener(ln),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		)
		assert.Nil(t, err)
		time.Sleep(20 * time.Millisecond)
	}()
	wg.Wait()

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	rspData, err := transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.LocalAddr().Network()),
		transport.WithDialAddress(ln.LocalAddr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)

	length := binary.BigEndian.Uint32(rspData[:4])
	helloRsp := &helloResponse{}
	err = json.Unmarshal(rspData[4:4+length], helloRsp)
	assert.Nil(t, err)
	assert.Equal(t, helloRsp.Msg, "HelloWorld")
}

// TestTCPListenAndServeWithSafeFramer tests that frame is not copied when Framer is already safe.
func TestTCPListenAndServeWithSafeFramer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
			transport.WithServerAsync(true),
		)
		assert.Nil(t, err)
		time.Sleep(20 * time.Millisecond)
	}()
	wg.Wait()

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	rspData, err := transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithDialAddress(ln.Addr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)

	length := binary.BigEndian.Uint32(rspData[:4])
	helloRsp := &helloResponse{}
	err = json.Unmarshal(rspData[4:4+length], helloRsp)
	assert.Nil(t, err)
	assert.Equal(t, helloRsp.Msg, "HelloWorld")
}

func TestWithDisableKeepAlives(t *testing.T) {
	disable := true
	o := transport.WithDisableKeepAlives(true)
	opts := &transport.ListenServeOptions{}
	o(opts)
	assert.Equal(t, disable, opts.DisableKeepAlives)
}

func TestWithServerIdleTimeout(t *testing.T) {
	idleTimeout := time.Second
	o := transport.WithServerIdleTimeout(idleTimeout)
	opts := &transport.ListenServeOptions{}
	o(opts)
	assert.Equal(t, opts.IdleTimeout, idleTimeout)
}

func TestWithServerReadTimeout(t *testing.T) {
	readTimeout := time.Second
	o := transport.WithServerReadTimeout(readTimeout)
	opts := &transport.ListenServeOptions{}
	o(opts)
	assert.Equal(t, opts.ReadTimeout, readTimeout)
}

func TestWithServiceActiveCnt(t *testing.T) {
	var cnt int64
	var o transport.ListenServeOptions
	transport.WithServiceActiveCnt(&cnt)(&o)
	o.ActiveCnt.Add(2)
	require.Equal(t, int64(2), cnt)
	o.ActiveCnt.Add(-3)
	require.Equal(t, int64(-1), cnt)
}

func TestUDPServeClose(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()
	ts := transport.NewServerTransport()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Nil(t, ts.ListenAndServe(
		ctx,
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(true),
	))
	time.Sleep(100 * time.Millisecond)
}

type MockUDPError struct{}

func (e MockUDPError) Error() string   { return "mock udp error" }
func (e MockUDPError) Timeout() bool   { return false }
func (e MockUDPError) Temporary() bool { return true }

func TestUDPReadError(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	require.Nil(t, transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(false),
	))
	time.Sleep(60 * time.Millisecond)
}

func TestUDPWriteError(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	require.Nil(t, transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(false),
	))
	time.Sleep(20 * time.Millisecond)

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.LocalAddr().Network()),
		transport.WithDialAddress(ln.LocalAddr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)
}

func TestPoolInvokeFail(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	require.Nil(t, transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(true),
	))
	time.Sleep(20 * time.Millisecond)

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.LocalAddr().Network()),
		transport.WithDialAddress(ln.LocalAddr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)
}

func TestCreatePoolFail(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	defer ln.Close()

	require.Nil(t, transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithUDPListener(ln),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(true),
	))

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.LocalAddr().Network()),
		transport.WithDialAddress(ln.LocalAddr().String()),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)
}

func TestListenAndServeTLSFail(t *testing.T) {
	s := transport.NewServerTransport()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	require.NotNil(t, s.ListenAndServe(ctx,
		transport.WithListenNetwork("tcp"),
		transport.WithServeTLS("fakeCertFileName", "fakeKeyFileName", "fakeCAFileName"),
		transport.WithServerFramerBuilder(&framerBuilder{}),
		transport.WithListener(ln),
	))
}

func TestListenAndServeWithStopListener(t *testing.T) {
	s := transport.NewServerTransport()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	ch := make(chan struct{})
	require.Nil(t, s.ListenAndServe(ctx,
		transport.WithListenNetwork("tcp"),
		transport.WithServerFramerBuilder(&framerBuilder{}),
		transport.WithListener(ln),
		transport.WithStopListening(ch),
	))
	_, err = net.Dial("tcp", ln.Addr().String())
	require.Nil(t, err)
	close(ch)
	time.Sleep(time.Millisecond)
	_, err = net.Dial("tcp", ln.Addr().String())
	require.NotNil(t, err)
}

func TestServerTransportReadTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerAsync(true),
			transport.WithServerReadTimeout(time.Second),
			transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		)
		assert.Nil(t, err)
		time.Sleep(20 * time.Millisecond)
	}()
	wg.Wait()

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail: %v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	rspData, err := transport.RoundTrip(ctx, reqData,
		transport.WithDialNetwork(ln.Addr().Network()),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}),
		transport.WithDialAddress(ln.Addr().String()))
	require.Nil(t, err)

	length := binary.BigEndian.Uint32(rspData[:4])
	helloRsp := &helloResponse{}
	require.Nil(t, json.Unmarshal(rspData[4:4+length], helloRsp))
	require.Equal(t, helloRsp.Msg, "HelloWorld")
}

func TestTCPListenAndServeKeepOrderPreDecode(t *testing.T) {
	old := rpczenable.Enabled
	defer func() {
		rpczenable.Enabled = old
	}()
	rpczenable.Enabled = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	h := &preDecodeHandler{
		values: make(map[string][]string),
	}

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	metaDataKey := "meta_key"
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(h),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithServiceName(t.Name()),
			transport.WithServerAsync(true),
			// Without this option, the keep-order feature will be disabled,
			// and this test case will fail.
			transport.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
				msg := codec.Message(ctx)
				meta := msg.ServerMetaData()
				if meta == nil {
					log.Printf("meta data is nil for %q\n", reqBody)
					return "", false
				}
				return string(meta[metaDataKey]), true
			}),
		)

		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()
	sendKeepOrderPreDecodeReq(t, ln.Addr().String(), metaDataKey, assertRspWithKeepOrder)
}

func TestTCPListenAndServeKeepOrderPreDecodeFail(t *testing.T) {
	// test extract key fail and fallback to non-keep-order scenario
	old := rpczenable.Enabled
	defer func() {
		rpczenable.Enabled = old
	}()
	rpczenable.Enabled = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	h := &preDecodeHandler{
		values: make(map[string][]string),
	}

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	metaDataKey := "meta_key"
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(h),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithServiceName(t.Name()),
			transport.WithServerAsync(true),
			transport.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
				return "", false
			}),
		)
		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()
	sendKeepOrderPreDecodeReq(t, ln.Addr().String(), metaDataKey, assertRspWithKeepOrderFail)
}

func sendKeepOrderPreDecodeReq(
	t *testing.T,
	addr string,
	metaDataKey string,
	rsp_checker func(t *testing.T, rsp_count int, keys []string, rsps map[string]string),
) {
	var (
		mu        sync.Mutex
		eg        errgroup.Group
		requestID uint32
	)
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	count := 10
	rsps := make(map[string]string)
	p := multiplexed.New(multiplexed.WithConnectNumber(1))
	for _, key := range keys {
		key := key
		for i := 0; i < count; i++ {
			i := i
			var (
				rsp []byte
				err error
			)
			ech := make(chan error, 1)
			ctx := keeporder.NewContextWithClientInfo(trpc.BackgroundContext(), &keeporder.ClientInfo{
				SendError: ech,
			})
			eg.Go(func() error {
				ctx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()
				msg := codec.Message(ctx)
				msg.WithRequestID(atomic.AddUint32(&requestID, 1))
				msg.WithClientMetaData(codec.MetaData{
					metaDataKey: []byte(key),
				})
				data := []byte(key + " " + strconv.Itoa(i))
				var reqData []byte
				reqData, err = trpc.DefaultClientCodec.Encode(msg, data)
				if err != nil {
					return fmt.Errorf("client codec encode err: %+v", err)
				}
				rsp, err = transport.RoundTrip(ctx, reqData,
					transport.WithDialNetwork("tcp"),
					transport.WithDialAddress(addr),
					transport.WithMultiplexedPool(p),
					transport.WithMsg(msg),
					transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder),
				)
				select {
				case ech <- err: // If the error is generated before transport write, this case will be executed.
				default:
				}
				if err != nil {
					return err
				}
				// Only store the final result.
				mu.Lock()
				s := string(rsp)
				if len(rsps[key]) < len(s) {
					rsps[key] = s
				}
				mu.Unlock()
				return err
			})
			if err := <-ech; err != nil {
				t.Errorf("request %q failed: %v", key, err)
			}
		}
	}
	require.NoError(t, eg.Wait())
	rsp_checker(t, count, keys, rsps)
}

type preDecodeHandler struct {
	mu     sync.Mutex
	values map[string][]string
}

func (h *preDecodeHandler) Handle(ctx context.Context, req []byte) ([]byte, error) {
	msg := codec.Message(ctx)
	req, err := trpc.DefaultServerCodec.Decode(msg, req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request %q: %v", req, err)
	}
	s := string(req)
	ss := strings.Split(s, " ")
	if len(ss) != 2 {
		return nil, fmt.Errorf("invalid request %q, should of format `key value`", req)
	}
	key, val := ss[0], ss[1]
	cnt, err := strconv.Atoi(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value %q, should be an integer", val)
	}
	// Sleep the amount of time that is inverse proportional to the count
	// to confuse result when keep-order feature is not enabled.
	time.Sleep(time.Duration(int32(10-cnt)*10) * time.Millisecond)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values[key] = append(h.values[key], val)
	body := []byte(strings.Join(h.values[key], " "))
	rsp, err := trpc.DefaultServerCodec.Encode(msg, body)
	return rsp, err
}

func (h *preDecodeHandler) PreDecode(ctx context.Context, reqBuf []byte) (reqBodyBuf []byte, err error) {
	msg := codec.Message(ctx)
	return trpc.DefaultServerCodec.Decode(msg, reqBuf)
}

func TestTCPListenAndServeKeepOrderPreUnmarshal(t *testing.T) {
	old := rpczenable.Enabled
	defer func() {
		rpczenable.Enabled = old
	}()
	rpczenable.Enabled = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	h := &preUnmarshalHandler{
		values: make(map[string][]string),
	}

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(h),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithServiceName(t.Name()),
			transport.WithServerAsync(true),
			// Without this option, the keep-order feature will be disabled,
			// and this test case will fail.
			transport.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
				request, ok := req.([]byte)
				if !ok {
					log.Printf("invalid request type %T, want []byte", req)
					return "", false
				}
				ss := strings.Split(string(request), " ")
				if len(ss) != 2 {
					log.Printf("invalid request %q, should be of format `key count`", request)
					return "", false
				}
				return ss[0], true
			}),
		)
		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()
	sendKeepOrderPreUnmarshalReq(t, ln.Addr().String(), assertRspWithKeepOrder)
}

func TestTCPListenAndServeKeepOrderPreUnmarshalFail(t *testing.T) {
	// test extract key fail and fallback to non-keep-order scenario
	old := rpczenable.Enabled
	defer func() {
		rpczenable.Enabled = old
	}()
	rpczenable.Enabled = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()

	h := &preUnmarshalHandler{
		values: make(map[string][]string),
	}

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListener(ln),
			transport.WithHandler(h),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithServiceName(t.Name()),
			transport.WithServerAsync(true),
			transport.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
				return "", false
			}),
		)
		if err != nil {
			t.Logf("ListenAndServe fail: %v", err)
		}
	}()
	wg.Wait()
	sendKeepOrderPreUnmarshalReq(t, ln.Addr().String(), assertRspWithKeepOrderFail)
}

func sendKeepOrderPreUnmarshalReq(
	t *testing.T,
	addr string,
	rsp_checker func(t *testing.T, rsp_count int, keys []string, rsps map[string]string),
) {
	var (
		mu        sync.Mutex
		eg        errgroup.Group
		requestID uint32
	)
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	count := 10
	rsps := make(map[string]string)
	p := multiplexed.New(multiplexed.WithConnectNumber(1))
	for _, key := range keys {
		key := key
		for i := 0; i < count; i++ {
			i := i
			var (
				rsp []byte
				err error
			)
			ech := make(chan error, 1)
			ctx := keeporder.NewContextWithClientInfo(trpc.BackgroundContext(), &keeporder.ClientInfo{
				SendError: ech,
			})
			eg.Go(func() error {
				ctx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()
				msg := codec.Message(ctx)
				msg.WithRequestID(atomic.AddUint32(&requestID, 1))
				data := []byte(key + " " + strconv.Itoa(i))
				var reqData []byte
				reqData, err = trpc.DefaultClientCodec.Encode(msg, data)
				if err != nil {
					return fmt.Errorf("client codec encode err: %+v", err)
				}
				rsp, err = transport.RoundTrip(ctx, reqData,
					transport.WithDialNetwork("tcp"),
					transport.WithDialAddress(addr),
					transport.WithMultiplexedPool(p),
					transport.WithMsg(msg),
					transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder),
				)
				select {
				case ech <- err: // If the error is generated before transport write, this case will be executed.
				default:
				}
				if err != nil {
					return err
				}
				// Only store the final result.
				mu.Lock()
				s := string(rsp)
				if len(rsps[key]) < len(s) {
					rsps[key] = s
				}
				mu.Unlock()
				return err
			})
			if err := <-ech; err != nil {
				t.Errorf("request %q failed: %v", key, err)
			}
		}
	}
	require.NoError(t, eg.Wait())
	rsp_checker(t, count, keys, rsps)
}

type preUnmarshalHandler struct {
	mu     sync.Mutex
	values map[string][]string
}

func (h *preUnmarshalHandler) Handle(ctx context.Context, req []byte) ([]byte, error) {
	msg := codec.Message(ctx)
	req, err := trpc.DefaultServerCodec.Decode(msg, req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request %q: %v", req, err)
	}
	s := string(req)
	ss := strings.Split(s, " ")
	if len(ss) != 2 {
		return nil, fmt.Errorf("invalid request %q, should of format `key value`", req)
	}
	key, val := ss[0], ss[1]
	cnt, err := strconv.Atoi(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value %q, should be an integer", val)
	}
	// Sleep the amount of time that is inverse proportional to the count
	// to confuse result when keep-order feature is not enabled.
	time.Sleep(time.Duration(int32(10-cnt)*10) * time.Millisecond)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values[key] = append(h.values[key], val)
	body := []byte(strings.Join(h.values[key], " "))
	rsp, err := trpc.DefaultServerCodec.Encode(msg, body)
	return rsp, err
}

func (h *preUnmarshalHandler) PreUnmarshal(ctx context.Context, reqBuf []byte) (req interface{}, err error) {
	msg := codec.Message(ctx)
	return trpc.DefaultServerCodec.Decode(msg, reqBuf)
}

func assertRspWithKeepOrder(t *testing.T, rsp_count int, rsp_keys []string, rsps map[string]string) {
	expectSlice := make([]string, 0, rsp_count)
	for i := 0; i < rsp_count; i++ {
		expectSlice = append(expectSlice, strconv.Itoa(i))
	}
	expect := strings.Join(expectSlice, " ")

	// check if rsp is in the order when the keep-order req is processed successfully
	for _, key := range rsp_keys {
		require.Equal(t, expect, rsps[key])
	}
}

func assertRspWithKeepOrderFail(t *testing.T, rsp_count int, rsp_keys []string, rsps map[string]string) {
	expect := (rsp_count - 1) * rsp_count / 2
	// check if rsp correct when the keep-order req is processed failed
	for _, key := range rsp_keys {
		str_slice := strings.Split(rsps[key], " ")
		sum := 0
		for _, str_v := range str_slice {
			v, err := strconv.Atoi(str_v)
			require.Nil(t, err)
			sum += v
		}
		require.Equal(t, expect, sum)
	}
}
