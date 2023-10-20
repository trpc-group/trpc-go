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

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/healthcheck"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	testVersion    = "v0.2.0-alpha"
	testAddress    = "localhost:0"
	testConfigPath = "../testdata/trpc_go.yaml"
)

func newDefaultAdminServer() *Server {
	s := NewServer(
		WithVersion(testVersion),
		WithAddr(testAddress),
		WithTLS(false),
		WithReadTimeout(defaultReadTimeout),
		WithWriteTimeout(defaultWriteTimeout),
		WithConfigPath(testConfigPath),
	)

	s.HandleFunc("/usercmd", userCmd)
	s.HandleFunc("/errout", errOutput)
	s.HandleFunc("/panicHandle", panicHandle)

	return s
}

func mustStartAdminServer(t *testing.T, s *Server) {
	t.Helper()

	go func() {
		if err := s.Serve(); err != nil {
			t.Log(err)
		}
	}()
	time.Sleep(200 * time.Millisecond)
}

func TestRPCZFailed(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	tests := []struct {
		name      string
		url       string
		errorCode int
		message   string
		content   interface{}
	}{
		{
			name:      "handleSpans failed because query parameter isn't a number",
			url:       fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpansList + "?num=xxx",
			errorCode: errCodeServer,
			message:   "must be a integer",
			content:   "",
		},
		{
			name:      "handleSpans failed because query parameter isn't a positive integer",
			url:       fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpansList + "?num=-1",
			errorCode: errCodeServer,
			message:   "must be a non-negative integer",
			content:   nil,
		},
		{
			name:      "handleSpan failed because can't find span_id",
			url:       fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpanGet + "1",
			errorCode: errCodeServer,
			message:   "cannot find span-id",
			content:   nil,
		},
		{
			name:      "handleSpan failed because query parameter span_id is empty",
			url:       fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpanGet + "",
			errorCode: errCodeServer,
			message:   "undefined command",
			content:   nil,
		},
		{
			name:      "handleSpan failed because query parameter span_id is negative",
			url:       fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpanGet + "-1",
			errorCode: errCodeServer,
			message:   "can not be negative",
			content:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := httpRequest(http.MethodGet, tt.url, "")
			require.Nil(t, err)
			require.Contains(t, string(r), tt.message)
		})
	}
	t.Run("url query doesn't match rpcz", func(t *testing.T) {
		r, err := httpRequest(http.MethodGet, fmt.Sprintf("http://%s", s.server.Addr)+"/cmd/rpcz", "")
		require.Nil(t, err)
		require.Contains(t, string(r), "404 page not found")
	})
}

type sliceSpanExporter struct {
	spans []rpcz.ReadOnlySpan
}

func (e *sliceSpanExporter) Export(span *rpcz.ReadOnlySpan) {
	e.spans = append(e.spans, *span)
}

func TestRPC_Exporter(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	oldGlobalRPCZ := rpcz.GlobalRPCZ
	defer func() {
		rpcz.GlobalRPCZ = oldGlobalRPCZ
	}()
	// Given a GlobalRPCZ configured with exporter
	exporter := &sliceSpanExporter{}
	rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{Fraction: 1.0, Capacity: 10, Exporter: exporter})

	// When End a "server" span with spanID.
	span := rpcz.SpanFromContext(context.Background())
	cs, end := span.NewChild("server")
	spanID := cs.ID()
	end.End()

	// Then the exporter contain the span exported by the GlobalRPCZ
	require.Len(t, exporter.spans, 1)
	require.Equal(t, spanID, exporter.spans[0].ID)

	// And the GlobalRPCZ still stores a copy of the exported span
	rRaw, err := httpRequest(http.MethodGet, fmt.Sprintf("http://%s", s.server.Addr)+patternRPCZSpansList+"?num", "")
	require.Nil(t, err)
	require.Contains(t, string(rRaw), fmt.Sprint(spanID))
}

