package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
)

func TestValueDetachedTransport(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer ln.Close()
	s := &http.Server{Addr: ln.Addr().String(), Handler: &handler{}}
	serveReturn := make(chan error)
	go func() {
		serveReturn <- s.Serve(ln)
	}()
	clientTransport := NewClientTransport(false)
	httpClientTransport := clientTransport.(*ClientTransport)
	ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), time.Second*10)
	defer cancel()

	type contextType struct{}
	ctx = context.WithValue(ctx, contextType{}, struct{}{})

	// Detects whether the data has been unloaded in the native RoundTripper.
	vdt := httpClientTransport.Client.Transport.(*valueDetachedTransport)
	vdt.RoundTripper = &testTransport{
		RoundTripper: vdt.RoundTripper,
		assertFunc: func(request *http.Request) {
			ctx := request.Context()
			// Cannot get value.
			if ctx.Value(contextType{}) != nil {
				t.Fatal("valueDetachedTransport not detach the value")
			}
			if httptrace.ContextClientTrace(ctx) == nil {
				t.Fatal("valueDetachedTransport not transmit the httptrace")
			}
		},
	}

	// Check if data is still retained in http.ClientTransport.
	// Check if httpClientTransport is type of http.ClientTransport or not.
	// Check if httpClientTransport.Client.Transport is type of *testTransport and can get data.
	// Check if httpClientTransport.Client.Transport.RoundTripper is type of *valueDetachedTransport and detached data.
	// Check if httpClientTransport.Client.Transport.RoundTripper.RoundTripper is type of *testTransport and cannot get value.
	// Check if httpClientTransport.Client.Transport.RoundTripper.RoundTripper.RoundTripper is type of *http.Transport,
	// which is equals to StdHTTPTransport.
	httpClientTransport.Client.Transport = &testTransport{
		RoundTripper: httpClientTransport.Client.Transport,
		assertFunc: func(request *http.Request) {
			t.Log(fmt.Sprintf("%+v", request))
			ctx := request.Context()
			// Can get data.
			if ctx.Value(contextType{}) == nil {
				t.Fatal("ClientTransport detach the value")
			}
			if httptrace.ContextClientTrace(ctx) == nil {
				t.Fatal("valueDetachedTransport not transmit the httptrace")
			}
		},
	}

	target := "ip://" + ln.Addr().String()
	c := NewClientProxy("", client.WithTarget(target), client.WithTransport(clientTransport))
	rsp := &codec.Body{}
	require.Eventually(t,
		func() bool {
			err = c.Get(ctx, "/", rsp,
				client.WithCurrentSerializationType(codec.SerializationTypeNoop),
				client.WithTimeout(10*time.Second))
			return err == nil
		}, time.Second, 10*time.Millisecond,
		"get %s return failure %v", target, err)
	require.Nil(t, s.Shutdown(ctx))
	<-serveReturn
}

type handler struct{}

func (h *handler) ServeHTTP(http.ResponseWriter, *http.Request) {
	return
}

type testTransport struct {
	http.RoundTripper
	assertFunc func(request *http.Request)
}

// RoundTrip implements http.RoundTripper.
func (rt *testTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt.assertFunc(request)
	response, err := rt.RoundTripper.RoundTrip(request)
	return response, err
}
