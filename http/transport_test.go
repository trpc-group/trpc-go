package http_test

// https certificate file generation method:
// 1. ca certificate:
// openssl genrsa -out ca.key 2048
// openssl req -x509 -new -nodes -key ca.key -subj "/CN=*" -days 5000 -out ca.pem
// 2. server certificate:
// openssl genrsa -out server.key 2048
// openssl req -new -key server.key -subj "/CN=*" -out server.csr
// openssl x509 -req -in server.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out server.crt -days 5000 <(printf "subjectAltName=DNS:localhost")
// 3. client certificate:
// openssl genrsa -out client.key 2048
// openssl req -new -key client.key -subj "/CN=*" -out client.csr
// openssl x509 -req -in client.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out client.crt -days 5000 <(printf "subjectAltName=DNS:localhost")
// 4. show certificate content:
// openssl x509 -text -in server.crt -noout

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
)

func newNoopStdHTTPServer() *http.Server { return &http.Server{} }

func TestStartServer(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	option := transport.WithListener(ln)
	handler := transport.WithHandler(transport.Handler(&h{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")
	require.NotNil(t, tp.ListenAndServe(ctx, transport.WithListenAddress("127.0.0.1:8888"), handler, transport.WithListenNetwork("tcp1")))
	tls := transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "ca1")
	require.NotNil(t, tp.ListenAndServe(ctx, option, handler, tls))
}

func TestH2C(t *testing.T) {
	ctx := context.Background()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	handler := transport.WithHandler(transport.Handler(&h{}))
	tp := thttp.NewServerTransport(newNoopStdHTTPServer, thttp.WithReusePort(), thttp.WithEnableH2C())
	require.Nil(t, tp.ListenAndServe(ctx, transport.WithListener(ln), handler))
}

func TestDisableReusePort(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln1.Close()
	option := transport.WithListener(ln1)
	handler := transport.WithHandler(transport.Handler(&h{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")

	option = transport.WithListenAddress(ln1.Addr().String())
	require.NotNil(t, tp.ListenAndServe(ctx, option, handler, transport.WithListenNetwork("tcp1")))

	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln2.Close()
	option = transport.WithListener(ln2)
	tls := transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "")
	require.Nil(t, tp.ListenAndServe(ctx, option, handler, tls))

	ln3, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln3.Close()
	option = transport.WithListener(ln3)
	tls = transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "root")
	require.Nil(t, tp.ListenAndServe(ctx, option, handler, tls))

	ln4, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln4.Close()
	option = transport.WithListener(ln4)
	tls = transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.key")
	require.NotNil(t, tp.ListenAndServe(ctx, option, handler, tls))
}

func TestStartServerWithNoHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	option := transport.WithListener(ln)
	require.NotNil(t, tp.ListenAndServe(ctx, option), "http server transport handler empty")
}

func TestErrHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	option := transport.WithListener(ln)
	h := transport.WithHandler(transport.Handler(&errHandler{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, h))

	ct := thttp.NewClientTransport(true)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestErrHeaderHandler(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	err = tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&errHeaderHandler{})),
		transport.WithListener(ln),
	)
	require.Nil(t, err)

	ct := thttp.NewClientTransport(true)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestListenAndServeFailedDueToBadCertificationFile(t *testing.T) {
	ctx := context.Background()
	oldLogger := log.DefaultLogger
	defer func() {
		log.DefaultLogger = oldLogger
	}()
	errorCh := make(chan error)
	log.DefaultLogger = &testLog{Logger: oldLogger, errorCh: errorCh}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	const badCertFile = "bad-file.cert"
	require.Nil(
		t,
		thttp.NewServerTransport(newNoopStdHTTPServer).ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS(badCertFile, "../testdata/server.key", ""),
		),
		"failed to new client transport",
	)

	select {
	case <-time.After(time.Second):
		t.Fatal("listen on a bad cert should log an error")
	case err := <-errorCh:
		require.Contains(t, err.Error(), badCertFile)
	}
}