func TestRPCZOk(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	oldGlobalRPCZ := rpcz.GlobalRPCZ
	defer func() {
		rpcz.GlobalRPCZ = oldGlobalRPCZ
	}()
	rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{Fraction: 1.0, Capacity: 10})
	span := rpcz.SpanFromContext(context.Background())

	cs, end := span.NewChild("server")
	spanID := cs.ID()
	end.End()

	tests := []struct {
		name      string
		url       string
		errorCode int
		message   string
		content   interface{}
	}{
		{
			name:    "handleSpans ok query parameter num is empty",
			url:     fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpansList + "?num",
			content: fmt.Sprintf("1:\n  span: (server, %d)\n", spanID),
		},
		{
			name:    "handleSpans ok without any query parameter",
			url:     fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpansList,
			content: fmt.Sprintf("1:\n  span: (server, %d)\n", spanID),
		},
		{
			name:    "handleSpans ok",
			url:     fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpansList + "?num=1",
			content: fmt.Sprintf("1:\n  span: (server, %d)\n", spanID),
		},
		{
			name:    "handleSpan ok",
			url:     fmt.Sprintf("http://%s", s.server.Addr) + patternRPCZSpanGet + fmt.Sprint(spanID),
			content: fmt.Sprintf("span: (server, %d)\n", spanID),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rRaw, err := httpRequest(http.MethodGet, tt.url, "")
			r := string(rRaw)
			require.Nil(t, err)
			require.Contains(t, r, tt.message)
			require.Contains(t, r, tt.content)

		})
	}
}

func TestCmdVersion(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	versionURL := fmt.Sprintf("http://%s", s.server.Addr) + "/version"
	respData, err := httpRequest(http.MethodGet, versionURL, "")
	if err != nil {
		require.Nil(t, err, "httpGetBody failed")
		return
	}

	res := struct {
		Errcode int    `json:"errorcode"`
		Message string `json:"message"`
		Version string `json:"version"`
	}{}
	err = json.Unmarshal(respData, &res)
	require.Nil(t, err, "testAdminServerVersion unmarshal failed")
	require.Equal(t, 0, res.Errcode)
	require.Equal(t, testVersion, res.Version)
}

func TestCmdsLogLevel(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})

	dlogger := log.GetDefaultLogger()

	// Preset test conditions
	log.Register("default", log.NewZapLog([]log.OutputConfig{
		{Writer: log.OutputConsole, Level: "debug"},
		{Writer: log.OutputFile, WriteConfig: log.WriteConfig{Filename: "test"}, Level: "info"},
	}))

	t.Cleanup(func() {
		log.Register("default", dlogger)
	})

	res := struct {
		Errcode  int    `json:"errorcode"`
		Message  string `json:"message"`
		Level    string `json:"level"`
		PreLevel string `json:"prelevel"`
	}{}

	t.Run("right case", func(t *testing.T) {
		logURL := fmt.Sprintf("http://%s", s.server.Addr) + "/cmds/loglevel?logger=default&output=1"
		// TestGet
		respData, err := httpRequest(http.MethodGet, logURL, "")
		require.Nil(t, err, "httpGetBody failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "testAdminServerLogLevel unmarshal failed")
		require.Equal(t, 0, res.Errcode)
		require.Equal(t, "info", res.Level)

		// TestUpdate
		body, err := httpRequest(http.MethodPut, logURL, "value=debug")
		require.Nil(t, err, "httpRequest failed:", err)
		err = json.Unmarshal(body, &res)
		require.Nil(t, err, "Unmarshal failed:", err)
		require.Equal(t, 0, res.Errcode)
		require.Equal(t, "info", res.PreLevel)
		require.Equal(t, "debug", res.Level)
	})
	t.Run("request parameter is empty", func(t *testing.T) {
		logURL := fmt.Sprintf("http://%s", s.server.Addr) + "/cmds/loglevel"
		respData, err := httpRequest(http.MethodGet, logURL, "")
		require.Nil(t, err, "httpGetBody failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "unmarshal failed")
		require.Equal(t, 0, res.Errcode)
		require.Equal(t, "debug", res.Level)
	})
	t.Run("failed to parse request parameters", func(t *testing.T) {
		logURL := fmt.Sprintf("http://%s", s.server.Addr) + "/cmds/loglevel?logger%"
		respData, err := httpRequest(http.MethodGet, logURL, "")
		require.Nil(t, err, "httpGetBody failed:", err)

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "Unmarshal failed", err)
		require.Equal(t, errCodeServer, res.Errcode)
	})
	t.Run("logger is invalid", func(t *testing.T) {
		logURL := fmt.Sprintf("http://%s", s.server.Addr) + "/cmds/loglevel?logger=invalid"
		respData, err := httpRequest(http.MethodGet, logURL, "")
		require.Nil(t, err, "httpGetBody failed:", err)

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "Unmarshal failed", err)
		require.Equal(t, errCodeServer, res.Errcode)
		require.Equal(t, "logger invalid not found", res.Message)
	})
}

