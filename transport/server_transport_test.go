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
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestNewServerTransport(t *testing.T) {
	st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
	assert.NotNil(t, st)
}

func TestTCPListenAndServe(t *testing.T) {
	var addr = getFreeAddr("tcp4")

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp4"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServiceName("test name"),
		)

		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
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
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("tcp4"),
		transport.WithDialAddress(addr),
		transport.WithClientFramerBuilder(&framerBuilder{}))
	assert.NotNil(t, err)
}

func TestTCPTLSListenAndServe(t *testing.T) {
	addr := getFreeAddr("tcp")

	// Wait until server transport ready.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport()
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.pem"),
		)

		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
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
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("tcp"),
		transport.WithDialAddress(addr),
		transport.WithClientFramerBuilder(&framerBuilder{}),
		transport.WithDialTLS("../testdata/client.crt", "../testdata/client.key", "../testdata/ca.pem", "localhost"))
	assert.Nil(t, err)

	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("tcp"),
		transport.WithDialAddress(addr),
		transport.WithClientFramerBuilder(&framerBuilder{}),
		transport.WithDialTLS("../testdata/client.crt", "../testdata/client.key", "none", ""))
	assert.Nil(t, err)
}

func TestHandleError(t *testing.T) {
	var addr = getFreeAddr("udp4")

	// Wait until server transport is ready.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp4"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
		)

		if err != nil {
			t.Logf("test fail:%v", err)
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
		t.Fatalf("test fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()
	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("udp4"),
		transport.WithDialAddress(addr),
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
	lis, err := net.Listen("tcp", getFreeAddr("tcp"))
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
	var addr = getFreeAddr("tcp4")

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp4"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
			transport.WithWritev(true),
		)

		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
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
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("tcp4"),
		transport.WithDialAddress(addr),
		transport.WithClientFramerBuilder(&framerBuilder{}))
	assert.NotNil(t, err)
}

// TestTCPListenAndServerRoutinePool tests serving with goroutine pool.
func TestTCPListenAndServerRoutinePool(t *testing.T) {
	var addr = getFreeAddr("tcp4")

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp4"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
			transport.WithMaxRoutines(100),
		)

		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
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
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	reqData := append(lenData, data...)

	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("tcp4"),
		transport.WithDialAddress(addr),
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
	port, err := getFreePort("tcp")
	if err != nil {
		return fmt.Errorf("get freeport error: %v", err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	var prepareErr error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithReusePort(reuseport))
		err := st.ListenAndServe(ctx,
			transport.WithListenNetwork("tcp"),
			transport.WithListenAddress(fmt.Sprintf(":%d", port)),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
		)
		if err != nil {
			prepareErr = err
		}
	}()
	wg.Wait()

	if prepareErr != nil {
		cancel()
		return fmt.Errorf("prepare listener error: %v", prepareErr)
	}

	// First time dial, should work.
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		cancel()
		return fmt.Errorf("tcp dial error: %v", err)
	}
	conn.Close()

	// Notify and wait server close.
	cancel()
	time.Sleep(5 * time.Millisecond)

	// Second time dial, must fail.
	_, err = net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 10*time.Millisecond)
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
	port, err := getFreePort("tcp")
	if err != nil {
		t.Fatalf("get freeport error: %v", err)
	}
	err = transport.SaveListener(NewPacketConn{})
	assert.NotNil(t, err)

	newListener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	err = transport.SaveListener(newListener)
	assert.Nil(t, err)
	savedListenerPort = port
}

func TestTCPSeverErr(t *testing.T) {
	st := transport.NewServerTransport()
	err := st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("tcp"),
		transport.WithListenAddress(getFreeAddr("tcp")),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{}))
	assert.Nil(t, err)
}

func TestUDPServerErr(t *testing.T) {
	st := transport.NewServerTransport()

	err := st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(getFreeAddr("udp")),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{}))
	assert.Nil(t, err)
}

type fakeListen struct {
}

func (c *fakeListen) Accept() (net.Conn, error) {
	return nil, &netError{errors.New("网络失败")}
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
			t.Logf("ListenAndServe fail:%v", err)
		}
	}()
}

