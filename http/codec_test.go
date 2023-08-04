package http_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	thttp.SetContentType("application/proto", codec.SerializationTypePB)
	thttp.RegisterContentType("application/proto", codec.SerializationTypePB)
	thttp.RegisterSerializer("application/proto", codec.SerializationTypePB, &codec.PBSerialization{})
	thttp.RegisterContentEncoding("gzip", codec.CompressTypeGzip)
	thttp.RegisterStatus(100, 500)

	req := thttp.Request(context.Background())
	require.Nil(t, req, "request empty")
	rsp := thttp.Response(context.Background())
	require.Nil(t, rsp, "response empty")
}

func TestServerEncode(t *testing.T) {
	r := &http.Request{}
	w := &httptest.ResponseRecorder{}
	m := &thttp.Header{Request: r, Response: w}
	ctx := thttp.WithHeader(context.Background(), m)
	msg := codec.Message(ctx)
	msg.WithCompressType(codec.CompressTypeGzip)
	sc := thttp.ServerCodec{}
	_, err := sc.Encode(msg, nil)
	require.Nil(t, err, "failed to encode http")
	require.NotNil(t, thttp.Request(ctx))
	require.NotNil(t, thttp.Response(ctx))
}

func TestServerEncodeWithContentType(t *testing.T) {
	r := &http.Request{}
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	m := &thttp.Header{Request: r, Response: w}
	ctx := thttp.WithHeader(context.Background(), m)
	msg := codec.Message(ctx)
	sc := thttp.ServerCodec{}
	_, err := sc.Encode(msg, nil)
	require.Nil(t, err, "failed to encode http")
}

func TestServerErrEncode(t *testing.T) {
	req := &http.Request{}
	w := &httptest.ResponseRecorder{}
	h := &thttp.Header{Request: req, Response: w}
	ctx := thttp.WithHeader(context.Background(), h)
	msg := codec.Message(ctx)
	msg.WithServerRspErr(errs.ErrServerNoFunc)
	sc := thttp.DefaultServerCodec
	_, err := sc.Encode(msg, nil)
	require.Nil(t, err, "failed to encode err http")

	// After the server returns an error, even there is a response data,
	// it will be ignored and will not be processed or returned.
	rsp := &responseRecorder{}
	h = &thttp.Header{Request: req, Response: rsp}
	ctx = thttp.WithHeader(context.Background(), h)
	msg = codec.Message(ctx)
	msg.WithServerRspErr(errs.ErrServerNoFunc)
	_, err = sc.Encode(msg, []byte("write failed"))
	require.Nil(t, err)
}

func TestNotHead(t *testing.T) {
	msg := codec.Message(context.Background())
	_, err := thttp.DefaultServerCodec.Decode(msg, nil)
	require.NotNil(t, err, "failed to decode get head")
	_, err = thttp.DefaultServerCodec.Encode(msg, nil)
	require.NotNil(t, err, "failed to encode get head")
}

