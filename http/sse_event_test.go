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

package http

import (
	"bytes"
	"context"
	"errors"
	stdhttp "net/http"
	"testing"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

type sseHandler func(*sse.Event) error

func (h sseHandler) Handle(e *sse.Event) error {
	return h(e)
}

type rspHandler func(*stdhttp.Response) error

func (h rspHandler) Handle(r *stdhttp.Response) error {
	return h(r)
}

func TestProcessEvent(t *testing.T) {
	raw := []byte(":comment\nid: 1\nevent: message\nretry: 1000\ndata: hello\ndata: world\n\n")
	event, err := processEvent(raw)
	require.NoError(t, err)
	require.Equal(t, []byte("comment"), event.Comment)
	require.Equal(t, []byte("1"), event.ID)
	require.Equal(t, []byte("message"), event.Event)
	require.Equal(t, []byte("1000"), event.Retry)
	require.Equal(t, []byte("hello\nworld"), event.Data)
}

func TestClientCodecEncodeFillSSEHeaders(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	reqHead := &ClientReqHeader{}
	rspHead := &ClientRspHeader{SSEHandler: sseHandler(func(*sse.Event) error { return nil })}
	msg.WithClientReqHead(reqHead)
	msg.WithClientRspHead(rspHead)

	_, err := DefaultClientCodec.Encode(msg, nil)
	require.NoError(t, err)
	require.Equal(t, "text/event-stream", reqHead.Header.Get("Accept"))
	require.Equal(t, "keep-alive", reqHead.Header.Get(Connection))
	require.Equal(t, "no-cache", reqHead.Header.Get("Cache-Control"))
}

func TestClientCodecDecodeSSE(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	var data []byte
	msg.WithClientRspHead(&ClientRspHeader{
		Response: newHTTPResponse("event: message\ndata: hello\n\n"),
		SSEHandler: sseHandler(func(e *sse.Event) error {
			data = append(data, e.Data...)
			return nil
		}),
	})

	body, err := DefaultClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Nil(t, body)
	require.Equal(t, []byte("hello"), data)
}

func TestClientCodecDecodeSSEBusinessError(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithClientRspHead(&ClientRspHeader{
		Response: newHTTPResponse("event: message\ndata: hello\n\n"),
		SSEHandler: sseHandler(func(*sse.Event) error {
			return errs.New(6028, "business validation failed")
		}),
	})

	body, err := DefaultClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Nil(t, body)
	var e *errs.Error
	require.True(t, errors.As(msg.ClientRspErr(), &e))
	require.Equal(t, int32(6028), int32(e.Code))
	require.Equal(t, errs.ErrorTypeBusiness, e.Type)
}

func TestClientCodecDecodeResponseHandler(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	var data []byte
	msg.WithClientRspHead(&ClientRspHeader{
		Response:     newHTTPResponse("hello"),
		SSECondition: func(*stdhttp.Response) bool { return false },
		ResponseHandler: rspHandler(func(r *stdhttp.Response) error {
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(r.Body)
			data = buf.Bytes()
			return err
		}),
	})

	body, err := DefaultClientCodec.Decode(msg, nil)
	require.NoError(t, err)
	require.Nil(t, body)
	require.Equal(t, []byte("hello"), data)
}

func newHTTPResponse(body string) *stdhttp.Response {
	return &stdhttp.Response{
		StatusCode: stdhttp.StatusOK,
		Header:     stdhttp.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       ioNopCloser{bytes.NewBufferString(body)},
	}
}

type ioNopCloser struct {
	*bytes.Buffer
}

func (c ioNopCloser) Close() error {
	return nil
}