func TestStartTLSServerAndNoCheckServer(t *testing.T) {
	ctx := context.Background()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	// Only enables https server and do not verify client certificate.
	require.Nil(
		t,
		thttp.NewServerTransport(newNoopStdHTTPServer).ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", ""),
		),
		"Failed to new client transport",
	)

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(
		ctx,
		[]byte("{\"username\":\"xyz\","+"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Fully trust the https server and do not verify server certificate,
		// can only be used in test env.
		transport.WithDialTLS("", "", "none", ""),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestServerWithListenerOption(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	service := server.New(
		server.WithServiceName("trpc.http.server.ListenerTest"),
		server.WithNetwork("tcp"),
		server.WithProtocol("http"),
		server.WithListener(ln),
	)
	thttp.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Printf("Protocol: %s\n", r.Proto)
		w.Write([]byte(r.Proto))
		return nil
	})
	thttp.RegisterDefaultService(service)
	s := &server.Server{}
	s.AddService("trpc.http.server.ListenerTest", service)
	go func() {
		require.Nil(t, s.Serve())
	}()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%v/index", ln.Addr()))
	require.Nil(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/1.1"), body)

	const invalidAddr = "localhost:910439"
	resp, err = http.Get(fmt.Sprintf("http://%s/index", invalidAddr))
	require.NotNil(t, err)
	require.Nil(t, resp)
}

func TestStartDisableKeepAlivesServer(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.http.server.ListenerTest"),
		server.WithNetwork("tcp"),
		server.WithProtocol("http"),
		server.WithTransport(thttp.NewServerTransport(newNoopStdHTTPServer)),
		server.WithDisableKeepAlives(true),
	)
	thttp.HandleFunc("/disable-keepalives", func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set("Connection", "keep-alive")
		return nil
	})
	thttp.RegisterDefaultService(service)
	s.AddService("trpc.http.server.ListenerTest", service)
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()
	defer func() {
		_ = s.Close(nil)
	}()

	time.Sleep(100 * time.Millisecond)

	dailCount := 0
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dailCount++
				conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
				return conn, err
			},
		},
	}
	num := 3
	url := fmt.Sprintf("http://%s/disable-keepalives", ln.Addr())
	for i := 0; i < num; i++ {
		resp, err := client.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		_, err = io.Copy(io.Discard, resp.Body)
		require.Nil(t, err)
	}
	require.Equal(t, num, dailCount)
}

func TestStartH2cServer(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.h2c.server.Greeter"),
		server.WithNetwork("tcp"),
		server.WithProtocol("http2"),
		server.WithTransport(thttp.NewServerTransport(newNoopStdHTTPServer, thttp.WithEnableH2C())),
	)
	thttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Printf("Protocol: %s\n", r.Proto)
		w.Write([]byte(r.Proto))
		return nil
	})
	thttp.HandleFunc("/main", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Printf("Protocol: %s\n", r.Proto)
		w.Write([]byte(r.Proto))
		return nil
	})
	thttp.RegisterDefaultService(service)
	s.AddService("trpc.h2c.server.Greeter", service)

	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// h2c client
	h2cClient := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	url := fmt.Sprintf("http://%s/", ln.Addr())
	resp, err := h2cClient.Get(url + "main")
	require.Nil(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/2.0"), body)

	// http1 client
	resp2, err := http.Get(url)
	require.Nil(t, err)
	defer resp2.Body.Close()
	body, err = io.ReadAll(resp2.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("HTTP/1.1"), body)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
}