func TestUDPServerConErr(t *testing.T) {
	fb := transport.GetFramerBuilder("trpc")
	st := transport.NewServerTransport()
	err := st.ListenAndServe(context.Background(),
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(getFreeAddr("udp")),
		transport.WithServerFramerBuilder(fb))
	if err != nil {
		t.Fatalf("ListenAndServe fail:%v", err)
	}
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

func TestGetFreePort(t *testing.T) {
	for i := 0; i < 10; i++ {
		p, err := getFreePort("tcp")
		assert.Nil(t, err)
		assert.NotEqual(t, p, -1)
		t.Logf("get freeport network:%s, port:%d", "tcp", p)
	}

	for i := 0; i < 10; i++ {
		p, err := getFreePort("udp")
		assert.Nil(t, err)
		assert.NotEqual(t, p, -1)
		t.Logf("get freeport network:%s, port:%d", "udp", p)
	}

	p1, err := getFreePort("tcp")
	assert.Nil(t, err)

	p2, err := getFreePort("tcp")
	assert.Nil(t, err)
	assert.NotEqual(t, p1, p2, "allocated 2 conflict ports")
}

func getFreeAddr(network string) string {
	p, err := getFreePort(network)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(":%d", p)
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
	var addr = getFreeAddr("tcp4")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp4"),
			transport.WithListenAddress(addr),
			transport.WithHandler(&errorHandler{}),
			transport.WithServerFramerBuilder(&framerBuilder{}),
			transport.WithServerAsync(true),
		)
		assert.Nil(t, err)
	}()
	wg.Wait()

	// First time dial, should work.
	conn, err := net.Dial("tcp", addr)
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

// TestTCPListenAndServeWithSafeFramer tests that we support safe framer without copying packages.
func TestUDPListenAndServeWithSafeFramer(t *testing.T) {
	var addr = getFreeAddr("udp")

	// Wait until server transport is ready.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := transport.ListenAndServe(
			transport.WithListenNetwork("udp"),
			transport.WithListenAddress(addr),
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
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	rspData, err := transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("udp"),
		transport.WithDialAddress(addr),
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
	var addr = getFreeAddr("tcp4")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		st := transport.NewServerTransport(transport.WithKeepAlivePeriod(time.Minute))
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp4"),
			transport.WithListenAddress(addr),
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
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, f := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer f()

	rspData, err := transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("tcp4"),
		transport.WithDialAddress(addr),
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

func TestUDPServeClose(t *testing.T) {
	ts := transport.NewServerTransport()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ts.ListenAndServe(
		ctx,
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(getFreeAddr("udp")),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(true),
	)
	assert.Nil(t, err)
	time.Sleep(100 * time.Millisecond)
}

type MockUDPError struct{}

func (e MockUDPError) Error() string   { return "mock udp error" }
func (e MockUDPError) Timeout() bool   { return false }
func (e MockUDPError) Temporary() bool { return true }

func TestUDPReadError(t *testing.T) {
	addr := getFreeAddr("udp")

	err := transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(addr),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(false),
	)
	assert.Nil(t, err)
	time.Sleep(60 * time.Millisecond)
}

func TestUDPWriteError(t *testing.T) {
	addr := getFreeAddr("udp")

	err := transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(addr),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(false),
	)
	assert.Nil(t, err)
	time.Sleep(20 * time.Millisecond)

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("udp"),
		transport.WithDialAddress(addr),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)
}

func TestPoolInvokeFail(t *testing.T) {

	addr := getFreeAddr("udp")

	err := transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(addr),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(true),
	)
	assert.Nil(t, err)
	time.Sleep(20 * time.Millisecond)

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("udp"),
		transport.WithDialAddress(addr),
		transport.WithClientFramerBuilder(&framerBuilder{safe: true}))
	assert.Nil(t, err)
}

func TestCreatePoolFail(t *testing.T) {
	addr := getFreeAddr("udp")

	err := transport.ListenAndServe(
		transport.WithListenNetwork("udp"),
		transport.WithListenAddress(addr),
		transport.WithHandler(&echoHandler{}),
		transport.WithServerFramerBuilder(&framerBuilder{safe: true}),
		transport.WithServerAsync(true),
	)
	assert.Nil(t, err)

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))
	reqData := append(lenData, data...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = transport.RoundTrip(ctx, reqData, transport.WithDialNetwork("udp"),
		transport.WithDialAddress(addr),
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