func TestCmdsConfig(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	configURL := fmt.Sprintf("http://%s//cmds/config", s.server.Addr)
	res := struct {
		Errcode int         `json:"errorcode"`
		Message string      `json:"message"`
		Content interface{} `json:"content"`
	}{}
	t.Run("failed to read configuration file", func(t *testing.T) {
		// Replace invalid config path
		s.config.configPath = "./invalid/invalid.yaml"
		respData, err := httpRequest(http.MethodGet, configURL, "")
		// Adjust back to the correct path
		s.config.configPath = testConfigPath
		require.Nil(t, err, "httpGetBody failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "unmarshal failed", err)
		require.Equal(t, errCodeServer, res.Errcode)
	})
	t.Run("failed to get unmarshaler", func(t *testing.T) {
		// Replace invalid unmarshaler
		config.RegisterUnmarshaler("yaml", nil)
		respData, err := httpRequest(http.MethodGet, configURL, "")
		// Adjust back to the correct unmarshaler
		config.RegisterUnmarshaler("yaml", &config.YamlUnmarshaler{})
		if err != nil {
			require.Nil(t, err, "httpGetBody failed")
			return
		}

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "unmarshal failed", err)
		require.Equal(t, errCodeServer, res.Errcode)
		require.Equal(t, "cannot find yaml unmarshaler", res.Message)
	})
	t.Run("failed to unmarshal configuration file", func(t *testing.T) {
		// Replace invalid config path
		s.config.configPath = "../testdata/greeter.trpc.go"
		respData, err := httpRequest(http.MethodGet, configURL, "")
		// Adjust back to the correct path
		s.config.configPath = testConfigPath
		require.Nil(t, err, "httpGetBody failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "unmarshal failed", err)
		require.Equal(t, errCodeServer, res.Errcode)
	})
	t.Run("right case", func(t *testing.T) {
		time.Sleep(1 * time.Second)
		respData, err := httpRequest(http.MethodGet, configURL, "")
		require.Nil(t, err, "httpGetBody failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "unmarshal failed", err)
		require.Equal(t, 0, res.Errcode)
		require.NotNil(t, res.Content, "config content is empty")
	})
}

func TestCmdsHealthCheck(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})

	rsp, err := http.Get(fmt.Sprintf("http://%s/is_healthy", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/not_exist", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	unregister, update, err := s.RegisterHealthCheck("service")
	require.Nil(t, err)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/service", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	update(healthcheck.Serving)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/service", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	update(healthcheck.NotServing)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/service", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)

	unregister()
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	rsp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/service", s.server.Addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestCmds(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})

	usercmdURL := fmt.Sprintf("http://%s", s.server.Addr) + "/cmds"
	respData, err := httpRequest(http.MethodGet, usercmdURL, "")
	require.Nil(t, err, "cmds request failed")

	res := struct {
		Errcode int      `json:"errorcode"`
		Message string   `json:"message"`
		Cmds    []string `json:"cmds"`
	}{}
	err = json.Unmarshal(respData, &res)
	require.Nil(t, err, "Unmarshal failed")
}

func TestErrorOutput(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})
	usercmdURL := fmt.Sprintf("http://%s", s.server.Addr) + "/errout"
	respData, err := httpRequest(http.MethodGet, usercmdURL, "")
	require.Nil(t, err, "cmds request failed")

	res := struct {
		Errcode int    `json:"errorcode"`
		Message string `json:"message"`
	}{}
	err = json.Unmarshal(respData, &res)
	require.Nil(t, err, "Unmarshal failed")
	require.Equal(t, 100, res.Errcode)
	require.Contains(t, res.Message, "error")
}