func TestHttp2StartTLSServerAndNoCheckServer(t *testing.T) {
	ctx := context.Background()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	// Only enables https server and do not verify client certificate.
	require.Nil(
		t,
		thttp.NewServerTransport(newNoopStdHTTPServer).ListenAndServe(
			ctx,
			transport.WithListener(ln),
			transport.WithHandler(transport.Handler(&h{})),
			transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", ""),
		),
		"Failed to new client transport",
	)

	ct := thttp.NewClientTransport(true)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(
		ctx,
		[]byte("{\"username\":\"xyz\","+"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Fully trust the https server and do not verify server certificate,
		// can only be used in test env.
		transport.WithDialTLS("", "", "none", ""),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckServer(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	err = tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Only enables https server and do not verify client certificate.
		transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", ""),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Uses ca public key to verify server certificate.
		transport.WithDialTLS("", "", "../testdata/ca.pem", "localhost"),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckClientNoCert(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	err = tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Enables two-way authentication http server and need to verify client certificate.
		transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.pem"),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	_, err = ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// If the client's own certificate is not sent, will return TLS verification failed.
		transport.WithDialTLS("", "", "../testdata/ca.pem", "localhost"),
	)
	require.NotNil(t, err, "Failed to roundtrip")
}

func TestStartTLSServerAndCheckClient(t *testing.T) {
	ctx := context.Background()
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() { require.Nil(t, ln.Close()) }()
	// Enables two-way authentication http server and need to verify client certificate.
	err = tp.ListenAndServe(ctx,
		transport.WithHandler(transport.Handler(&h{})),
		// Only enables https server and do not verify client certificate.
		transport.WithServeTLS("../testdata/server.crt", "../testdata/server.key", "../testdata/ca.pem"),
		transport.WithListener(ln),
	)
	require.Nil(t, err, "Failed to new client transport")

	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()),
		// Need to send the client's own certificate to server.
		transport.WithDialTLS("../testdata/client.crt", "../testdata/client.key", "../testdata/ca.pem", "localhost"),
	)
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestNewClientTransport(t *testing.T) {
	ct := thttp.NewClientTransport(false)
	require.NotNil(t, ct, "Failed to new client transport")

	ct2 := thttp.NewClientTransport(true)
	require.NotNil(t, ct2, "Failed to new http2 client transport")
}

func TestClientRoundTrip(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	msg.WithClientReqHead(&thttp.ClientReqHeader{})
	msg.WithClientRspHead(&thttp.ClientRspHeader{})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, nil)
	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress(ln.Addr().String()))
	require.Nil(t, rsp, "roundtrip rsp not empty")
	require.Nil(t, err, "Failed to roundtrip")
}

func TestClientRoundTripWithNoHead(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")

	rsp, err := ct.RoundTrip(ctx, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		transport.WithDialAddress("127.0.0.1:18080"))
	require.Nil(t, rsp, "no head roundtrip rsp not empty")
	require.NotNil(t, err, "no head roundtrip err nil")

}

func TestClientWithSelectorNode(t *testing.T) {
	ctx := context.Background()
	type testCase struct {
		target   string
		address  string
		listener net.Listener
	}
	var tests []testCase
	for i := 0; i < 2; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.Nil(t, err)
		defer ln.Close()
		addr := ln.Addr().String()
		tests = append(tests, testCase{"ip://" + addr, addr, ln})
	}
	for _, tt := range tests {
		tp := thttp.NewServerTransport(newNoopStdHTTPServer)
		option := transport.WithListener(tt.listener)
		handler := transport.WithHandler(transport.Handler(&h{}))
		err := tp.ListenAndServe(ctx, option, handler)
		require.Nil(t, err, "Failed to new client transport")

		proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
			client.WithTarget(tt.target),
			client.WithSerializationType(codec.SerializationTypeNoop),
		)

		reqBody := &codec.Body{
			Data: []byte("{\"username\":\"xyz\"," +
				"\"password\":\"xyz\",\"from\":\"xyz\"}"),
		}
		rspBody := &codec.Body{}
		n := &registry.Node{}
		require.Nil(t,
			proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody, client.WithSelectorNode(n)),
			"Failed to post")
		require.Equal(t, tt.address, n.Address)
	}
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	old := codec.GetSerializer(codec.SerializationTypeJSON)
	defer func() { codec.RegisterSerializer(codec.SerializationTypeJSON, old) }()
	codec.RegisterSerializer(codec.SerializationTypeJSON, &codec.JSONPBSerialization{})
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	option := transport.WithListener(ln)
	handler := transport.WithHandler(transport.Handler(&h{}))
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")

	header := &thttp.ClientReqHeader{}
	header.AddHeader("ContentType", "application/json")

	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithReqHead(header),
		client.WithMetaData("k1", []byte("v1")),
	)
	reqBody := &codec.Body{
		Data: []byte("{\"username\":\"xyz\"," +
			"\"password\":\"xyz\",\"from\":\"xyz\"}"),
	}
	rspBody := &codec.Body{}

	require.Nil(t, proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to post")
	require.Nil(t, proxy.Put(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to put")
	require.Nil(t, proxy.Delete(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to delete")
	require.Nil(t, proxy.Get(ctx, "/trpc.test.helloworld.Greeter/SayHello", rspBody), "Failed to get")
	require.Nil(t, proxy.Patch(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody), "Failed to patch")

	// Test client with options.
	proxy = thttp.NewClientProxy("trpc.test.helloworld.Greeter")
	reqBody = &codec.Body{
		Data: []byte("{\"username\":\"xyz\"," +
			"\"password\":\"xyz\",\"from\":\"xyz\"}"),
	}
	rspBody = &codec.Body{}
	require.Nil(t,
		proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody,
			client.WithTarget("ip://"+ln.Addr().String()),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithReqHead(header),
			client.WithMetaData("k1", []byte("v1")),
		), "Failed to post")

	require.NotNil(t,
		proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody,
			client.WithTarget("ip://127.0.0.1:180"),
		), "Failed to post")
}

func TestReqHeader(t *testing.T) {
	ctx := context.Background()
	// Invalid url.
	header := &thttp.ClientReqHeader{}
	header.AddHeader("Content-Type", "application/json")
	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://127.0.0.1:18080:www.baidu.com//"),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithReqHead(header),
	)
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}
	err := proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody)
	require.NotNil(t, err)
}

