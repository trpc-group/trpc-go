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

package tnet_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	reuseport "github.com/kavu/go_reuseport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/keeporder"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

var (
	port                     uint64 = 9000
	helloWorld                      = []byte("helloworld")
	defaultUserDefineHandler        = &userDefineHandler{handleFunc: defaultServerHandle}
)

// Test basic ListenAndServe functionality.
func TestServerTCP_ListenAndServe(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		nil,
		func(addr string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(addr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test asynchronous server functionality.
func TestServerTCP_Asyn(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithServerAsync(true)},
		func(addr string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(addr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test customized framer with buffer reuse.
func TestServerTCP_CustomizedFramerCopyFrame(t *testing.T) {
	startServerTest(
		t,
		newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
			return req, nil
		}),
		[]transport.ListenServeOption{
			transport.WithServerFramerBuilder(&reuseBufferFramerBuilder{}),
			transport.WithServerAsync(true),
		},
		func(addr string) {
			req := helloWorld
			ctx, _ := codec.EnsureMessage(context.Background())
			reqbytes, err := (&emptyClientCodec{}).Encode(
				codec.Message(ctx),
				req,
			)
			assert.Nil(t, err)

			cliOpts := []transport.RoundTripOption{
				transport.WithDialAddress(addr),
				transport.WithDialNetwork("tcp"),
				transport.WithClientFramerBuilder(&reuseBufferFramerBuilder{}),
				transport.WithDialTimeout(5 * time.Second),
			}
			clientTrans := transport.NewClientTransport()
			rspbytes, err := clientTrans.RoundTrip(
				ctx,
				reqbytes,
				cliOpts...,
			)
			assert.Nil(t, err)

			rsp, err := (&emptyClientCodec{}).Decode(
				codec.Message(ctx),
				rspbytes,
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test user-defined listener functionality.
func TestServerTCP_UserDefineListener(t *testing.T) {
	serverAddr := getAddr()
	ln, err := tnet.Listen("tcp", serverAddr)
	assert.Nil(t, err)
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithListener(ln)},
		func(_ string) {
			rsp, err := gonetRequest(context.Background(), transport.WithDialAddress(serverAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test error cases.
func TestServerTCP_ErrorCases(t *testing.T) {
	s := tnettrans.NewServerTransport()

	// Without framerBuilder
	serOpts := getListenServeOption(
		transport.WithServerFramerBuilder(nil),
	)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.NotNil(t, err)

	// Unsupported network type
	serOpts = getListenServeOption(
		transport.WithListenNetwork("ip"),
	)
	err = s.ListenAndServe(context.Background(), serOpts...)
	assert.NotNil(t, err)
}

// Test handler error.
func TestServerTCP_HandleErr(t *testing.T) {
	startServerTest(
		t,
		newUserDefineHandler(errServerHandle),
		nil,
		func(addr string) {
			_, err := gonetRequest(context.Background(), transport.WithDialAddress(addr))
			assert.NotNil(t, err)
		},
	)
}

// Test idle timeout.
func TestServerTCP_IdleTimeout(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithServerIdleTimeout(time.Second)},
		func(addr string) {
			cliconn, err := tnet.DialTCP("tcp", addr, 0)
			assert.Nil(t, err)
			_, err = cliconn.Write([]byte("0"))
			assert.Nil(t, err)

			// sleep to make sure ListenAndServe run into onRequest()
			time.Sleep(2 * time.Second)
			_, err = cliconn.Write([]byte("0"))
			assert.NotNil(t, err)
		},
	)
}

// Test write failure.
func TestServerTCP_WriteFail(t *testing.T) {
	ch := make(chan struct{}, 1)
	var isHandled bool
	startServerTest(
		t,
		newUserDefineHandler(func(ctx context.Context, req []byte) ([]byte, error) {
			isHandled = true
			<-ch
			return nil, nil
		}),
		[]transport.ListenServeOption{transport.WithServerAsync(true)},
		func(addr string) {
			ctx, _ := codec.EnsureMessage(context.Background())
			req, err := trpc.DefaultClientCodec.Encode(codec.Message(ctx), helloWorld)
			assert.Nil(t, err)

			cliconn, err := tnet.DialTCP("tcp", addr, 0)
			assert.Nil(t, err)
			_, err = cliconn.Write(req)
			assert.Nil(t, err)

			// sleep to make sure server received data
			time.Sleep(50 * time.Millisecond)
			cliconn.Close()
			// notify server write back data, but server will fail, because connection is closed
			ch <- struct{}{}
			_, err = cliconn.ReadN(1)
			assert.NotNil(t, err)
			// make sure server run into handle
			assert.True(t, isHandled)
		},
	)
}

func testServerTCPAndUDP_PassedListener(t *testing.T) {
	tcpServerAddr := getAddr()
	tcpListener, err := net.Listen("tcp", tcpServerAddr)
	assert.Nil(t, err)
	transport.SaveListener(tcpListener)

	udpServerAddr := getAddr()
	udpListener, err := net.ListenPacket("udp", udpServerAddr)
	assert.Nil(t, err)
	transport.SaveListener(udpListener)

	fds := transport.GetListenersFds()
	var tcpFD int
	var udpFD int
	for _, f := range fds {
		if f.Address == tcpServerAddr {
			tcpFD = int(f.Fd)
		}
		if f.Address == udpServerAddr {
			udpFD = int(f.Fd)
		}
	}

	minFD, maxFD := tcpFD, udpFD
	if minFD > maxFD {
		minFD, maxFD = maxFD, minFD
	}
	os.Setenv(transport.EnvGraceRestart, "1")
	os.Setenv(transport.EnvGraceFirstFd, fmt.Sprint(minFD))
	os.Setenv(transport.EnvGraceRestartFdNum, fmt.Sprint(maxFD-minFD+1))

	defer func() {
		os.Setenv(transport.EnvGraceRestart, "0")
		os.Setenv(transport.EnvGraceFirstFd, "0")
		os.Setenv(transport.EnvGraceRestartFdNum, "0")
	}()

	tcpCtx, tcpCancel := context.WithTimeout(context.Background(), time.Second)
	defer tcpCancel()
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithListenAddress(tcpServerAddr)},
		func(_ string) {
			rsp, err := gonetRequest(
				tcpCtx,
				transport.WithDialAddress(tcpServerAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)

	udpCtx, udpCancel := context.WithTimeout(context.Background(), time.Second)
	defer udpCancel()
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{
			transport.WithListenNetwork("udp"),
			transport.WithListenAddress(udpServerAddr)},
		func(_ string) {
			rsp, err := gonetRequest(
				udpCtx,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(udpServerAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

func TestServerTCPAndUDP_PassedListenerFallback(t *testing.T) {
	tcpListener, err := reuseport.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	tcpServerAddr := tcpListener.Addr().String()

	udpListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	assert.Nil(t, err)
	udpServerAddr := udpListener.LocalAddr().String()

	// Store the original environment values for graceful restart.
	oldGraceRestart := os.Getenv(transport.EnvGraceRestart)
	oldGraceFirstFd := os.Getenv(transport.EnvGraceFirstFd)
	oldGraceRestartFdNum := os.Getenv(transport.EnvGraceRestartFdNum)

	// Set environment variables for testing graceful restart.
	os.Setenv(transport.EnvGraceRestart, "1")
	os.Setenv(transport.EnvGraceFirstFd, "0")
	os.Setenv(transport.EnvGraceRestartFdNum, "0")

	// Close the test listeners.
	tcpListener.Close()
	udpListener.Close()

	// Restore original environment values after test.
	defer func() {
		os.Setenv(transport.EnvGraceRestart, oldGraceRestart)
		os.Setenv(transport.EnvGraceFirstFd, oldGraceFirstFd)
		os.Setenv(transport.EnvGraceRestartFdNum, oldGraceRestartFdNum)
	}()

	tcpCtx, tcpCancel := context.WithTimeout(context.Background(), time.Second)
	defer tcpCancel()
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithListenAddress(tcpServerAddr)},
		func(_ string) {
			rsp, err := gonetRequest(
				tcpCtx,
				transport.WithDialAddress(tcpServerAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)

	udpCtx, udpCancel := context.WithTimeout(context.Background(), time.Second)
	defer udpCancel()
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{
			transport.WithListenNetwork("udp"),
			transport.WithListenAddress(udpServerAddr)},
		func(_ string) {
			rsp, err := gonetRequest(
				udpCtx,
				transport.WithDialNetwork("udp"),
				transport.WithDialAddress(udpServerAddr))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test client wrong request.
func TestServerTCP_ClientWrongReq(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		nil,
		func(addr string) {
			cliconn, err := tnet.DialTCP("tcp", addr, 0)
			assert.Nil(t, err)
			_, err = cliconn.Write([]byte("1234567890123456"))
			assert.Nil(t, err)

			// sleep to make sure ListenAndServe run into onRequest()
			time.Sleep(50 * time.Millisecond)
			err = cliconn.Close()
			assert.Nil(t, err)
		},
	)
}

// Test send and close.
func TestServerTCP_SendAndClose(t *testing.T) {
	addr := getAddr()
	s := tnettrans.NewServerTransport()
	serOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithServerAsync(true),
	)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.Nil(t, err)

	cliconn, err := tnet.DialTCP("tcp", addr, 0)
	assert.Nil(t, err)
	cliAddr := cliconn.LocalAddr()

	time.Sleep(50 * time.Millisecond)
	streamTransport, ok := s.(transport.ServerStreamTransport)
	assert.True(t, ok)
	ctx, msg := codec.EnsureMessage(context.Background())
	msg.WithRemoteAddr(cliAddr)
	svrAddr, err := net.ResolveTCPAddr("tcp", addr)
	assert.Nil(t, err)
	msg.WithLocalAddr(svrAddr)
	err = streamTransport.Send(ctx, helloWorld)
	assert.Nil(t, err)

	b := make([]byte, len(helloWorld))
	cliconn.Read(b)
	assert.Equal(t, b, helloWorld)

	streamTransport.Close(ctx)
	err = streamTransport.Send(ctx, helloWorld)
	assert.NotNil(t, err)
}

// Test TLS functionality.
func TestServerTCP_TLS(t *testing.T) {
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{transport.WithServeTLS("../../testdata/server.crt", "../../testdata/server.key", "../../testdata/ca.pem")},
		func(addr string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialAddress(addr),
				transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "../../testdata/ca.pem", "localhost"),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)

			rsp, err = gonetRequest(
				context.Background(),
				transport.WithDialAddress(addr),
				transport.WithDialTLS("../../testdata/client.crt", "../../testdata/client.key", "none", ""),
			)
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test Unix socket functionality.
func TestUnix(t *testing.T) {
	// Unix socket is not supported, but it will switch to gonet default transport to serve.
	myAddr := "/tmp/server.sock"
	os.Remove(myAddr)
	startServerTest(
		t,
		defaultUserDefineHandler,
		[]transport.ListenServeOption{
			transport.WithListenNetwork("unix"),
			transport.WithListenAddress(myAddr),
		},
		func(_ string) {
			rsp, err := gonetRequest(
				context.Background(),
				transport.WithDialAddress(myAddr),
				transport.WithDialNetwork("unix"))
			assert.Nil(t, err)
			assert.Equal(t, helloWorld, rsp)
		},
	)
}

// Test keep order pre-decode functionality.
func TestServerTCP_KeepOrderPreDecode(t *testing.T) {
	metaDataKey := "meta_key"
	startServerTest(
		t,
		&tnetPreDecodeHandler{tnetKeepOrderHandler: tnetKeepOrderHandler{values: make(map[string][]string)}},
		[]transport.ListenServeOption{
			transport.WithServerAsync(true),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
				msg := codec.Message(ctx)
				meta := msg.ServerMetaData()
				if meta == nil {
					return "", false
				}
				return string(meta[metaDataKey]), true
			}),
		},
		func(addr string) {
			sendKeepOrderPreDecodeReq(t, addr, metaDataKey, assertRspWithKeepOrder)
		},
	)
}

// Test keep order pre-unmarshal functionality.
func TestServerTCP_KeepOrderPreUnmarshal(t *testing.T) {
	startServerTest(
		t,
		&tnetPreUnmarshalHandler{tnetKeepOrderHandler: tnetKeepOrderHandler{values: make(map[string][]string)}},
		[]transport.ListenServeOption{
			transport.WithServerAsync(false),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
				request, ok := req.([]byte)
				if !ok {
					return "", false
				}
				ss := strings.Split(string(request), " ")
				if len(ss) != 2 {
					return "", false
				}
				return ss[0], true
			}),
		},
		func(addr string) {
			sendKeepOrderPreUnmarshalReq(t, addr, assertRspWithKeepOrder)
		},
	)
}

// Test keep order pre-decode failure.
func TestServerTCP_KeepOrderPreDecodeFail(t *testing.T) {
	metaDataKey := "meta_key"

	// test PreDecode fail and fallback to non-keep-order scenario
	startServerTest(
		t,
		&tnetPreDecodeHandlerWithErr{tnetKeepOrderHandler: tnetKeepOrderHandler{values: make(map[string][]string)}},
		[]transport.ListenServeOption{
			transport.WithServerAsync(true),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
				msg := codec.Message(ctx)
				meta := msg.ServerMetaData()
				if meta == nil {
					return "", false
				}
				return string(meta[metaDataKey]), true
			}),
		},
		func(addr string) {
			sendKeepOrderPreDecodeReq(t, addr, metaDataKey, assertRspWithKeepOrderFail)
		},
	)

	// test extract key fail and fallback to non-keep-order scenario
	startServerTest(
		t,
		&tnetPreDecodeHandler{tnetKeepOrderHandler: tnetKeepOrderHandler{values: make(map[string][]string)}},
		[]transport.ListenServeOption{
			transport.WithServerAsync(true),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
				return "", false
			}),
		},
		func(addr string) {
			sendKeepOrderPreDecodeReq(t, addr, metaDataKey, assertRspWithKeepOrderFail)
		},
	)
}

// Test keep order pre-unmarshal failure.
func TestServerTCP_KeepOrderPreUnmarshalFail(t *testing.T) {
	// test PreUnmarshal fail and fallback to non-keep-order scenario
	startServerTest(
		t,
		&tnetPreUnmarshalHandlerWithErr{tnetKeepOrderHandler: tnetKeepOrderHandler{values: make(map[string][]string)}},
		[]transport.ListenServeOption{
			transport.WithServerAsync(false),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
				request, ok := req.([]byte)
				if !ok {
					return "", false
				}
				ss := strings.Split(string(request), " ")
				if len(ss) != 2 {
					return "", false
				}
				return ss[0], true
			}),
		},
		func(addr string) {
			sendKeepOrderPreUnmarshalReq(t, addr, assertRspWithKeepOrderFail)
		},
	)

	// test extract key fail and fallback to non-keep-order scenario
	startServerTest(
		t,
		&tnetPreUnmarshalHandler{tnetKeepOrderHandler: tnetKeepOrderHandler{values: make(map[string][]string)}},
		[]transport.ListenServeOption{
			transport.WithServerAsync(false),
			transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
			transport.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
				return "", false
			}),
		},
		func(addr string) {
			sendKeepOrderPreUnmarshalReq(t, addr, assertRspWithKeepOrderFail)
		},
	)
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
		for i := 0; i < count; i++ {
			var (
				rsp []byte
				err error
			)
			ech := make(chan error, 1)
			ctx := keeporder.NewContextWithClientInfo(trpc.BackgroundContext(), &keeporder.ClientInfo{SendError: ech})
			eg.Go(func(key string, i int) func() error {
				return func() error {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					msg := codec.Message(ctx)
					msg.WithRequestID(atomic.AddUint32(&requestID, 1))
					msg.WithClientMetaData(codec.MetaData{metaDataKey: []byte(key)})
					data := []byte(key + " " + strconv.Itoa(i))
					var reqData []byte
					reqData, err = trpc.DefaultClientCodec.Encode(msg, data)
					if err != nil {
						return fmt.Errorf("client codec encode err: %+v", err)
					}
					rtOpts := getRoundTripOption(
						transport.WithDialNetwork("tcp"),
						transport.WithDialAddress(addr),
						transport.WithMsg(msg),
						transport.WithMultiplexedPool(p),
						transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder),
					)
					rsp, err = transport.RoundTrip(ctx, reqData, rtOpts...)
					select {
					case ech <- err: // If the error is generated before transport write, this case will be executed.
					default:
					}
					if err != nil {
						return err
					}
					mu.Lock()
					s := string(rsp)
					if len(rsps[key]) < len(s) {
						rsps[key] = s
					}
					mu.Unlock()
					return err
				}
			}(key, i))
			if err := <-ech; err != nil {
				t.Errorf("request %q failed: %v", key, err)
			}
		}
	}
	require.NoError(t, eg.Wait())
	rsp_checker(t, count, keys, rsps)
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
		for i := 0; i < count; i++ {
			var (
				rsp []byte
				err error
			)
			ech := make(chan error, 1)
			ctx := keeporder.NewContextWithClientInfo(trpc.BackgroundContext(), &keeporder.ClientInfo{
				SendError: ech,
			})
			eg.Go(func(key string, i int) func() error {
				return func() error {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					msg := codec.Message(ctx)
					msg.WithRequestID(atomic.AddUint32(&requestID, 1))
					data := []byte(key + " " + strconv.Itoa(i))
					var reqData []byte
					reqData, err = trpc.DefaultClientCodec.Encode(msg, data)
					if err != nil {
						return fmt.Errorf("client codec encode err: %+v", err)
					}
					rtOpts := getRoundTripOption(
						transport.WithDialNetwork("tcp"),
						transport.WithDialAddress(addr),
						transport.WithMsg(msg),
						transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder),
						transport.WithMultiplexedPool(p),
					)
					rsp, err = transport.RoundTrip(ctx, reqData, rtOpts...)
					select {
					case ech <- err: // If the error is generated before transport write, this case will be executed.
					default:
					}
					if err != nil {
						return err
					}
					mu.Lock()
					s := string(rsp)
					if len(rsps[key]) < len(s) {
						rsps[key] = s
					}
					mu.Unlock()
					return err
				}
			}(key, i))
			if err := <-ech; err != nil {
				t.Errorf("request %q failed: %v", key, err)
			}
		}
	}
	require.NoError(t, eg.Wait())
	rsp_checker(t, count, keys, rsps)
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

