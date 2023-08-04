// Package stream_test Unit test for package stream.
package stream_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	"trpc.group/trpc-go/trpc-go/stream"
	"trpc.group/trpc-go/trpc-go/transport"

	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

type fakeTransport struct {
	expectChan chan recvExpect
}

// RoundTrip Mock RoundTrip method.
func (c *fakeTransport) RoundTrip(ctx context.Context, req []byte,
	roundTripOpts ...transport.RoundTripOption) (rsp []byte, err error) {
	return nil, nil
}

// Send Mock Send method.
func (c *fakeTransport) Send(ctx context.Context, req []byte, opts ...transport.RoundTripOption) error {
	err, ok := ctx.Value("send-error").(string)
	if ok {
		return errors.New(err)
	}
	return nil
}

type recvExpect func(*trpc.FrameHead, codec.Msg) ([]byte, error)

// Recv Mock recv method.
func (c *fakeTransport) Recv(ctx context.Context, opts ...transport.RoundTripOption) ([]byte, error) {
	msg := codec.Message(ctx)
	var fh *trpc.FrameHead
	fh, ok := msg.FrameHead().(*trpc.FrameHead)
	if !ok {
		fh = &trpc.FrameHead{}
		msg.WithFrameHead(fh)
	}
	f := <-c.expectChan
	return f(fh, msg)
}

// Init Mock Init method.
func (c *fakeTransport) Init(ctx context.Context, opts ...transport.RoundTripOption) error {
	return nil
}

// Close Mock Close method.
func (c *fakeTransport) Close(ctx context.Context) {
	return
}

type fakeCodec struct {
}

// Encode Mock codec Encode method.
func (c *fakeCodec) Encode(msg codec.Msg, reqBody []byte) (reqBuf []byte, err error) {
	if string(reqBody) == "failbody" {
		return nil, errors.New("encode fail")
	}
	return reqBody, nil
}

// Decode Mock codec Decode method.
func (c *fakeCodec) Decode(msg codec.Msg, rspBuf []byte) (rspBody []byte, err error) {
	if string(rspBuf) == "businessfail" {
		return nil, errors.New("businessfail")
	}
	if string(rspBuf) == "msgfail" {
		msg.WithClientRspErr(errors.New("msgfail"))
		return nil, nil
	}
	return rspBuf, nil
}

// TestMain tests the Main function.
func TestMain(m *testing.M) {
	transport.DefaultServerTransport = &fakeServerTransport{}
	m.Run()
}

// TestClient tests the streaming client.
func TestClient(t *testing.T) {
	var reqBody = &codec.Body{Data: []byte("body")}
	var rspBody = &codec.Body{}
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	codec.Register("fake-nil", nil, nil)

	cli := stream.NewStreamClient()
	assert.Equal(t, cli, stream.DefaultStreamClient)

	ctx := context.Background()
	var ft = &fakeTransport{expectChan: make(chan recvExpect, 1)}
	transport.DefaultClientTransport = ft

	f := func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, nil
	}
	ft.expectChan <- f
	cs, err := cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithStreamTransport(ft))
	assert.NotNil(t, cs)
	assert.Nil(t, err)

	// Test Context.
	resultCtx := cs.Context()
	assert.NotNil(t, resultCtx)
	// Test to send data normally.
	err = cs.SendMsg(reqBody)
	assert.Nil(t, err)

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
		return []byte("body"), nil
	}
	ft.expectChan <- f

	// Test to receive data normally.
	err = cs.RecvMsg(rspBody)
	assert.Nil(t, err)
	assert.Equal(t, rspBody.Data, []byte("body"))

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE)
		return nil, nil
	}
	ft.expectChan <- f

	// Test received io.EOF.
	rspBody = &codec.Body{}
	err = cs.RecvMsg(rspBody)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, rspBody.Data)

	err = cs.CloseSend()
	assert.Nil(t, err)

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, nil
	}
	ft.expectChan <- f
	cs, err = cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft))
	assert.NotNil(t, cs)
	assert.Nil(t, err)

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		msg.WithClientRspErr(errors.New("close type is reset"))
		fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE)
		return nil, nil
	}
	ft.expectChan <- f
	// test reset.
	rspBody = &codec.Body{}
	err = cs.RecvMsg(rspBody)
	assert.NotNil(t, err)
	assert.Nil(t, rspBody.Data)
	assert.Contains(t, err.Error(), "close type is reset")

}