func TestPanicHandle(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)
	t.Cleanup(func() {
		if err := s.Close(nil); err != nil {
			t.Log(err)
		}
	})

	usercmdURL := fmt.Sprintf("http://%s", s.server.Addr) + "/panicHandle"
	respData, err := httpRequest(http.MethodGet, usercmdURL, "")
	require.Nil(t, err, "cmds request failed")

	res := struct {
		Errcode int    `json:"errorcode"`
		Message string `json:"message"`
	}{}
	err = json.Unmarshal(respData, &res)
	require.Nil(t, err, "Unmarshal failed")
	require.Equal(t, 500, res.Errcode)
	require.Contains(t, res.Message, "panic")
}

func TestListen(t *testing.T) {
	s := NewServer()

	// listen fail on invalid address
	err := os.Setenv(transport.EnvGraceRestart, "0")
	assert.Nil(t, err)
	ln, err := s.listen("tcp", "invalid address")
	assert.NotNil(t, err)
	assert.Nil(t, ln)

	// listen success
	ln, err = s.listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	assert.NotNil(t, ln)
	defer func(ln net.Listener) {
		assert.Nil(t, ln.Close())
	}(ln)
	assert.IsType(t, &net.TCPListener{}, ln)
}

func TestClose(t *testing.T) {
	s := newDefaultAdminServer()
	mustStartAdminServer(t, s)

	err := s.Close(nil)
	require.Nil(t, err)

	usercmdURL := fmt.Sprintf("http://%s/cmds", s.server.Addr)
	_, err = httpRequest(http.MethodGet, usercmdURL, "")
	var netErr *net.OpError

	require.ErrorAs(t, err, &netErr)
}

func TestOptionsConfig(t *testing.T) {
	s := newDefaultAdminServer()
	WithTLS(true)(s.config)
	err := s.Serve()
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "not support")
}

func httpRequest(method string, url string, body string) ([]byte, error) {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func userCmd(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("usercmd"))
}

func errOutput(w http.ResponseWriter, r *http.Request) {
	ErrorOutput(w, "error output", 100)
}

func panicHandle(w http.ResponseWriter, r *http.Request) {
	panic("panic error handle")
}