func getListenServeOption(opts ...transport.ListenServeOption) []transport.ListenServeOption {
	lsopts := []transport.ListenServeOption{
		transport.WithServerFramerBuilder(trpc.DefaultFramerBuilder),
		transport.WithListenNetwork("tcp"),
		transport.WithHandler(defaultUserDefineHandler),
		transport.WithServerIdleTimeout(5 * time.Second),
	}
	lsopts = append(lsopts, opts...)
	return lsopts
}

func defaultServerHandle(ctx context.Context, req []byte) (rsp []byte, err error) {
	msg := codec.Message(ctx)
	fmt.Printf("defaultServerHandle req: %q\n", req)
	reqdata, err := trpc.DefaultServerCodec.Decode(msg, req)
	if err != nil {
		return nil, err
	}
	rspdata := make([]byte, len(reqdata))
	copy(rspdata, reqdata)
	rsp, err = trpc.DefaultServerCodec.Encode(msg, rspdata)
	fmt.Printf("defaultServerHandle rsp: %q\n", rsp)
	return rsp, err
}

func errServerHandle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return nil, errors.New("mock error")
}

type userDefineHandler struct {
	handleFunc func(context.Context, []byte) ([]byte, error)
}

func newUserDefineHandler(f func(context.Context, []byte) ([]byte, error)) *userDefineHandler {
	return &userDefineHandler{handleFunc: f}
}

