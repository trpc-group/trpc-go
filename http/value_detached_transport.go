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
	"net/http"
	"net/http/httptrace"

	"trpc.group/trpc-go/trpc-go/transport"
)

// newValueDetachedTransport creates a new valueDetachedTransport.
func newValueDetachedTransport(r http.RoundTripper) http.RoundTripper {
	return &valueDetachedTransport{RoundTripper: r}
}

// roundTripperWithOptions configures an http.RoundTripper based on the provided RoundTripOptions.
// If r implements clonableRoundTripper, it'll clone a new instance from r and applies the options to this new instance.
func roundTripperWithOptions(r http.RoundTripper, opts transport.RoundTripOptions) http.RoundTripper {
	if crt, ok := r.(clonableRoundTripper); ok {
		r = crt.clone()
	}
	detachedTransport := r
	if vdt, ok := r.(*valueDetachedTransport); ok {
		detachedTransport = vdt.RoundTripper
	}
	tr, ok := detachedTransport.(*http.Transport)
	if !ok {
		return r
	}
	// Apply HTTP specific options from opts to the transport.
	tr.MaxIdleConns = opts.HTTPOpts.Pool.MaxIdleConns
	tr.MaxIdleConnsPerHost = opts.HTTPOpts.Pool.MaxIdleConnsPerHost
	tr.MaxConnsPerHost = opts.HTTPOpts.Pool.MaxConnsPerHost
	tr.IdleConnTimeout = opts.HTTPOpts.Pool.IdleConnTimeout
	tr.DisableKeepAlives = opts.DisableConnectionPool
	return r
}

// clonableRoundTripper defines an interface for round trippers that can create a clone of themselves.
type clonableRoundTripper interface {
	clone() http.RoundTripper
}

// valueDetachedTransport detaches ctx value before RoundTripping a http.Request.
type valueDetachedTransport struct {
	http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (vdt *valueDetachedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	trace := httptrace.ContextClientTrace(ctx)
	ctx = detachCtxValue(ctx)
	if trace != nil {
		ctx = httptrace.WithClientTrace(ctx, trace)
	}
	req = req.WithContext(ctx)
	return vdt.RoundTripper.RoundTrip(req)
}

// CancelRequest implements canceler.
func (vdt *valueDetachedTransport) CancelRequest(req *http.Request) {
	// canceler judges whether RoundTripper implements
	// the http.RoundTripper.CancelRequest function.
	// CancelRequest is supported after go 1.5 or 1.6.
	type canceler interface{ CancelRequest(*http.Request) }
	if v, ok := vdt.RoundTripper.(canceler); ok {
		v.CancelRequest(req)
	}
}

// clone creates a copy of the valueDetachedTransport.
func (vdt *valueDetachedTransport) clone() http.RoundTripper {
	detachedTransport := vdt.RoundTripper
	// Check if the embedded RoundTripper implements clonableRoundTripper and clone it if possible.
	if crt, ok := detachedTransport.(clonableRoundTripper); ok {
		detachedTransport = crt.clone()
	}
	return newValueDetachedTransport(detachedTransport)
}