func TestMultipartFormData(t *testing.T) {
	require := require.New(t)
	r, _ := http.NewRequest("POST", "http://www.qq.com/trpc.http.test.helloworld/SayHello", bytes.NewReader([]byte("")))
	r.Header.Add("Content-Type", "multipart/form-data; boundary=--------------------------487682300036072392114180")
	body := `----------------------------487682300036072392114180
Content-Disposition: form-data; name="competition"

NBA
----------------------------487682300036072392114180
Content-Disposition: form-data; name="teams"

湖人
----------------------------487682300036072392114180
Content-Disposition: form-data; name="teams"

勇士
----------------------------487682300036072392114180
Content-Disposition: form-data; name="season"

2021
----------------------------487682300036072392114180
Content-Disposition: form-data; name="file1"; filename="1.txt"
Content-Type: text/plain

1
----------------------------487682300036072392114180
Content-Disposition: form-data; name="file2"; filename="1px.png"
Content-Type: image/png

�PNG

IHDR%�V�PLTE�����
IDA�c�!�3IEND�B�
----------------------------487682300036072392114180
Content-Disposition: form-data; name="file3"; filename="json.json"
Content-Type: application/json

{
    "name":"1"
}
----------------------------487682300036072392114180--
`
	// Decode multipart form data.
	r.Body = io.NopCloser(strings.NewReader(body))
	w := &httptest.ResponseRecorder{}
	header := &thttp.Header{Request: r, Response: w}
	msg := codec.Message(thttp.WithHeader(context.Background(), header))
	in, err := thttp.DefaultServerCodec.Decode(msg, nil)
	require.Nil(err)
	require.Equal("competition=NBA&season=2021&teams=%E6%B9%96%E4%BA%BA&teams=%E5%8B%87%E5%A3%AB", string(in))
	head := thttp.Head(msg.Context())

	// Content-Type: text/plain.
	file1, fileHeader1, err := head.Request.FormFile("file1")
	require.Nil(err)
	defer func() { require.Nil(file1.Close()) }()
	require.Equal("text/plain", fileHeader1.Header.Get("Content-Type"))
	require.Equal("1.txt", fileHeader1.Filename)
	file1Content := make([]byte, 256)
	n, err := io.ReadFull(file1, file1Content)
	require.NotNil(err)
	file1Content = file1Content[:n]
	require.Equal("1", string(file1Content))

	// Content-Type: image/png.
	file2, fileHeader2, err := head.Request.FormFile("file2")
	require.Nil(err)
	defer func() { require.Nil(file2.Close()) }()
	require.Equal("image/png", fileHeader2.Header.Get("Content-Type"))
	require.Equal("1px.png", fileHeader2.Filename)

	// Content-Type: application/json.
	file3, fileHeader3, err := head.Request.FormFile("file3")
	require.Nil(err)
	defer func() { require.Nil(file3.Close()) }()
	require.Equal("application/json", fileHeader3.Header.Get("Content-Type"))
	require.Equal("json.json", fileHeader3.Filename)
	defer func() { require.Nil(file1.Close()) }()
	file3Content := make([]byte, 256)
	n, err = io.ReadFull(file3, file3Content)
	require.NotNil(err)
	file3Content = file3Content[:n]
	expected := `{
    "name":"1"
}`
	require.Equal(expected, string(file3Content))

	// Encode json response data.
	rsp := []byte(`{"competitionID":100000,"player":"opta"}`)
	b, err := thttp.DefaultServerCodec.Encode(msg, rsp)
	require.Nil(err)
	ct := header.Response.Header().Get("Content-Type")
	require.Equal("application/json", ct)
	require.Nil(b)
}