func (uh *userDefineHandler) Handle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return uh.handleFunc(ctx, req)
}

type tnetKeepOrderHandler struct {
	mu     sync.Mutex
	values map[string][]string
}

func (t *tnetKeepOrderHandler) Handle(ctx context.Context, req []byte) ([]byte, error) {
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
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values[key] = append(t.values[key], val)
	body := []byte(strings.Join(t.values[key], " "))
	rsp, err := trpc.DefaultServerCodec.Encode(msg, body)
	return rsp, err
}

type tnetPreDecodeHandler struct {
	tnetKeepOrderHandler
}

func (t *tnetPreDecodeHandler) PreDecode(ctx context.Context, reqBuf []byte) (reqBodyBuf []byte, err error) {
	msg := codec.Message(ctx)
	return trpc.DefaultServerCodec.Decode(msg, reqBuf)
}

type tnetPreUnmarshalHandler struct {
	tnetKeepOrderHandler
}

func (t *tnetPreUnmarshalHandler) PreUnmarshal(ctx context.Context, reqBuf []byte) (reqBody interface{}, err error) {
	msg := codec.Message(ctx)
	return trpc.DefaultServerCodec.Decode(msg, reqBuf)
}

type tnetPreDecodeHandlerWithErr struct {
	tnetKeepOrderHandler
}

