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
	"context"
	"net/http"
	"strings"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

// Client provides the HTTP client interface.
// The primary use of this interface is to request standard HTTP services,
// if you wish to request HTTP RPC services use the client provided by the
// stub code (simply specify the protocol as "http").
type Client interface {
	Get(ctx context.Context, path string, rspBody interface{}, opts ...client.Option) error
	Post(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
	Put(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
	Patch(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
	Delete(ctx context.Context, path string, reqBody interface{}, rspBody interface{}, opts ...client.Option) error
}

// cli is the struct of backend request.
type cli struct {
	serviceName string
	client      client.Client
	opts        []client.Option
}

// NewClientProxy creates a new http backend request proxy.
// Parameter name means the name of backend http service (e.g. trpc.http.xxx.xxx),
// mainly used for metrics, can be freely defined but
// format needs to follow "trpc.app.server.service".
var NewClientProxy = func(name string, opts ...client.Option) Client {
	c := &cli{
		serviceName: name,
		client:      client.DefaultClient,
	}
	c.opts = make([]client.Option, 0, len(opts)+1)
	c.opts = append(c.opts, client.WithProtocol("http"))
	c.opts = append(c.opts, opts...)
	return c
}

// NewStdHTTPClient returns http.Client of the go sdk, which is convenient for
// third-party clients to use, and can report monitoring metrics.
func NewStdHTTPClient(name string, opts ...client.Option) *http.Client {
	c := &cli{
		serviceName: name,
		client:      client.DefaultClient,
	}
	c.opts = make([]client.Option, 0, len(opts)+1)
	c.opts = append(c.opts, client.WithProtocol("http"))
	c.opts = append(c.opts, opts...)
	return &http.Client{Transport: c}
}

// RoundTrip implements the http.RoundTripper interface of http.Client of go sdk.
func (c *cli) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx, msg := codec.WithCloneMessage(request.Context())
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, request.Method, request.URL.Path)

	header := &ClientReqHeader{
		Schema:  request.URL.Scheme,
		Method:  request.Method,
		Host:    request.URL.Host,
		Request: request,
		Header:  request.Header,
	}

	opts := append([]client.Option{
		client.WithReqHead(header),
		client.WithCurrentCompressType(0),       // no compression
		client.WithCurrentSerializationType(-1), // no serialization
	}, c.opts...)

	err := c.client.Invoke(ctx, nil, nil, opts...)
	var rsp *http.Response
	if h, ok := msg.ClientRspHead().(*ClientRspHeader); ok && h.Response != nil {
		rsp = h.Response
	}

	if err != nil {
		// If the error is caused by the status code, ignore it and return the response normally.
		if rsp != nil && rsp.StatusCode == int(errs.Code(err)) {
			return rsp, nil
		}
		return nil, err
	}
	return rsp, nil
}

// Post uses trpc client to send http POST request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/add_xxx
// Param rspBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// client.WithClientReqHead will be called within this method to ensure that httpMethod is POST.
func (c *cli) Post(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, http.MethodPost, path)
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Put uses trpc client to send http PUT request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/update_xxx
// Param rspBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// client.WithClientReqHead will be called within this method to ensure that httpMethod is PUT.
func (c *cli) Put(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, http.MethodPut, path)
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Patch uses trpc client to send http PATCH request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/update_xxx
// Param rspBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// client.WithClientReqHead will be called within this method to ensure that httpMethod is PATCH.
func (c *cli) Patch(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, http.MethodPatch, path)
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Delete uses trpc client to send http DELETE request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/delete_xxx
// Param reqBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// client.WithClientReqHead will be called within this method to ensure that httpMethod is DELETE.
//
// Delete may have body, if it is empty, set reqBody and rspBody with nil.
func (c *cli) Delete(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, http.MethodDelete, path)
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Get uses trpc client to send http GET request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/get_xxx?k1=v1&k2=v2
// Param reqBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// client.WithClientReqHead will be called within this method to ensure that httpMethod is GET.
func (c *cli) Get(ctx context.Context, path string, rspBody interface{}, opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, http.MethodGet, path)
	return c.send(ctx, nil, rspBody, opts...)
}

// send uses trpc client to send http request.
func (c *cli) send(ctx context.Context, reqBody, rspBody interface{}, opts ...client.Option) error {
	return c.client.Invoke(ctx, reqBody, rspBody, append(c.opts, opts...)...)
}

// setDefaultCallOption sets default call option.
func (c *cli) setDefaultCallOption(msg codec.Msg, method, path string) {
	msg.WithClientRPCName(path)
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeJSON)

	// Use ClientReqHeader to specify HTTP method.
	msg.WithClientReqHead(&ClientReqHeader{
		Method: method,
	})
	msg.WithClientRspHead(&ClientRspHeader{})

	// Callee method is mainly for metrics.
	// If you have special requirements, you can copy this part of code and modify it yourself.
	if s := strings.Split(path, "?"); len(s) > 0 {
		msg.WithCalleeMethod(s[0])
	}
}
