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

package http

import (
	"context"
	"strings"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"github.com/valyala/fasthttp"
)

// FastHTTPCli is the struct for invoking service based on http.
type FastHTTPCli struct {
	serviceName string
	client      client.Client
	opts        []client.Option
}

// NewFastHTTPClientProxy creates a new fasthttp backend request proxy.
// Parameter name means the name of backend http service (e.g. trpc.http.xxx.xxx),
// mainly used for metrics, can be freely defined but
// format needs to follow "trpc.app.server.service".
var NewFastHTTPClientProxy = func(name string, opts ...client.Option) *FastHTTPCli {
	c := &FastHTTPCli{
		serviceName: name,
		client:      client.DefaultClient,
	}
	c.opts = make([]client.Option, 0, len(opts)+1)
	c.opts = append(c.opts, client.WithProtocol(protocol.FastHTTP))
	c.opts = append(c.opts, opts...)
	return c
}

// NewFastHTTPClient returns fasthttp.Client of the go sdk, which is convenient
// for third-party clients to use, and can report monitoring metrics.
// After returning, user can configure the fasthttp.Client.
// User can configure the fasthttp.HostClient by modifying field `ConfigureClient`
// It is recommended to use fasthttp.Client.Do(req, rsp) for making requests.
// Notice: name should NOT be "".
func NewFastHTTPClient(name string, opts ...client.Option) *fasthttp.Client {
	c := &FastHTTPCli{
		serviceName: name,
		client:      client.DefaultClient,
	}
	c.opts = make([]client.Option, 0, len(opts)+2)
	c.opts = append(c.opts, client.WithProtocol(protocol.FastHTTP))
	c.opts = append(c.opts, opts...)
	// Use passthrough selector to bypass the naming process,
	// as the framework will ignore the result of naming.
	// Ensure it takes effect by placing it afterwards.
	c.opts = append(c.opts, client.WithTarget("passthrough://"+name))
	return &fasthttp.Client{
		ConfigureClient: func(hc *fasthttp.HostClient) error {
			hc.Transport = c
			return nil
		},
	}
}

// RoundTrip implements the fasthttp.RoundTripper interface for fastHTTPCli.
// Notice: Calls through FastHTTPClientProxy do NOT go through RoundTrip.
// Currently, retries are always returned false in RoundTrip.
func (c *FastHTTPCli) RoundTrip(
	hc *fasthttp.HostClient,
	req *fasthttp.Request,
	rsp *fasthttp.Response,
) (retry bool, err error) {
	ctx, msg := codec.WithCloneMessage(context.Background())
	defer codec.PutBackMessage(msg)

	c.setDefaultCallOption(msg, string(req.URI().Path()))
	// Align with thttp.
	msg.WithClientReqHead(&FastHTTPClientReqHeader{
		Scheme:  string(req.URI().Scheme()),
		Method:  string(req.Header.Method()),
		Host:    string(req.Host()),
		Request: req,
	})
	msg.WithClientRspHead(&FastHTTPClientRspHeader{Response: rsp})

	if err := c.client.Invoke(ctx, nil, nil, c.opts...); err != nil {
		// If the error is caused by the status code, ignore it and return the response normally.
		if rsp != nil && rsp.StatusCode() == errs.Code(err) {
			return false, nil
		}
		return false, err
	}
	return false, nil
}

// setDefaultCallOption sets default call option.
func (c *FastHTTPCli) setDefaultCallOption(msg codec.Msg, path string) {
	msg.WithClientRPCName(path)
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeJSON)

	// Callee method is mainly for metrics.
	// User can copy this part of code and modify it yourself to meet special requirements.
	if s := strings.Split(path, "?"); len(s) > 0 {
		msg.WithCalleeMethod(s[0])
	}
}

// Get uses trpc client to send fasthttp GET request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/get_xxx?k1=v1&k2=v2.
// Param reqBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// msg.WithClientReqHead will be called within this method to ensure that Method is GET.
func (c *FastHTTPCli) Get(ctx context.Context, path string, rspBody interface{}, opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, path)
	msg.WithClientReqHead(&FastHTTPClientReqHeader{Method: fasthttp.MethodGet})
	return c.send(ctx, nil, rspBody, opts...)
}

// Post uses trpc client to send fasthttp POST request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/add_xxx.
// Param rspBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// msg.WithClientReqHead will be called within this method to ensure that Method is POST.
func (c *FastHTTPCli) Post(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, path)
	msg.WithClientReqHead(&FastHTTPClientReqHeader{Method: fasthttp.MethodPost})
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Put uses trpc client to send fasthttp PUT request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/update_xxx.
// Param rspBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// msg.WithClientReqHead will be called within this method to ensure that Method is PUT.
func (c *FastHTTPCli) Put(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, path)
	msg.WithClientReqHead(&FastHTTPClientReqHeader{Method: fasthttp.MethodPut})
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Patch uses trpc client to send fasthttp PATCH request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/update_xxx.
// Param rspBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// msg.WithClientReqHead will be called within this method to ensure that Method is PATCH.
func (c *FastHTTPCli) Patch(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, path)
	msg.WithClientReqHead(&FastHTTPClientReqHeader{Method: fasthttp.MethodPatch})
	return c.send(ctx, reqBody, rspBody, opts...)
}

// Delete uses trpc client to send fasthttp DELETE request.
// Param path represents the url segments that follow domain, e.g. /cgi-bin/delete_xxx.
// Param reqBody and rspBody are passed in with specific type,
// corresponding serialization should be specified, or json by default.
// msg.WithClientReqHead will be called within this method to ensure that Method is DELETE.
//
// Delete may have body, if it is empty, set reqBody and rspBody with nil.
func (c *FastHTTPCli) Delete(ctx context.Context, path string, reqBody interface{}, rspBody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	c.setDefaultCallOption(msg, path)
	msg.WithClientReqHead(&FastHTTPClientReqHeader{Method: fasthttp.MethodDelete})
	return c.send(ctx, reqBody, rspBody, opts...)
}

// send uses client (protocol: fasthttp) to send fasthttp request.
func (c *FastHTTPCli) send(ctx context.Context, reqBody, rspBody interface{}, opts ...client.Option) error {
	options := make([]client.Option, 0, len(c.opts)+len(opts))
	options = append(options, c.opts...)
	options = append(options, opts...)
	return c.client.Invoke(ctx, reqBody, rspBody, options...)
}