func (t *tnetPreDecodeHandlerWithErr) PreDecode(ctx context.Context, reqBuf []byte) (reqBodyBuf []byte, err error) {
	return nil, errors.New("mock error")
}

type tnetPreUnmarshalHandlerWithErr struct {
	tnetKeepOrderHandler
}

func (t *tnetPreUnmarshalHandlerWithErr) PreUnmarshal(ctx context.Context, reqBuf []byte) (reqBody interface{}, err error) {
	return nil, errors.New("mock error")
}

func startServerTest(
	t *testing.T,
	handler transport.Handler,
	svrCustomOpts []transport.ListenServeOption,
	clientHandle func(addr string),
) {
	addr := getAddr()
	s := tnettrans.NewServerTransport(
		tnettrans.WithKeepAlivePeriod(15*time.Second),
		tnettrans.WithReusePort(true),
	)
	require.NotNil(t, handler)
	serOpts := getListenServeOption(
		transport.WithListenAddress(addr),
		transport.WithHandler(handler),
	)
	serOpts = append(serOpts, svrCustomOpts...)
	err := s.ListenAndServe(context.Background(), serOpts...)
	assert.Nil(t, err)

	clientHandle(addr)
}

func gonetRequest(ctx context.Context, opts ...transport.RoundTripOption) ([]byte, error) {
	req := helloWorld
	ctx, _ = codec.EnsureMessage(ctx)
	reqbytes, err := trpc.DefaultClientCodec.Encode(
		codec.Message(ctx),
		req,
	)
	if err != nil {
		return nil, err
	}

	cliOpts := getRoundTripOption(opts...)
	clientTrans := transport.NewClientTransport()
	rspbytes, err := clientTrans.RoundTrip(
		ctx,
		reqbytes,
		cliOpts...,
	)
	if err != nil {
		return nil, err
	}
	rsp, err := trpc.DefaultClientCodec.Decode(
		codec.Message(ctx),
		rspbytes,
	)
	return rsp, err
}

