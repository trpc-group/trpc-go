package admin

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	// Given a router
	r := newRouter()

	// And config its handler function with pattern "/index" and Desc "index page"
	r.add("/index", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	})

	// When List current handlers that have already registered from the router
	handlers := r.list()

	// Then the handlers should contain only the single handler function previously registered
	require.Len(t, handlers, 1)
	require.Equal(t, "/index", handlers[0].pattern)

	// When config a testHandler with pattern "/test1", and try to find the new handler from the router
	testHandler := func(w http.ResponseWriter, r *http.Request) {}
	r.add("/test", testHandler)
	var handler *routerHandler
	for _, h := range r.list() {
		if h.pattern == "/test" {
			handler = h
			break
		}
	}

	// Then the handler found should be the testHandler
	require.Equal(t,
		runtime.FuncForPC(reflect.ValueOf(testHandler).Pointer()).Name(),
		runtime.FuncForPC(reflect.ValueOf(handler.handler).Pointer()).Name(),
	)

	// When start a http server with the router, and send a http GET request to access index page
	addr := mustListenAndServe(t, r)
	resp, err := http.Get(fmt.Sprintf("http://%v%s", addr, "/index"))
	require.Nil(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Then response body should be "/index"
	require.Nil(t, err)
	require.Equal(t, "/index", string(body))

	// When send a http GET request to access nonexistent resource
	resp, err = http.Get(fmt.Sprintf("http://%v%s", addr, "/nonexistent-resource"))
	require.Nil(t, err)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()

	// Then the response body should contain 404 error message
	require.Nil(t, err)
	require.Equal(t, "404 page not found\n", string(body))
}

func TestRouter_ServeHTTP(t *testing.T) {
	t.Run("panic but recovered", func(t *testing.T) {
		// Given a router
		r := newRouter()

		// And config its handler function that always panic with pattern "/index"
		const panicMessage = "there must be something wrong with your code"
		r.add("/index", func(w http.ResponseWriter, r *http.Request) {
			panic(panicMessage)
		})

		// When start a http server with the router, and send a http GET request to access index page
		addr := mustListenAndServe(t, r)
		resp, err := http.Get(fmt.Sprintf("http://%v%s", addr, "/index"))
		require.Nil(t, err)
		require.Nil(t, err)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Then response body should contain panic message
		require.Nil(t, err)
		require.Contains(t, string(body), "PANIC : "+panicMessage)
	})
}

func mustListenAndServe(t *testing.T, r *router) net.Addr {
	l, err := net.Listen("tcp", testAddress)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if http.Serve(l, r); err != nil && err != http.ErrServerClosed {
			t.Log(err)
		}
	}()
	time.Sleep(time.Second)
	return l.Addr()
}