func TestReqHeaderWithContentType(t *testing.T) {
	ctx := context.Background()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	option := transport.WithListener(ln)
	handler := transport.WithHandler(transport.Handler(&h{}))
	tp := thttp.NewServerTransport(newNoopStdHTTPServer)
	require.Nil(t, tp.ListenAndServe(ctx, option, handler), "Failed to new client transport")
	var tests = []struct {
		expected string
	}{
		{"application/json"},
		{"application/jsonp"},
		{"application/jsonp123"},
		{"application/text123"},
	}
	for _, tt := range tests {
		header := &thttp.ClientReqHeader{}
		header.AddHeader("Content-Type", tt.expected)
		proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
			client.WithTarget("ip://"+ln.Addr().String()),
			client.WithSerializationType(codec.SerializationTypeForm),
			client.WithReqHead(header),
		)
		reqBody := &codec.Body{}
		rspBody := &codec.Body{}
		err := proxy.Post(ctx, "/trpc.test.helloworld.Greeter/SayHello", reqBody, rspBody)
		require.Nil(t, err)
	}
}

func TestHandler(t *testing.T) {
	var (
		handler = func(w http.ResponseWriter, r *http.Request) {
			return
		}
		handlerFunc = func(w http.ResponseWriter, r *http.Request) error {
			return nil
		}
		service = server.New(server.WithProtocol("http"))
	)

	thttp.Handle("*", http.HandlerFunc(handler))
	thttp.HandleFunc("/path/do/not/equal/to/*", handlerFunc)
	thttp.RegisterDefaultService(service)

	for _, method := range thttp.ServiceDesc.Methods {
		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})

		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), errors.New("invalid filter")
		})

		header := &thttp.Header{
			Request:  &http.Request{},
			Response: &httptest.ResponseRecorder{},
		}
		ctx := thttp.WithHeader(context.TODO(), header)
		_, err := method.Func(nil, ctx, func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})
		require.Nil(t, err)
	}
}

func TestMux(t *testing.T) {
	var handler = func(w http.ResponseWriter, r *http.Request) {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	var service = &mockService{}
	thttp.RegisterServiceMux(service, mux)
	desc, _ := service.desc.(*server.ServiceDesc)
	for _, method := range desc.Methods {
		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})

		method.Func(nil, context.TODO(), func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), errors.New("invalid filter")
		})

		req, _ := http.NewRequest("GET", "/", nil)
		header := &thttp.Header{
			Request:  req,
			Response: &httptest.ResponseRecorder{},
		}
		ctx := thttp.WithHeader(context.TODO(), header)
		_, err := method.Func(nil, ctx, func(reqBody interface{}) (filter.ServerChain, error) {
			return make([]filter.ServerFilter, 0), nil
		})
		require.Nil(t, err)
	}
}

// TestCheckRedirect tests set CheckRedirect
func TestCheckRedirect(t *testing.T) {
	ctx := context.Background()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	// server
	go func() {
		// real backend
		h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("real"))
		})
		http.Handle("/real", h)

		// redirect a
		rha := http.RedirectHandler("/b", http.StatusMovedPermanently)
		http.Handle("/a", rha)

		// redirect b
		rhb := http.RedirectHandler("/real", http.StatusMovedPermanently)
		http.Handle("/b", rhb)

		http.Serve(ln, nil)
	}()
	time.Sleep(200 * time.Millisecond)

	// sets CheckRedirect
	checkRedirect := func(_ *http.Request, via []*http.Request) error {
		if len(via) > 1 {
			return errors.New("more than once")
		}
		return nil
	}
	thttp.DefaultClientTransport.(*thttp.ClientTransport).CheckRedirect = checkRedirect
	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
	)
	reqBody := &codec.Body{}
	rspBody := &codec.Body{}
	// only redirect once form /b
	require.Nil(t, proxy.Post(ctx, "/b", reqBody, rspBody))
	// redirect twice from /a
	err = proxy.Post(ctx, "/a", reqBody, rspBody)
	require.NotNil(t, err)
	require.Equal(t, true, strings.Contains(err.Error(), "more than once"))
}