func getRoundTripOption(opts ...transport.RoundTripOption) []transport.RoundTripOption {
	rtopts := []transport.RoundTripOption{
		transport.WithDialNetwork("tcp"),
		transport.WithClientFramerBuilder(trpc.DefaultFramerBuilder),
		transport.WithDialTimeout(5 * time.Second),
	}
	rtopts = append(rtopts, opts...)
	return rtopts
}

func getAddr() string {
	atomic.AddUint64(&port, 1)
	return "127.0.0.1:" + fmt.Sprint(port)
}

type reuseBufferFramerBuilder struct{}

func (*reuseBufferFramerBuilder) New(r io.Reader) codec.Framer {
	return &reuseBufferFramer{r: r, reuseBuffer: make([]byte, len(helloWorld))}
}

type reuseBufferFramer struct {
	r           io.Reader
	reuseBuffer []byte
}

func (f *reuseBufferFramer) ReadFrame() ([]byte, error) {
	_, err := io.ReadFull(f.r, f.reuseBuffer)
	if err != nil {
		return nil, fmt.Errorf("io.ReadFull err: %w", err)
	}
	return f.reuseBuffer, nil
}

type emptyServerCodec struct{}

func (s *emptyServerCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	return reqBuf, nil
}

func (s *emptyServerCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	return rspBody, nil
}

type emptyClientCodec struct{}

func (s *emptyClientCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	return reqBuf, nil
}

func (s *emptyClientCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	return rspBody, nil
}