// TestClientFlowControl tests the streaming client.
func TestClientFlowControl(t *testing.T) {
	var reqBody = &codec.Body{Data: []byte("body")}

	var rspBody = &codec.Body{}
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	codec.Register("fake-nil", nil, nil)

	cli := stream.NewStreamClient()
	assert.Equal(t, cli, stream.DefaultStreamClient)

	ctx := context.Background()
	var ft = &fakeTransport{expectChan: make(chan recvExpect, 1)}
	transport.DefaultClientTransport = ft

	f := func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT)
		msg.WithStreamFrame(&trpcpb.TrpcStreamInitMeta{InitWindowSize: 2000})
		return nil, nil
	}
	ft.expectChan <- f

	cs, err := cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft))
	assert.NotNil(t, cs)
	assert.Nil(t, err)

	// Test Context.
	resultCtx := cs.Context()
	assert.NotNil(t, resultCtx)
	// Test to send data normally.
	err = cs.SendMsg(reqBody)
	assert.Nil(t, err)

	for i := 0; i < 20000; i++ {
		f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
			fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA)
			return []byte("body"), nil
		}
		ft.expectChan <- f
		// Test to receive data normally.
		err = cs.RecvMsg(rspBody)
		assert.Nil(t, err)
		assert.Equal(t, rspBody.Data, []byte("body"))
	}

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		fh.StreamFrameType = uint8(trpcpb.TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE)
		return nil, nil
	}
	ft.expectChan <- f

	// Test received io.EOF.
	rspBody = &codec.Body{}
	err = cs.RecvMsg(rspBody)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, rspBody.Data)
}

// TestClientError tests the case of streaming Client error handling.
func TestClientError(t *testing.T) {
	var rspBody = &codec.Body{}
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	codec.Register("fake-nil", nil, nil)

	cli := stream.NewStreamClient()
	assert.Equal(t, cli, stream.DefaultStreamClient)

	var ft = &fakeTransport{expectChan: make(chan recvExpect, 1)}
	transport.DefaultClientTransport = ft
	f := func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, errors.New("init error")
	}
	ft.expectChan <- f

	// Test for init transport errors.
	cs, err := cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft))
	assert.Nil(t, cs)
	assert.NotNil(t, err)

	// test Init error.
	cs, err = cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake-nil"), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft))
	assert.Nil(t, cs)
	assert.NotNil(t, err)

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, nil
	}
	ft.expectChan <- f
	cs, err = cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft), client.WithClientStreamQueueSize(100000))
	assert.NotNil(t, cs)
	assert.Nil(t, err)
	// test recv data error.
	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, errors.New("recv data error")
	}
	ft.expectChan <- f
	err = cs.RecvMsg(rspBody)
	assert.NotNil(t, err)
	assert.Nil(t, rspBody.Data)

	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		msg.WithClientRspErr(errors.New("test init with clientRspError"))
		return nil, nil
	}
	ft.expectChan <- f
	cs, err = cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(ft), client.WithClientStreamQueueSize(100000))
	assert.Nil(t, cs)
	assert.NotNil(t, err)

}

// TestClientContext tests the case of streaming client context cancel and timeout.
func TestClientContext(t *testing.T) {

	var rspBody = &codec.Body{}
	codec.RegisterSerializer(0, &codec.NoopSerialization{})
	codec.Register("fake", nil, &fakeCodec{})
	codec.Register("fake-nil", nil, nil)

	cli := stream.NewStreamClient()
	assert.Equal(t, cli, stream.DefaultStreamClient)

	ctx := context.Background()
	var ft = &fakeTransport{expectChan: make(chan recvExpect, 1)}
	transport.DefaultClientTransport = ft
	// test context cancel situation.
	f := func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, nil
	}
	ft.expectChan <- f
	ctx, cancel := context.WithCancel(context.Background())
	cs, err := cli.NewStream(ctx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft))
	assert.NotNil(t, cs)
	assert.Nil(t, err)
	cancel()
	err = cs.RecvMsg(rspBody)
	assert.Contains(t, err.Error(), "tcp client stream canceled before recv")
	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, errors.New("close it")
	}
	ft.expectChan <- f
	time.Sleep(5 * time.Millisecond)
	// test context timeout situation.
	f = func(fh *trpc.FrameHead, msg codec.Msg) ([]byte, error) {
		return nil, nil
	}
	ft.expectChan <- f

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cs, err = cli.NewStream(timeoutCtx, bidiDesc, "/trpc.test.helloworld.Greeter/SayHello",
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithProtocol("fake"), client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentCompressType(codec.CompressTypeNoop),
		client.WithTransport(ft),
		client.WithStreamTransport(ft))
	assert.NotNil(t, cs)
	assert.Nil(t, err)

	err = cs.RecvMsg(rspBody)
	assert.Contains(t, err.Error(), "tcp client stream canceled timeout before recv")
}