func TestTransportError(t *testing.T) {
	http.HandleFunc("/timeout", func(http.ResponseWriter, *http.Request) {
		time.Sleep(time.Second)
	})
	http.HandleFunc("/cancel", func(http.ResponseWriter, *http.Request) {})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	go func() { http.Serve(ln, nil) }()
	time.Sleep(200 * time.Millisecond)

	proxy := thttp.NewClientProxy("trpc.test.helloworld.Greeter",
		client.WithTarget("ip://"+ln.Addr().String()),
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTimeout(time.Millisecond*500),
	)
	rspBody := &codec.Body{}

	err = proxy.Get(context.Background(), "/timeout", rspBody)
	terr, ok := err.(*errs.Error)
	require.True(t, ok)
	require.EqualValues(t, terr.Code, int32(errs.RetClientTimeout))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = proxy.Get(ctx, "/cancel", rspBody)
	terr, ok = err.(*errs.Error)
	require.True(t, ok)
	require.EqualValues(t, terr.Code, int32(errs.RetClientCanceled))
}

func TestClientRoundDyeing(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithDyeing(true)
	dyeingKey := "dyeingkey"
	msg.WithDyeingKey(dyeingKey)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	meta := codec.MetaData{
		thttp.TrpcDyeingKey: []byte(dyeingKey),
	}
	msg.WithClientMetaData(meta)
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Equal(t, req.Header.Get(thttp.TrpcMessageType),
		strconv.Itoa(int(trpcpb.TrpcMessageType_TRPC_DYEING_MESSAGE)))
}

func TestClientRoundEnvTransfer(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithEnvTransfer("feat,master")
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), thttp.TrpcEnv)
}

func TestDisableBase64EncodeTransInfo(t *testing.T) {
	ctx := context.Background()
	ct := thttp.NewClientTransport(false, transport.WithDisableEncodeTransInfoBase64())
	ctx, msg := codec.WithNewMessage(ctx)
	var (
		envTrans  = "feat,master"
		metaVal   = "value"
		dyeingKey = "dyeingkey"
	)
	msg.WithEnvTransfer(envTrans)
	msg.WithClientMetaData(codec.MetaData{"key": []byte(metaVal)})
	msg.WithDyeing(true)
	msg.WithDyeingKey(dyeingKey)
	msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err := ct.RoundTrip(ctx, nil)
	require.NotNil(t, err)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), envTrans)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), metaVal)
	require.Contains(t, req.Header.Get(thttp.TrpcTransInfo), dyeingKey)
}

func TestDisableServiceRouterTransInfo(t *testing.T) {
	ctx := context.Background()
	a := require.New(t)
	ct := thttp.NewClientTransport(false)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithClientMetaData(codec.MetaData{thttp.TrpcEnv: []byte("orienv")}) // this emulate decode trpc protocol client request
	msg.WithEnvTransfer("feat,master")
	req := &http.Request{
		Header: http.Header{},
	}
	reqHeader := &thttp.ClientReqHeader{
		Request: req,
	}
	msg.WithClientReqHead(reqHeader)
	rspHeader := &thttp.ClientRspHeader{}
	msg.WithClientRspHead(rspHeader)
	_, err := ct.RoundTrip(ctx, nil)
	a.NotNil(err)
	info, err := thttp.UnmarshalTransInfo(msg, req.Header.Get(thttp.TrpcTransInfo))
	a.NoError(err)
	a.Equal(string(info[thttp.TrpcEnv]), "feat,master")

	msg.WithEnvTransfer("") // DisableServiceRouter would clear EnvTransfer
	_, err = ct.RoundTrip(ctx, nil)
	a.NotNil(err)
	info, err = thttp.UnmarshalTransInfo(msg, req.Header.Get(thttp.TrpcTransInfo))
	a.NoError(err)
	a.Equal(string(info[thttp.TrpcEnv]), "")
}

