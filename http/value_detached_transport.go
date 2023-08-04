package http

import (
	"net/http"
	"net/http/httptrace"
)

// newValueDetachedTransport creates a new valueDetachedTransport.
func newValueDetachedTransport(r http.RoundTripper) http.RoundTripper {
	return &valueDetachedTransport{RoundTripper: r}
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