func TestUnregisterHandlers(t *testing.T) {
	_ = newDefaultAdminServer()
	mux, err := extractServeMuxData()
	require.Nil(t, err)
	require.Len(t, mux.m, 0)
	require.Len(t, mux.es, 0)
	require.False(t, mux.hosts)

	http.HandleFunc("/usercmd", userCmd)
	http.HandleFunc("/errout", errOutput)
	http.HandleFunc("/panicHandle", panicHandle)
	http.HandleFunc("www.qq.com/", userCmd)
	http.HandleFunc("anything/", userCmd)

	l := mustListenTCP(t)
	go func() {
		if err := http.Serve(l, nil); err != nil {
			t.Log(err)
		}
	}()
	time.Sleep(200 * time.Millisecond)

	mux, err = extractServeMuxData()
	require.Nil(t, err)
	require.Equal(t, 5, len(mux.m))
	require.Equal(t, 2, len(mux.es))
	require.Equal(t, true, mux.hosts)

	err = unregisterHandlers(
		[]string{
			"/usercmd",
			"/errout",
			"/panicHandle",
			"www.qq.com/",
			"anything/",
		},
	)
	require.Nil(t, err)

	mux, err = extractServeMuxData()
	require.Nil(t, err)
	require.Len(t, mux.m, 0)
	require.Len(t, mux.es, 0)
	require.False(t, mux.hosts)

	resp1, err := http.Get(fmt.Sprintf("http://%v/usercmd", l.Addr()))
	require.Nil(t, err)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusNotFound, resp1.StatusCode)

	http.HandleFunc("/usercmd", userCmd)
	http.HandleFunc("/errout", errOutput)
	http.HandleFunc("/panicHandle", panicHandle)

	mux, err = extractServeMuxData()
	require.Nil(t, err)
	require.Len(t, mux.m, 3)
	require.Len(t, mux.es, 0)
	require.False(t, mux.hosts)

	resp2, err := http.Get(fmt.Sprintf("http://%v/usercmd", l.Addr()))
	require.Nil(t, err)
	defer resp2.Body.Close()
	respBody, err := io.ReadAll(resp2.Body)
	require.Nil(t, err)
	require.Equal(t, []byte("usercmd"), respBody)
}
func mustListenTCP(t *testing.T) *net.TCPListener {
	l, err := net.Listen("tcp", testAddress)
	if err != nil {
		t.Fatal(err)
	}
	return l.(*net.TCPListener)
}

// serveMux keep the same structure with http.ServeMux
type serveMux struct {
	m     map[string]muxEntry
	es    []muxEntry
	hosts bool
}

// muxEntry keep the same structure with muxEntry in net/http pkg
type muxEntry struct {
}

// extractServeMuxData get http.DefaultServeMux 's data and show
func extractServeMuxData() (*serveMux, error) {
	v := reflect.ValueOf(http.DefaultServeMux)

	// lock
	muField := v.Elem().FieldByName("mu")
	if !muField.IsValid() {
		return nil, errors.New("http.DefaultServeMux does not have a field called `mu`")
	}
	muPointer := unsafe.Pointer(muField.UnsafeAddr())
	mu := (*sync.RWMutex)(muPointer)
	(*mu).Lock()
	defer (*mu).Unlock()

	// get value of map
	mField := v.Elem().FieldByName("m")
	if !mField.IsValid() {
		return nil, errors.New("http.DefaultServeMux does not have a field called `m`")
	}
	mPointer := unsafe.Pointer(mField.UnsafeAddr())
	m := (*map[string]muxEntry)(mPointer)

	// get value of slice
	esField := v.Elem().FieldByName("es")
	if !esField.IsValid() {
		return nil, errors.New("http.DefaultServeMux does not have a field called `es`")
	}
	esPointer := unsafe.Pointer(esField.UnsafeAddr())
	es := (*[]muxEntry)(esPointer)

	// get hosts
	hostsField := v.Elem().FieldByName("hosts")
	if !hostsField.IsValid() {
		return nil, errors.New("http.DefaultServeMux does not have a field called `hosts`")
	}
	hostsPointer := unsafe.Pointer(hostsField.UnsafeAddr())
	hosts := (*bool)(hostsPointer)

	return &serveMux{
		m:     *m,
		es:    *es,
		hosts: *hosts,
	}, nil
}

func TestTrpcAdminServer(t *testing.T) {
	s := NewServer(WithAddr("invalid addr"))
	err := s.Serve()
	require.NotNil(t, err)

	s = NewServer(WithAddr(testAddress))
	err = s.Register(struct{}{}, struct{}{})
	require.Nil(t, err)

	go func() {
		if err := s.Serve(); err != nil {
			t.Log(err)
		}
	}()
	time.Sleep(200 * time.Millisecond)

	ch := make(chan struct{}, 1)
	err = s.Close(ch)
	closed := <-ch
	require.NotNil(t, closed)
	require.Nil(t, err)
}