func TestHTTPSUseClientVerify(t *testing.T) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork(network),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"../testdata/ca.pem",
		),
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(t.Name()))
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(
				"../testdata/client.crt",
				"../testdata/client.key",
				"../testdata/ca.pem",
				"localhost",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}

func TestHTTPSSkipClientVerify(t *testing.T) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork(network),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"",
		),
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(t.Name()))
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(
				"", "", "none", "",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}

func TestListenAndServeHTTPHead(t *testing.T) {
	ctx := context.Background()
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	st := thttp.NewServerTransport(newNoopStdHTTPServer)
	require.Nil(t, st.ListenAndServe(ctx,
		transport.WithHandler(&httpHeadHandler{
			func(ctx context.Context, _ []byte) (rsp []byte, err error) {
				head := thttp.Head(ctx)
				head.Response.WriteHeader(http.StatusOK)
				head.Response.Write([]byte(fmt.Sprintf("%+v", thttp.Head(head.Request.Context()) != nil)))
				return
			}}),
		transport.WithListener(ln),
	))
	time.Sleep(200 * time.Millisecond)
	rsp, err := http.Get("http://" + ln.Addr().String())
	require.Nil(t, err)
	bs, err := io.ReadAll(rsp.Body)
	require.Nil(t, err)
	require.Equal(t, fmt.Sprintf("%+v", true), string(bs))
}

type httpHeadHandler struct {
	handle func(ctx context.Context, req []byte) (rsp []byte, err error)
}

func (h *httpHeadHandler) Handle(ctx context.Context, req []byte) (rsp []byte, err error) {
	return h.handle(ctx, req)
}

func TestHTTPStreamFileUpload(t *testing.T) {
	// Start server.
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, &fileHandler{})
	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	// Open and read file.
	fileDir, err := os.Getwd()
	require.Nil(t, err)
	fileName := "README.md"
	filePath := path.Join(fileDir, fileName)
	file, err := os.Open(filePath)
	require.Nil(t, err)
	defer file.Close()
	// Construct multipart form file.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("field_name", filepath.Base(file.Name()))
	require.Nil(t, err)
	io.Copy(part, file)
	require.Nil(t, writer.Close())
	// Add multipart form data header.
	header := http.Header{}
	header.Add("Content-Type", writer.FormDataContentType())
	reqHeader := &thttp.ClientReqHeader{
		Header:  header,
		ReqBody: body, // Stream send.
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	// Upload file.
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHeader),
		))
	require.Equal(t, []byte(fileName), rsp.Data)
}

type fileHandler struct{}

func (*fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h, err := r.FormFile("field_name")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	// Write back file name.
	w.Write([]byte(h.Filename))
	return
}

func TestHTTPStreamRead(t *testing.T) {
	// Start server.
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, &fileServer{})

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
	defer body.Close()            // Do remember to close the body.
	bs, err := io.ReadAll(body)
	require.Nil(t, err)
	require.NotNil(t, bs)
}

func TestHTTPSendReceiveChunk(t *testing.T) {
	// HTTP chunked example:
	//   1. Client sends chunks: Add "chunked" transfer encoding header, and use io.Reader as body.
	//   2. Client reads chunks: The Go/net/http automatically handles the chunked reading.
	//                           Users can simply read resp.Body in a loop until io.EOF.
	//   3. Server reads chunks: Similar to client reads chunks.
	//   4. Server sends chunks: Assert http.ResponseWriter as http.Flusher, call flusher.Flush() after
	//         writing a part of data, it will automatically trigger "chunked" encoding to send a chunk.

	// Start server.
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, &chunkedServer{})

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// Open and read file.
	fileDir, err := os.Getwd()
	require.Nil(t, err)
	fileName := "README.md"
	filePath := path.Join(fileDir, fileName)
	file, err := os.Open(filePath)
	require.Nil(t, err)
	defer file.Close()

	// 1. Client sends chunks.

	// Add request headers.
	header := http.Header{}
	header.Add("Content-Type", "text/plain")
	// Add chunked transfer encoding header.
	header.Add("Transfer-Encoding", "chunked")
	reqHead := &thttp.ClientReqHeader{
		Header:  header,
		ReqBody: file, // Stream send (for chunks).
	}

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHead),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)

	// 2. Client reads chunks.

	// Do stream reads directly from rspHead.Response.Body.
	body := rspHead.Response.Body
	defer body.Close() // Do remember to close the body.
	buf := make([]byte, 4096)
	var idx int
	for {
		n, err := body.Read(buf)
		if err == io.EOF {
			t.Logf("reached io.EOF\n")
			break
		}
		t.Logf("read chunk %d of length %d: %q\n", idx, n, buf[:n])
		idx++
	}
}