func clientFilterAdd1(ctx context.Context, desc *client.ClientStreamDesc, newStream client.Streamer) (client.ClientStream, error) {
	var msg codec.Msg
	ctx, msg = codec.EnsureMessage(ctx)
	meta := msg.ClientMetaData()
	if meta == nil {
		meta = codec.MetaData{}
	}
	meta[testKey1] = []byte(testData[testKey1])
	msg.WithClientMetaData(meta)

	s, err := newStream(ctx, desc)
	if err != nil {
		return nil, err
	}

	return newWrappedClientStream(s), nil
}

func clientFilterAdd2(ctx context.Context, desc *client.ClientStreamDesc, newStream client.Streamer) (client.ClientStream, error) {
	var msg codec.Msg
	ctx, msg = codec.EnsureMessage(ctx)
	meta := msg.ClientMetaData()
	if meta == nil {
		meta = codec.MetaData{}
	}
	meta[testKey2] = []byte(testData[testKey2])
	msg.WithClientMetaData(meta)

	s, err := newStream(ctx, desc)
	if err != nil {
		return nil, err
	}
	return newWrappedClientStream(s), nil
}

type wrappedClientStream struct {
	client.ClientStream
}

func newWrappedClientStream(s client.ClientStream) client.ClientStream {
	return &wrappedClientStream{s}
}

func (w *wrappedClientStream) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)
	num := binary.LittleEndian.Uint64(m.(*codec.Body).Data[:8])
	binary.LittleEndian.PutUint64(m.(*codec.Body).Data[:8], num+1)
	return err
}

func (w *wrappedClientStream) SendMsg(m interface{}) error {
	num := binary.LittleEndian.Uint64(m.(*codec.Body).Data[:8])
	binary.LittleEndian.PutUint64(m.(*codec.Body).Data[:8], num+1)
	return w.ClientStream.SendMsg(m)
}

