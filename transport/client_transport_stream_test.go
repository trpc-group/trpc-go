// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package transport_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/transport"

	_ "trpc.group/trpc-go/trpc-go"
)

// TestClientStreamNetworkError test client decode error.
func TestClientStreamNetworkError(t *testing.T) {
	st := transport.NewServerStreamTransport()
	svrCtx, close := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer close()
	go func() {
		err := st.ListenAndServe(svrCtx,
			transport.WithListenNetwork("tcp"),
			transport.WithListenAddress(":12017"),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&multiplexedFramerBuilder{}),
		)
		require.Nil(t, err)
	}()

	roundTripOpts := []transport.RoundTripOption{
		transport.WithDialNetwork("tcp"),
		transport.WithDialAddress(":12017"),
	}

	time.Sleep(20 * time.Millisecond)
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	require.Nil(t, err)

	lenData := make([]byte, 4)
	binary.BigEndian.PutUint32(lenData, uint32(len(data)))

	headData := make([]byte, 8)
	binary.BigEndian.PutUint32(headData[:4], defaultStreamID)
	binary.BigEndian.PutUint32(headData[4:8], uint32(len(data)))

	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithStreamID(100)

	// test IO EOF.
	ct := transport.NewClientStreamTransport()
	fb := &multiplexedFramerBuilder{}
	fb.SetError(io.EOF)
	roundTripOpts = append(roundTripOpts, transport.WithClientFramerBuilder(fb), transport.WithMsg(msg))
	err = ct.Init(ctx, roundTripOpts...)
	assert.Nil(t, err)
	rsp, err := ct.Recv(ctx)
	assert.NotNil(t, err)
	assert.Nil(t, rsp)

	// test ctx canceled.
	msg.WithStreamID(101)
	fb = &multiplexedFramerBuilder{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cancel()
	roundTripOpts = append(roundTripOpts, transport.WithClientFramerBuilder(fb), transport.WithMsg(msg))
	err = ct.Init(ctx, roundTripOpts...)
	assert.NotNil(t, err)

	// test ctx timeout.
	msg.WithStreamID(102)
	fb = &multiplexedFramerBuilder{}
	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	<-ctx.Done()
	roundTripOpts = append(roundTripOpts, transport.WithClientFramerBuilder(fb), transport.WithMsg(msg))
	err = ct.Init(ctx, roundTripOpts...)
	assert.NotNil(t, err)
}

func TestConcurrent(t *testing.T) {
	st := transport.NewServerStreamTransport()
	serverFinish := make(chan int)
	go func() {
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp,udp"),
			transport.WithListenAddress(":12015"),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&multiplexedFramerBuilder{}),
		)
		require.Nil(t, err)
		serverFinish <- 1
	}()
	<-serverFinish

	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}
	data, err := json.Marshal(req)
	require.Nil(t, err)
	headData := make([]byte, 8) // head = streamID + data length
	binary.BigEndian.PutUint32(headData[4:8], uint32(len(data)))
	reqData := append(headData, data...)

	ct := transport.NewClientStreamTransport(transport.WithMaxConcurrentStreams(20))

	// close stream send and receive.
	var wg sync.WaitGroup
	var index uint32
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			ctx := context.Background()
			ctx, msg := codec.WithNewMessage(ctx)
			newIndex := atomic.AddUint32(&index, 1)
			streamID := defaultStreamID + newIndex
			msg.WithStreamID(streamID)
			msg.WithRequestID(streamID)

			err = ct.Init(ctx, transport.WithDialNetwork("tcp"), transport.WithDialAddress(":12015"),
				transport.WithClientFramerBuilder(&multiplexedFramerBuilder{}),
				transport.WithMsg(msg))
			assert.Nil(t, err)

			copyData := make([]byte, len(reqData))
			copy(copyData, reqData)
			binary.BigEndian.PutUint32(copyData, streamID)

			err = ct.Send(ctx, copyData)
			assert.Nil(t, err)
			rspData, err := ct.Recv(ctx)
			assert.Nil(t, err)
			assert.Equal(t, copyData, rspData)
			ct.Close(ctx)
			wg.Done()
		}()
		if i%50 == 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	wg.Wait()
}

/* --------------------------------------------------- mock multiplexed framer ---------------------------------------------- */

type multiplexedFramerBuilder struct {
	errSet bool
	err    error
	safe   bool
}

func (fb *multiplexedFramerBuilder) SetError(err error) {
	fb.errSet = true
	fb.err = err
}

func (fb *multiplexedFramerBuilder) ClearError() {
	fb.errSet = false
	fb.err = nil
}

func (fb *multiplexedFramerBuilder) New(r io.Reader) codec.Framer {
	return &multiplexedFramer{r: r, fb: fb}
}

func (fb *multiplexedFramerBuilder) Parse(r io.Reader) (vid uint32, buf []byte, err error) {
	buf, err = fb.New(r).ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	return binary.BigEndian.Uint32(buf[:4]), buf, nil
}

type multiplexedFramer struct {
	fb *multiplexedFramerBuilder
	r  io.Reader
}

func (f *multiplexedFramer) ReadFrame() ([]byte, error) {
	if f.fb.errSet {
		return nil, f.fb.err
	}
	var headData [8]byte

	_, err := io.ReadFull(f.r, headData[:])
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(headData[4:])

	msg := make([]byte, len(headData)+int(length))
	copy(msg, headData[:])

	_, err = io.ReadFull(f.r, msg[len(headData):])
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (f *multiplexedFramer) IsSafe() bool {
	return f.fb.safe
}