func TestServerDecodeHTTPHeader(t *testing.T) {
	r, err := http.NewRequest("POST", "http://www.qq.com/trpc.http.test.helloworld/SayHello", bytes.NewReader([]byte("")))
	require.Nil(t, err)
	r.Header.Add("Content-Encoding", "gzip")
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add(thttp.TrpcVersion, "1")
	r.Header.Add(thttp.TrpcCallType, "1")
	r.Header.Add(thttp.TrpcMessageType, "1")
	r.Header.Add(thttp.TrpcRequestID, "1")
	r.Header.Add(thttp.TrpcTimeout, "1000")
	r.Header.Add(thttp.TrpcCaller, "trpc.app.server.helloworld")
	r.Header.Add(thttp.TrpcCallee, "trpc.http.test.helloworld")
	// Request data must encode by base64 first.
	// val1 -> dmFsMQ==   val2 -> dmFsMg==
	r.Header.Add(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsMg=="}`)
	w := &httptest.ResponseRecorder{}
	h := &thttp.Header{Request: r, Response: w}
	ctx := thttp.WithHeader(context.Background(), h)
	msg := codec.Message(ctx)
	_, err = thttp.DefaultServerCodec.Decode(msg, nil)
	require.Nil(t, err, "failed to decode get body")

	require.Equal(t, codec.CompressTypeGzip, msg.CompressType())
	require.Equal(t, codec.SerializationTypeJSON, msg.SerializationType())

	req, ok := msg.ServerReqHead().(*trpcpb.RequestProtocol)
	require.True(t, ok)
	require.NotNil(t, req, "failed to decode get trpc req head")
	require.Equal(t, 1, int(req.GetVersion()))
	require.Equal(t, 1, int(req.GetCallType()))
	require.Equal(t, 1, int(req.GetMessageType()))
	require.Equal(t, 1, int(req.GetRequestId()))
	require.Equal(t, 1000, int(req.GetTimeout()))
	require.Equal(t, "trpc.app.server.helloworld", string(req.GetCaller()))
	require.Equal(t, "trpc.http.test.helloworld", string(req.GetCallee()))
	require.Equal(t, "val1", string(req.GetTransInfo()["key1"]))
	require.Equal(t, "val2", string(req.GetTransInfo()["key2"]))

	// JSON unmarshal failed.
	r.Header.Set(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsMg=="`)
	w = &httptest.ResponseRecorder{}
	h = &thttp.Header{Request: r, Response: w}
	ctx = thttp.WithHeader(context.Background(), h)
	msg = codec.Message(ctx)
	_, err = thttp.DefaultServerCodec.Decode(msg, nil)
	require.NotNil(t, err)

	// base64 decode failed.
	// If parsing fails then use raw data.
	r.Header.Set(thttp.TrpcTransInfo, fmt.Sprintf(`{"%s":"%s"}`, thttp.TrpcEnv, "Production"))
	w = &httptest.ResponseRecorder{}
	h = &thttp.Header{Request: r, Response: w}
	ctx = thttp.WithHeader(context.Background(), h)
	msg = codec.Message(ctx)
	_, err = thttp.DefaultServerCodec.Decode(msg, nil)
	req, _ = msg.ServerReqHead().(*trpcpb.RequestProtocol)
	require.Nil(t, err)
	require.Equal(t, "Production", string(req.GetTransInfo()[thttp.TrpcEnv]))

	// ReadAll failed.
	r.Header.Set(thttp.TrpcTransInfo, `{"key1":"dmFsMQ==", "key2":"dmFsM-1"}`)
	w = &httptest.ResponseRecorder{}
	rp, _ := io.Pipe()
	_ = rp.CloseWithError(err)
	r.Body = rp
	h = &thttp.Header{Request: r, Response: w}
	ctx = thttp.WithHeader(context.Background(), h)
	msg = codec.Message(ctx)
	_, err = thttp.DefaultServerCodec.Decode(msg, nil)
	require.NotNil(t, err)
}

func TestServerDecode(t *testing.T) {
	r, _ := http.NewRequest("GET", "www.qq.com/xyz=abc", bytes.NewReader([]byte("")))
	w := &httptest.ResponseRecorder{}
	m := &thttp.Header{Request: r, Response: w}
	ctx := thttp.WithHeader(context.Background(), m)
	msg := codec.Message(ctx)
	msg.WithServerRspErr(errs.ErrServerNoFunc)
	_, err := thttp.DefaultServerCodec.Decode(msg, nil)
	require.Nil(t, err, "failed to decode get body")
}

func TestServerPostDecode(t *testing.T) {
	r, _ := http.NewRequest("POST", "www.qq.com", bytes.NewReader([]byte("{xyz:\"abc\"")))
	w := &httptest.ResponseRecorder{}
	m := &thttp.Header{Request: r, Response: w}
	ctx := thttp.WithHeader(context.Background(), m)
	msg := codec.Message(ctx)
	msg.WithServerRspErr(errs.ErrServerNoFunc)
	sc := thttp.ServerCodec{}
	_, err := sc.Decode(msg, nil)
	require.Nil(t, err, "failed to decode post body")
}

func TestClientEncode(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	cc := thttp.ClientCodec{}
	_, err := cc.Encode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to encode")
	require.NotNil(t, msg.ClientReqHead(), "req head is nil")
}

func TestClientEncodeWithHeader(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	httpReqHeader := &thttp.ClientReqHeader{}
	msg.WithClientReqHead(httpReqHeader)
	cc := thttp.ClientCodec{}
	_, err := cc.Encode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "failed to encode")
	require.NotNil(t, msg.ClientReqHead(), "req head is nil")

	// Failed to parse req header.
	_, msg = codec.WithNewMessage(context.Background())
	reqHeader := &thttp.ClientRspHeader{}
	msg.WithClientReqHead(reqHeader)
	cc = thttp.ClientCodec{}
	_, err = cc.Encode(msg, nil)
	require.NotNil(t, err)

	// Failed to parse rsp header.
	_, msg = codec.WithNewMessage(context.Background())
	rspHeader := &thttp.ClientReqHeader{}
	msg.WithClientRspHead(rspHeader)
	cc = thttp.ClientCodec{}
	_, err = cc.Encode(msg, nil)
	require.NotNil(t, err)
}

func TestClientErrDecode(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	httprsp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[0].Raw)), &http.Request{Method: "POST"})
	require.Nil(t, err)
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	cc := thttp.ClientCodec{}
	_, err = cc.Decode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err)
	require.NotNil(t, msg.ClientRspHead(), "req head is nil")

	// Failed to parse rsp header.
	_, m := codec.WithNewMessage(context.Background())
	m.WithClientRspHead(&thttp.ClientReqHeader{})
	cc = thttp.ClientCodec{}
	_, err = cc.Decode(m, nil)
	require.NotNil(t, err)

	// Failed to read body.
	rp, _ := io.Pipe()
	_ = rp.CloseWithError(errors.New("read failed"))
	httprsp, err = http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[0].Raw)),
		&http.Request{Method: "POST"})
	require.Nil(t, err)
	httprsp.Body = rp
	httprsp.StatusCode = http.StatusOK

	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	cc = thttp.ClientCodec{}
	_, err = cc.Decode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.NotNil(t, err)

	// HTTP status code is 300 (when status code >= 300, ClientCodec.Decode should return response error).
	httprsp, err = http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[0].Raw)),
		&http.Request{Method: "POST"})
	require.Nil(t, err)
	httprsp.StatusCode = http.StatusMultipleChoices
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})

	cc = thttp.ClientCodec{}
	_, err = cc.Decode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to decode")
	require.NotNil(t, msg.ClientRspErr(), "response error should not be nil")
}

