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

package http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/require"
)

const (
	network = "tcp"
	address = "127.0.0.1:0"
)

func TestHTTPSendAndReceiveSSE(t *testing.T) {
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
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(sseHandlerFunc))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	t.Run("automatically", func(t *testing.T) {
		reqHeader := &thttp.ClientReqHeader{
			Method: http.MethodPost,
		}
		var data []byte
		rspHead := &thttp.ClientRspHeader{
			ManualReadBody: false,
			SSECondition:   nil,
			SSEHandler: sseHandler(func(e *sse.Event) error {
				t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
				if string(e.Event) == "message" {
					data = append(data, e.Data...)
				}
				return nil
			}),
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
				client.WithTimeout(time.Minute),
			))
		require.Equal(t, "hello0hello1hello2", string(data))
	})

	t.Run("manually", func(t *testing.T) {
		reqHeader := &thttp.ClientReqHeader{
			Method: http.MethodPost,
		}
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
				client.WithTimeout(time.Minute),
			))

		body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
		defer body.Close()            // Do remember to close the body.
		// Note that the following code disobeys the SSE protocol, which is simply splitting the lines with '\n'
		// and discarding the "data:" prefix. Since the manual process is too troublesome, we do not recommend this.
		buf := make([]byte, 1024)
		var data strings.Builder
		for {
			n, err := body.Read(buf)
			if err == io.EOF {
				break
			}
			require.Nil(t, err)
			lines := bytes.Split(buf[:n], []byte("\n"))
			for _, line := range lines {
				if !bytes.HasPrefix(line, []byte("data:")) {
					continue
				}
				fromIndex := len("data:")
				if line[fromIndex] == ' ' {
					fromIndex++ // Ignore the optional space after the data: prefix.
				}
				data.Write(line[fromIndex:])
			}
		}

		require.Equal(t, "hello0hello1hello2", data.String())
	})
}

// sseHandler is a handler that handles sse events.
// It sends responses with the header of "Content-Type: text/event-stream".
func sseHandlerFunc(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg := string(bs)
	// Send sse message.
	for i := 0; i < 3; i++ {
		e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
		if err := thttp.WriteSSE(w, e); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		flusher.Flush()
		time.Sleep(500 * time.Millisecond)
	}
}

// normalHandler is a handler that handles normal responses.
// It sends responses with the header of "Content-Type: text/plain".
func normalHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := string(bs)
	var data []byte
	for i := 0; i < 3; i++ {
		data = append(data, []byte(msg+strconv.Itoa(i))...)
	}
	_, _ = w.Write(data)
}

func TestHTTPSendAndReceiveLongSSE(t *testing.T) {
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
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(longSSEHandlerFunc))
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}
	var data []byte
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: false,
		SSECondition:   nil,
		SSEHandler: sseHandler(func(e *sse.Event) error {
			t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
			if string(e.Event) == "message" {
				data = append(data, e.Data...)
			}
			return nil
		}),
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
			client.WithTimeout(time.Minute),
		))
	var expected strings.Builder
	for i := 0; i < 3; i++ {
		expected.WriteString("hello")
		expected.WriteString(strings.Repeat(strconv.Itoa(i), 4096))
	}
	require.Equal(t, expected.String(), string(data))
}

// longSSEHandlerFunc is a handler that handles long SSE responses.
func longSSEHandlerFunc(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set(thttp.Connection, "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg := string(bs)
	// Send sse message.
	for i := 0; i < 3; i++ {
		// The data is a long string, which is larger than 4096 bytes.
		e := sse.Event{Event: []byte("message"), Data: []byte(msg + strings.Repeat(strconv.Itoa(i), 4096))}
		if err := thttp.WriteSSE(w, e); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		flusher.Flush()
		time.Sleep(500 * time.Millisecond)
	}
}

type sseHandler func(*sse.Event) error

// Handle handles sse event, if the returned error is non-nil,
// the framework will abort the reading of the HTTP connection.
func (h sseHandler) Handle(e *sse.Event) error {
	return h(e)
}

type rspHandler func(*http.Response) error

// Handle handles common HTTP response.
func (h rspHandler) Handle(r *http.Response) error {
	return h(r)
}

func TestHTTPSendAndReceiveSSEAndNormalResponse(t *testing.T) {
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
	isSSE := true // Whether to send an SSE event, the first time is true.
	thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Switch between SSE and normal response.
		defer func() { isSSE = !isSSE }()
		if isSSE {
			sseHandlerFunc(w, r)
			return
		}
		normalHandlerFunc(w, r)
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

	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}

	var data []byte
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: false,
		SSECondition: func(r *http.Response) bool {
			return r.Header.Get("Content-Type") == "text/event-stream"
		},
		ResponseHandler: rspHandler(func(r *http.Response) error {
			bs, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			t.Logf("Receive http response: %s", string(bs))
			data = append(data, bs...)
			return nil
		}),
		SSEHandler: sseHandler(func(e *sse.Event) error {
			t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
			if string(e.Event) == "message" {
				data = append(data, e.Data...)
			}
			return nil
		}),
	}

	req := &codec.Body{Data: []byte("hello")}
	rsp := &codec.Body{}
	// The first time we send a request, the response is an SSE event, and the second is a normal response.
	// It is to say, the handler will switch between SSE and normal response, but the response data are the same.
	for i := 0; i < 4; i++ {
		t.Run(fmt.Sprintf("request "+strconv.Itoa(i)), func(t *testing.T) {
			data = []byte{} // Clear the data.
			require.Nil(t,
				c.Post(context.Background(), pattern, req, rsp,
					client.WithCurrentSerializationType(codec.SerializationTypeNoop),
					client.WithSerializationType(codec.SerializationTypeNoop),
					client.WithCurrentCompressType(codec.CompressTypeNoop),
					client.WithReqHead(reqHeader),
					client.WithRspHead(rspHead),
					client.WithTimeout(time.Minute),
				))
			require.Equal(t, "hello0hello1hello2", string(data))
		})
	}
}

func TestHTTPSendAndReceiveSSEWithR3Lab(t *testing.T) {
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

	svr := sse.New()
	mux := http.NewServeMux()
	mux.Handle(pattern, svr)
	thttp.RegisterNoProtocolServiceMux(service, mux)
	svr.CreateStream("test")

	for i := 0; i < 3; i++ {
		event := &sse.Event{
			ID:    []byte(fmt.Sprintf("%d", i)),
			Event: []byte("message"),
			Data:  []byte(fmt.Sprintf("This is message %d", i)),
		}
		svr.Publish("test", event)
	}

	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := sse.NewClient(fmt.Sprintf("http://%s%s", ln.Addr().String(), pattern))

	events := make(chan *sse.Event)
	go func() {
		err = c.Subscribe("test", func(msg *sse.Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	// Wait for the subscription to succeed.
	time.Sleep(200 * time.Millisecond)
	require.Nil(t, err)

	for i := 0; i < 3; i++ {
		msg, err := wait(events, 500*time.Millisecond)
		require.Nil(t, err)
		require.Equal(t, []byte(fmt.Sprintf("This is message %d", i)), msg)
	}
}

// wait waits for the sse event and read data into msg. If timeout, return error.
func wait(ch chan *sse.Event, duration time.Duration) ([]byte, error) {
	var err error
	var msg []byte

	select {
	case event := <-ch:
		msg = event.Data
	case <-time.After(duration):
		err = errors.New("timeout")
	}
	return msg, err
}
