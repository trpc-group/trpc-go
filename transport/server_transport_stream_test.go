//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
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
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
)

// TestStreamTCPListenAndServe tests listen and send.
func TestStreamTCPListenAndServe(t *testing.T) {
	st := transport.NewServerStreamTransport()
	go func() {
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp"),
			transport.WithListenAddress(":12013"),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&multiplexedFramerBuilder{}),
		)
		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
		}
	}()

	ctx, f := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer f()
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	headData := make([]byte, 8)
	binary.BigEndian.PutUint32(headData[:4], defaultStreamID)
	binary.BigEndian.PutUint32(headData[4:8], uint32(len(data)))
	reqData := append(headData, data...)

	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithStreamID(defaultStreamID)

	time.Sleep(time.Millisecond * 20)
	ct := transport.NewClientStreamTransport()
	err = ct.Init(ctx, transport.WithDialNetwork("tcp"), transport.WithDialAddress(":12013"),
		transport.WithClientFramerBuilder(&multiplexedFramerBuilder{}),
		transport.WithMsg(msg))
	assert.Nil(t, err)

	err = ct.Send(ctx, reqData)
	assert.Nil(t, err)
	err = st.Send(ctx, reqData)
	assert.NotNil(t, err)

	rsp, err := ct.Recv(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, rsp)
	ct.Close(ctx)
	err = ct.Send(ctx, reqData)
	assert.NotNil(t, err)

}

// TestStreamTCPListenAndServeFail tests listen and send failures.
func TestStreamTCPListenAndServeFail(t *testing.T) {
	st := transport.NewServerStreamTransport()
	go func() {
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp"),
			transport.WithListenAddress(":12014"),
			transport.WithHandler(&echoHandler{}),
			transport.WithServerFramerBuilder(&multiplexedFramerBuilder{}),
		)
		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
		}
	}()

	ctx, f := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer f()
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	headData := make([]byte, 8)
	binary.BigEndian.PutUint32(headData[:4], defaultStreamID)
	binary.BigEndian.PutUint32(headData[4:8], uint32(len(data)))
	reqData := append(headData, data...)

	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithStreamID(defaultStreamID)

	time.Sleep(time.Millisecond * 20)
	ct := transport.NewClientStreamTransport()
	err = ct.Init(ctx, transport.WithDialNetwork("tcp"), transport.WithDialAddress(":12015"),
		transport.WithClientFramerBuilder(&multiplexedFramerBuilder{}))
	assert.NotNil(t, err)
	err = ct.Send(ctx, reqData)
	assert.NotNil(t, err)
	_, err = ct.Recv(ctx)
	assert.NotNil(t, err)
	ct.Close(ctx)

	// Test opts pool is nil.
	err = ct.Init(ctx, transport.WithDialPool(nil))
	assert.NotNil(t, err)

	// Test frame builder is nil.
	err = ct.Init(ctx)
	assert.NotNil(t, err)

	// test context.
	ct = transport.NewClientStreamTransport()
	err = ct.Init(ctx, transport.WithDialNetwork("tcp"), transport.WithDialAddress(":12014"),
		transport.WithClientFramerBuilder(&multiplexedFramerBuilder{}))
	assert.NotNil(t, err)

	ctx = context.Background()
	ctx, msg = codec.WithNewMessage(ctx)
	msg.WithStreamID(defaultStreamID)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err = ct.Recv(ctx)
	// type:framework, code:161, msg:tcp client transport canceled before Write: context canceled
	assert.NotNil(t, err)

	ctx = context.Background()
	ctx, msg = codec.WithNewMessage(ctx)
	msg.WithStreamID(defaultStreamID)
	ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_, err = ct.Recv(ctx)
	// type:framework, code:101, msg:tcp client transport timeout before Write: context deadline exceeded
	assert.NotNil(t, err)

}

// TestStreamTCPListenAndServeSend tests listen and send failures.
func TestStreamTCPListenAndServeSend(t *testing.T) {
	lnAddr := "127.0.0.1:12016"
	st := transport.NewServerStreamTransport()
	go func() {
		err := st.ListenAndServe(context.Background(),
			transport.WithListenNetwork("tcp"),
			transport.WithListenAddress(lnAddr),
			transport.WithHandler(&echoStreamHandler{}),
			transport.WithServerFramerBuilder(&multiplexedFramerBuilder{}),
		)
		if err != nil {
			t.Logf("ListenAndServe fail:%v", err)
		}
	}()
	time.Sleep(20 * time.Millisecond)
	req := &helloRequest{
		Name: "trpc",
		Msg:  "HelloWorld",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json marshal fail:%v", err)
	}
	headData := make([]byte, 8)
	binary.BigEndian.PutUint32(headData[:4], defaultStreamID)
	binary.BigEndian.PutUint32(headData[4:8], uint32(len(data)))
	reqData := append(headData, data...)

	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithStreamID(defaultStreamID)
	fb := &multiplexedFramerBuilder{}

	// Test IO EOF.
	port := getFreeAddr("tcp")
	la := "127.0.0.1" + port
	ct := transport.NewClientStreamTransport()
	err = ct.Init(ctx, transport.WithDialNetwork("tcp"), transport.WithDialAddress(lnAddr),
		transport.WithClientFramerBuilder(fb), transport.WithMsg(msg), transport.WithLocalAddr(la))
	assert.Nil(t, err)
	time.Sleep(100 * time.Millisecond)
	raddr, err := net.ResolveTCPAddr("tcp", la)
	assert.Nil(t, err)
	laddr, err := net.ResolveTCPAddr("tcp", lnAddr)
	assert.Nil(t, err)
	msg.WithRemoteAddr(raddr)
	msg.WithLocalAddr(laddr)
	err = st.Send(ctx, reqData)
	assert.Nil(t, err)
}