func TestClientStreamClientFilters(t *testing.T) {
	rawData := make([]byte, 1024)
	rand.Read(rawData)
	var beginNum uint64 = 100

	counts := 1000
	svrOpts := []server.Option{
		server.WithAddress("127.0.0.1:30211"),
		server.WithStreamFilters(serverFilterAdd1, serverFilterAdd2),
	}
	handle := func(s server.Stream) error {
		var req *codec.Body

		// server receives data.
		for i := 0; i < counts; i++ {
			req = getBytes(1024)
			err := s.RecvMsg(req)
			assert.Nil(t, err)
			resultNum := binary.LittleEndian.Uint64(req.Data[:8])

			// After the client SendMsg + server RecvMsg, two Filter, Num+4.
			assert.Equal(t, beginNum+4, resultNum)
			assert.Equal(t, rawData[8:], req.Data[8:])
		}
		err := s.RecvMsg(getBytes(1024))
		assert.Equal(t, io.EOF, err)

		// server sends data.
		rsp := getBytes(1024)
		for i := 0; i < counts; i++ {
			copy(rsp.Data, req.Data)
			err = s.SendMsg(rsp)
			assert.Nil(t, err)
		}
		return nil
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{
		client.WithTarget("ip://127.0.0.1:30211"),
		client.WithStreamFilters(clientFilterAdd1, clientFilterAdd2),
	}
	cliStream, err := getClientStream(context.Background(), bidiDesc, cliOpts)
	assert.Nil(t, err)

	// client sends data.
	for i := 0; i < counts; i++ {
		req := getBytes(1024)
		copy(req.Data, rawData)
		binary.LittleEndian.PutUint64(req.Data[:8], beginNum)

		err = cliStream.SendMsg(req)
		assert.Nil(t, err)
	}
	err = cliStream.CloseSend()
	assert.Nil(t, err)

	// client receives data.
	for i := 0; i < counts; i++ {
		rsp := getBytes(1024)
		err = cliStream.RecvMsg(rsp)
		assert.Nil(t, err)

		// After the client once SendMsg, once RecvMsg, two Filter, Num+4.
		resultNum := binary.LittleEndian.Uint64(rsp.Data[:8])
		assert.Equal(t, beginNum+8, resultNum)
		assert.Equal(t, rawData[8:], rsp.Data[8:])
	}
	rsp := getBytes(1024)
	err = cliStream.RecvMsg(rsp)
	assert.Equal(t, io.EOF, err)
}

func TestClientStreamFlowControlStop(t *testing.T) {
	windows := 102400
	dataLen := 1024
	maxSends := windows / dataLen
	svrOpts := []server.Option{
		server.WithAddress("127.0.0.1:30211"),
		server.WithMaxWindowSize(uint32(windows)),
	}
	handle := func(s server.Stream) error {
		time.Sleep(time.Hour)
		return nil
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(200*time.Millisecond))
	defer cancel()
	cliOpts := []client.Option{client.WithTarget("ip://127.0.0.1:30211")}
	cliStream, err := getClientStream(ctx, bidiDesc, cliOpts)
	assert.Nil(t, err)

	req := getBytes(dataLen)
	rand.Read(req.Data)

	for i := 0; i < maxSends; i++ {
		err = cliStream.SendMsg(req)
		assert.Nil(t, err)
	}
	err = cliStream.SendMsg(req)
	assert.Equal(t, errors.New("stream is already closed"), err)
}

func TestServerStreamFlowControlStop(t *testing.T) {
	windows := 102400
	dataLen := 1024
	maxSends := windows / dataLen
	waitCh := make(chan struct{}, 1)
	svrOpts := []server.Option{server.WithAddress("127.0.0.1:30211")}
	handle := func(s server.Stream) error {
		rsp := getBytes(dataLen)
		rand.Read(rsp.Data)
		for i := 0; i < maxSends; i++ {
			err := s.SendMsg(rsp)
			assert.Nil(t, err)
		}

		finish := make(chan struct{}, 1)
		go func() {
			err := s.SendMsg(rsp)
			assert.Equal(t, errors.New("stream is already closed"), err)
			finish <- struct{}{}
		}()

		deadline := time.NewTimer(200 * time.Millisecond)
		select {
		case <-deadline.C:
		case <-finish:
			assert.Fail(t, "SendMsg should block")
		}

		waitCh <- struct{}{}
		return nil
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{
		client.WithTarget("ip://127.0.0.1:30211"),
		client.WithMaxWindowSize(uint32(windows)),
	}
	_, err := getClientStream(context.Background(), bidiDesc, cliOpts)
	assert.Nil(t, err)
	<-waitCh
}

func TestClientStreamSendRecvNoBlock(t *testing.T) {
	svrOpts := []server.Option{server.WithAddress("127.0.0.1:30210")}
	handle := func(s server.Stream) error {
		// Must sleep, to avoid returning before receiving the first packet from the client,
		// resulting in the processing of the first packet returns an error,
		// losing the chance for the test client to block on the second SendMsg.
		time.Sleep(200 * time.Millisecond)
		return errors.New("test error")
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{client.WithTarget("ip://127.0.0.1:30210")}
	cliStream, err := getClientStream(context.Background(), bidiDesc, cliOpts)
	assert.Nil(t, err)

	// defaultInitWindowSize = 65535.
	req := getBytes(65535)
	err = cliStream.SendMsg(req)
	assert.Nil(t, err)

	err = cliStream.SendMsg(req)
	fmt.Println(err)
	assert.NotNil(t, err)

	rsp := getBytes(1024)
	err = cliStream.RecvMsg(rsp)
	assert.NotNil(t, err)
}

func TestServerStreamSendRecvNoBlock(t *testing.T) {
	svrOpts := []server.Option{server.WithAddress("127.0.0.1:30210")}
	SendMsgReturn := make(chan struct{}, 1)
	RecvMsgReturn := make(chan struct{}, 1)
	handle := func(s server.Stream) error {
		go func() {
			msg := getBytes(65535)
			s.SendMsg(msg)
			s.SendMsg(msg)
			SendMsgReturn <- struct{}{}
		}()
		go func() {
			msg := getBytes(1024)
			s.RecvMsg(msg)
			s.RecvMsg(msg)
			RecvMsgReturn <- struct{}{}
		}()
		// Must sleep, to avoid the second SendMsg does not enter the waiting window to block.
		time.Sleep(200 * time.Millisecond)
		return nil
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{client.WithTarget("ip://127.0.0.1:30210")}
	_, err := getClientStream(context.Background(), bidiDesc, cliOpts)
	assert.Nil(t, err)

	<-SendMsgReturn
	<-RecvMsgReturn
}

func TestClientStreamReturn(t *testing.T) {
	const (
		invalidCompressType = -1
		dataLen             = 1024
	)

	svrOpts := []server.Option{
		server.WithAddress("127.0.0.1:30211"),
		server.WithCurrentCompressType(invalidCompressType),
	}
	handle := func(s server.Stream) error {
		req := getBytes(dataLen)
		s.RecvMsg(req)
		rsp := req
		s.SendMsg(rsp)
		return errs.NewFrameError(101, "expected error")
	}
	svr := startStreamServer(handle, svrOpts)
	defer closeStreamServer(svr)

	cliOpts := []client.Option{
		client.WithTarget("ip://127.0.0.1:30211"),
		client.WithCompressType(invalidCompressType),
	}

	clientStream, err := getClientStream(context.Background(), clientDesc, cliOpts)
	assert.Nil(t, err)
	err = clientStream.SendMsg(getBytes(dataLen))
	assert.Nil(t, err)

	rsp := getBytes(dataLen)
	err = clientStream.RecvMsg(rsp)
	assert.EqualValues(t, int32(101), err.(*errs.Error).Code)
}

// TestClientSendFailWhenServerUnavailable test when the client blocks
// on SendMsg because of flow control, if the server is closed, the client
// SendMsg should return.
func TestClientSendFailWhenServerUnavailable(t *testing.T) {
	codec.Register("mock", nil, &fakeCodec{})
	tp := &fakeTransport{expectChan: make(chan recvExpect, 1)}
	tp.expectChan <- func(fh *trpc.FrameHead, m codec.Msg) ([]byte, error) {
		return nil, nil
	}
	cs, err := stream.NewStreamClient().NewStream(ctx, &client.ClientStreamDesc{}, "",
		client.WithProtocol("mock"),
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(tp),
	)
	assert.Nil(t, err)
	assert.NotNil(t, cs)
	assert.Nil(t, cs.SendMsg(getBytes(65535)))
	tp.expectChan <- func(fh *trpc.FrameHead, m codec.Msg) ([]byte, error) {
		return nil, errors.New("server is closed")
	}
	assert.Eventually(
		t,
		func() bool {
			cs.SendMsg(getBytes(65535))
			return true
		},
		time.Second,
		100*time.Millisecond,
	)
}

// TestClientReceiveErrorWhenServerUnavailable tests that the client receives a non-io.EOF
// error when the server crashes or the connection is closed, otherwise the client would
// mistakenly think that the server closed the stream normally.
func TestClientReceiveErrorWhenServerUnavailable(t *testing.T) {
	codec.Register("mock", nil, &fakeCodec{})
	tp := &fakeTransport{expectChan: make(chan recvExpect, 1)}
	tp.expectChan <- func(fh *trpc.FrameHead, m codec.Msg) ([]byte, error) {
		return nil, nil
	}
	cs, err := stream.NewStreamClient().NewStream(ctx, &client.ClientStreamDesc{}, "",
		client.WithProtocol("mock"),
		client.WithTarget("ip://127.0.0.1:8000"),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithStreamTransport(tp),
	)
	assert.Nil(t, err)
	assert.NotNil(t, cs)
	tp.expectChan <- func(fh *trpc.FrameHead, m codec.Msg) ([]byte, error) {
		return nil, io.EOF
	}
	err = cs.RecvMsg(nil)
	assert.NotEqual(t, io.EOF, err)
	assert.ErrorIs(t, err, io.EOF)
}