func TestClientSuccessDecode(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	httprsp, _ := http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[1].Raw)),
		&http.Request{Method: "POST"})
	httprsp.Header.Add("Content-Encoding", "gzip")
	httprsp.Header.Add("trpc-trans-info", `{"key1":"val1", "key2":"val2"}`)
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	body, err := thttp.DefaultClientCodec.Decode(msg, []byte("{\"username\":\"xyz\","+
		"\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to decode")
	require.NotNil(t, msg.ClientRspHead(), "req head is nil")
	require.Equal(t, string(body), respTests[1].Body, "body is error", string(body))
	require.Equal(t, codec.CompressTypeGzip, msg.CompressType())

	// HTTP status code 101.
	httprsp, err = http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[2].Raw)),
		&http.Request{Method: "POST"})
	require.Nil(t, err)
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})

	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	cc := thttp.ClientCodec{}
	body, err = cc.Decode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to decode")
	require.Empty(t, body)

	// HTTP status code 201.
	httprsp, err = http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[0].Raw)),
		&http.Request{Method: "POST"})
	require.Nil(t, err)
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	httprsp.StatusCode = http.StatusCreated

	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	cc = thttp.ClientCodec{}
	body, err = cc.Decode(msg, []byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to decode")
	require.Equal(t, respTests[0].Body, string(body), "body is error", string(body))
}

func TestClientRetDecode(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	httprsp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[1].Raw)), &http.Request{Method: "POST"})
	require.Nil(t, err)
	httprsp.Header.Add("trpc-ret", "1")
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	_, err = thttp.DefaultClientCodec.Decode(msg,
		[]byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to decode")
	require.NotNil(t, msg.ClientRspErr())
	require.EqualValues(t, 1, errs.Code(msg.ClientRspErr()))
}

func TestClientFuncRetDecode(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	httprsp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(respTests[1].Raw)), &http.Request{Method: "POST"})
	require.Nil(t, err)
	httprsp.Header.Add("trpc-func-ret", "1000")
	httprsp.Header.Add("Content-Type", "application/json")
	msg.WithClientRspHead(&thttp.ClientRspHeader{Response: httprsp})
	_, err = thttp.DefaultClientCodec.Decode(msg,
		[]byte("{\"username\":\"xyz\",\"password\":\"xyz\",\"from\":\"xyz\"}"))
	require.Nil(t, err, "Failed to decode")
	require.NotNil(t, msg.ClientRspErr())
	require.EqualValues(t, 1000, errs.Code(msg.ClientRspErr()))
}

func TestServiceDecodeWithHeader(t *testing.T) {
	req := &http.Request{
		URL: &url.URL{
			Path: "my_path",
		},
	}
	header := &thttp.Header{
		Request: req,
	}

	sc := &thttp.ServerCodec{}
	ctx := thttp.WithHeader(context.Background(), header)
	_, msg := codec.WithNewMessage(ctx)

	_, err := sc.Decode(msg, nil)
	assert.Nil(t, err)

	method := msg.CalleeMethod()
	rpcName := msg.ServerRPCName()

	assert.Equal(t, method, req.URL.Path)
	assert.Equal(t, rpcName, req.URL.Path)
}

func TestServerCodecDecodeTransInfo(t *testing.T) {
	transInfo := map[string]string{
		thttp.TrpcEnv:       base64.StdEncoding.EncodeToString([]byte("env-test")),
		thttp.TrpcDyeingKey: base64.StdEncoding.EncodeToString([]byte("dyeing-test")),
	}
	data, err := json.Marshal(transInfo)
	require.Nil(t, err)
	head := http.Header{}
	head.Add(thttp.TrpcTransInfo, string(data))

	req := &http.Request{
		Header: head,
		URL:    &url.URL{},
	}

	header := &thttp.Header{
		Request: req,
	}

	sc := &thttp.ServerCodec{
		AutoGenTrpcHead: true,
	}
	ctx := thttp.WithHeader(context.Background(), header)
	_, msg := codec.WithNewMessage(ctx)

	_, err = sc.Decode(msg, nil)
	require.Nil(t, err)

	assert.Equal(t, msg.EnvTransfer(), "env-test")
	assert.Equal(t, msg.DyeingKey(), "dyeing-test")
}