type chunkedServer struct{}

func (*chunkedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 3. Server reads chunks.

	// io.ReadAll will read until io.EOF.
	// Go/net/http will automatically handle chunked body reads.
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("io.ReadAll err: %+v", err)))
		return
	}

	// 4. Server sends chunks.

	// Send HTTP chunks using http.Flusher.
	// Reference: https://stackoverflow.com/questions/26769626/send-a-chunked-http-response-from-a-go-server.
	// The "Transfer-Encoding" header will be handled by the writer implicitly, so no need to set it.
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("expected http.ResponseWriter to be an http.Flusher"))
		return
	}
	chunks := 10
	chunkSize := (len(bs) + chunks - 1) / chunks
	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if end > len(bs) {
			end = len(bs)
		}
		w.Write(bs[start:end])
		flusher.Flush() // Trigger "chunked" encoding and send a chunk.
		time.Sleep(500 * time.Millisecond)
	}
	return
}

type fileServer struct{}

func (*fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./README.md")
	return
}

func TestHTTPSendAndReceiveSSE(t *testing.T) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork(network),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		msg := string(bs)
		for i := 0; i < 3; i++ {
			msgBytes := []byte("event: message\n\ndata: " + msg + strconv.Itoa(i) + "\n\n")
			_, err = w.Write(msgBytes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
		return
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	header := http.Header{}
	header.Set("Cache-Control", "no-cache")
	header.Set("Accept", "text/event-stream")
	header.Set("Connection", "keep-alive")
	reqHeader := &thttp.ClientReqHeader{
		Header: header,
	}
	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{Data: []byte("hello")}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHeader),
			client.WithRspHead(rspHead),
		))
	body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
	defer body.Close()            // Do remember to close the body.
	data := make([]byte, 1024)
	for {
		n, err := body.Read(data)
		if err == io.EOF {
			break
		}
		require.Nil(t, err)
		t.Logf("Received message: \n%s\n", string(data[:n]))
	}
}

func TestHTTPClientReqRspDifferentContentType(t *testing.T) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork(network),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
	)
	const (
		hello = "hello "
		key   = "key"
	)
	pattern := "/" + t.Name()
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		req, err := url.ParseQuery(string(bs))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rsp := &helloworld.HelloReply{Message: hello + req.Get(key)}
		bs, err = codec.Marshal(codec.SerializationTypePB, rsp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/protobuf")
		w.Write(bs)
		return
	}))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := make(url.Values)
	req.Add(key, t.Name())
	rsp := &helloworld.HelloReply{}
	require.Nil(t,
		c.Post(context.Background(), pattern, req, rsp,
			client.WithSerializationType(codec.SerializationTypeForm),
		))
	require.Equal(t, hello+t.Name(), rsp.Message)
}

type h struct{}

func (*h) Handle(ctx context.Context, reqBuf []byte) (rsp []byte, err error) {
	fmt.Println("recv http req")
	return nil, nil
}

type testLog struct {
	log.Logger
	errorCh chan error
}

func (ln *testLog) Errorf(format string, args ...interface{}) {
	ln.errorCh <- fmt.Errorf(format, args...)
}

// mockService is a mock service.
type mockService struct {
	desc interface{}
}

// Register registers route information.
func (m *mockService) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	m.desc = serviceDesc
	return nil
}

// Serve runs service.
func (m *mockService) Serve() error {
	return nil
}

// Close closes service.
func (m *mockService) Close(chan struct{}) error {
	return nil
}

type errHandler struct{}

func (*errHandler) Handle(ctx context.Context, reqBuf []byte) (rsp []byte, err error) {
	return nil, errors.New("mock error")
}

type errHeaderHandler struct{}

func (*errHeaderHandler) Handle(ctx context.Context, reqBuf []byte) (rsp []byte, err error) {
	return nil, thttp.ErrEncodeMissingHeader
}