func TestDisableEncodeBase64(t *testing.T) {
	r, err := http.NewRequest("POST", "/SayHello", bytes.NewReader([]byte("")))
	require.Nil(t, err)
	w := &httptest.ResponseRecorder{}
	h := &thttp.Header{Request: r, Response: w}
	ctx := thttp.WithHeader(context.Background(), h)
	ctx, msg := codec.EnsureMessage(ctx)
	msg.WithServerMetaData(codec.MetaData{"meta-key": []byte("meta-value")})

	serverCodec := thttp.ServerCodec{
		DisableEncodeTransInfoBase64: true,
	}
	_, err = serverCodec.Encode(msg, nil)
	require.Nil(t, err)
	rsp := thttp.Head(ctx).Response
	require.Contains(t, string(rsp.Header().Get(thttp.TrpcTransInfo)), "meta-value")
}

func TestCoexistenceOfHTTPRPCAndNoProtocol(t *testing.T) {
	defer func() { thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0] }()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.test.hello.service" + t.Name()
	s := server.New(
		server.WithServiceName(serviceName),
		server.WithListener(ln),
		// Although the "http" protocol is represented as an HTTP RPC service and
		// the standard HTTP service has its own protocol "http_no_protocol", some
		// users require that both protocols can coexist in the same service
		// (with the same ip and port).
		// This requires that the standard HTTP handler function can still read the
		// request body, even if the `AutoReadBody` field in the default server
		// codec `DefaultServerCodec` for the `http` protocol is `true`.
		server.WithProtocol("http"),
	)
	// Register standard HTTP handle.
	thttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) error {
		s := &codec.JSONPBSerialization{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		req := &helloworld.HelloRequest{}
		if err := s.Unmarshal(body, req); err != nil {
			return err
		}
		rsp := &helloworld.HelloReply{Message: req.Name}
		body, err = s.Marshal(rsp)
		if err != nil {
			return err
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return nil
	})
	thttp.RegisterNoProtocolService(s)
	// Register protocol file service (HTTP RPC) implementation.
	helloworld.RegisterGreeterService(s, &greeterImpl{})

	// Start server.
	go s.Serve()

	ctx := context.Background()
	target := "ip://" + ln.Addr().String()

	// Send standard HTTP request.
	c := thttp.NewClientProxy(serviceName, client.WithTarget(target))
	msg := "hello"
	req := &helloworld.HelloRequest{Name: msg}
	rsp := &helloworld.HelloReply{}
	require.Nil(t, c.Post(ctx, "/", req, rsp,
		client.WithSerializationType(codec.SerializationTypeJSON)))
	require.Equal(t, msg, rsp.Message)

	// Send HTTP RPC request.
	proxy := helloworld.NewGreeterClientProxy(client.WithTarget(target), client.WithProtocol("http"))
	resp, err := proxy.SayHello(ctx, &helloworld.HelloRequest{Name: msg})
	require.Nil(t, err)
	require.Equal(t, msg, resp.Message)
}

type greeterImpl struct{}

func (i *greeterImpl) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: req.Name}, nil
}

func (i *greeterImpl) SayHi(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return nil, nil
}

type responseRecorder struct {
	httptest.ResponseRecorder
}

func (r *responseRecorder) Write(buf []byte) (int, error) {
	return 0, errors.New("write failed")
}

type respTest struct {
	Raw  string
	Body string
}

var respTests = []respTest{
	// Unchunked response without Content-Length.
	{
		"HTTP/1.0 404 NOT FOUND\r\n" +
			"Connection: close\r\n" +
			"\r\n" +
			"Body here\n",

		"Body here\n",
	},

	// Unchunked HTTP/1.1 response without Content-Length or
	// Connection headers.
	{
		"HTTP/1.1 200 OK\r\n" +
			"\r\n" +
			"{\"msg\":\"from hi\"}\n",

		"{\"msg\":\"from hi\"}\n",
	},

	// Unchunked HTTP/1.1 response without body.
	{
		"HTTP/1.1 101 Switching Protocols\r\n" +
			"\r\n",

		"",
	}}
