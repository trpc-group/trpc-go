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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
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
	testAddress    = "127.0.0.1:0"
	testVersion    = "v0.2.0-alpha"
	testConfigPath = "../testdata/trpc_go.yaml"
)

var baseURL = fmt.Sprintf("http://%s", defaultListenAddr)

var adminServer = NewTrpcAdminServer(

	WithVersion(testVersion),
	WithAddr(defaultListenAddr),
	WithTLS(false),
	WithReadTimeout(defaultReadTimeout),
	WithWriteTimeout(defaultWriteTimeout),
	WithConfigPath(testConfigPath),
)

func TestMain(m *testing.M) {
	HandleFunc("/usercmd", userCmd)
	HandleFunc("/errout", errOutput)
	HandleFunc("/panicHandle", panicHandle)
	startAdminServer()
	exitCode := m.Run()
	adminServer.Close(nil)
	os.Exit(exitCode)
}

func startAdminServer() {
	go func() {
		err := adminServer.Serve()
		if err != nil {
			log.Errorf("serve error: %s", err)
		}
	}()
	time.Sleep(200 * time.Millisecond)
}

func TestRPCZFailed(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		errorCode int
		message   string
		content   interface{}
	}{
		{
			name:      "handleSpans failed because query parameter isn't a number",
			url:       baseURL + patternRPCZSpansList + "?num=xxx",
			errorCode: ErrCodeServer,
			message:   "must be a integer",
			content:   "",
		},
		{
			name:      "handleSpans failed because query parameter isn't a positive integer",
			url:       baseURL + patternRPCZSpansList + "?num=-1",
			errorCode: ErrCodeServer,
			message:   "must be a non-negative integer",
			content:   nil,
		},
		{
			name:      "handleSpan failed because can't find span_id",
			url:       baseURL + patternRPCZSpanGet + "1",
			errorCode: ErrCodeServer,
			message:   "cannot find span-id",
			content:   nil,
		},
		{
			name:      "handleSpan failed because query parameter span_id is empty",
			url:       baseURL + patternRPCZSpanGet + "",
			errorCode: ErrCodeServer,
			message:   "undefined command",
			content:   nil,
		},
		{
			name:      "handleSpan failed because query parameter span_id is negative",
			url:       baseURL + patternRPCZSpanGet + "-1",
			errorCode: ErrCodeServer,
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
		r, err := httpRequest(http.MethodGet, baseURL+"/cmd/rpcz", "")
		require.Nil(t, err)
		require.Contains(t, string(r), "404 page not found")
	})
	t.Run("method not allowed", func(t *testing.T) {
		r, err := httpRequest(http.MethodDelete, baseURL+patternRPCZSpansList+"?num", "")
		require.Nil(t, err)
		require.Contains(t, string(r), "Method Not Allowed")
	})
}

type sliceSpanExporter struct {
	spans []rpcz.ReadOnlySpan
}

func (e *sliceSpanExporter) Export(span *rpcz.ReadOnlySpan) {
	e.spans = append(e.spans, *span)
}

func TestRPC_Exporter(t *testing.T) {
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
	rRaw, err := httpRequest(http.MethodGet, baseURL+patternRPCZSpansList+"?num", "")
	require.Nil(t, err)
	require.Contains(t, string(rRaw), fmt.Sprint(spanID))
}

func TestRPCZOk(t *testing.T) {
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
			url:     baseURL + patternRPCZSpansList + "?num",
			content: fmt.Sprintf("1:\n  span: (server, %d)\n", spanID),
		},
		{
			name:    "handleSpans ok without any query parameter",
			url:     baseURL + patternRPCZSpansList,
			content: fmt.Sprintf("1:\n  span: (server, %d)\n", spanID),
		},
		{
			name:    "handleSpans ok",
			url:     baseURL + patternRPCZSpansList + "?num=1",
			content: fmt.Sprintf("1:\n  span: (server, %d)\n", spanID),
		},
		{
			name:    "handleSpan ok",
			url:     baseURL + patternRPCZSpanGet + fmt.Sprint(spanID),
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
	res := struct {
		Errcode int    `json:"errorcode"`
		Message string `json:"message"`
		Version string `json:"version"`
	}{}
	t.Run("ok", func(t *testing.T) {
		versionURL := baseURL + "/version"
		respData, err := httpRequest(http.MethodGet, versionURL, "")
		if err != nil {
			require.Nil(t, err, "httpGetBody failed")
			return
		}
		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "testAdminServerVersion unmarshal failed")
		require.Equal(t, 0, res.Errcode)
		require.Equal(t, testVersion, res.Version)
	})
	t.Run("method not allowed", func(t *testing.T) {
		versionURL := baseURL + "/version"
		rsp, err := httpRequest(http.MethodDelete, versionURL, "")
		require.Nil(t, err)
		err = json.Unmarshal(rsp, &res)
		require.Nil(t, err)
		require.Equal(t, http.StatusMethodNotAllowed, res.Errcode)
	})
}

func TestCmdsLogLevel(t *testing.T) {
	dlogger := log.GetDefaultLogger()

	// Preset test conditions
	log.Register(
		"default", log.NewZapLog(
			[]log.OutputConfig{
				{Writer: log.OutputConsole, Level: "debug"},
				{Writer: log.OutputFile, WriteConfig: log.WriteConfig{Filename: "test"}, Level: "info"},
			},
		),
	)

	t.Cleanup(
		func() {
			log.Register("default", dlogger)
		},
	)

	res := struct {
		Errcode  int    `json:"errorcode"`
		Message  string `json:"message"`
		Level    string `json:"level"`
		PreLevel string `json:"prelevel"`
	}{}

	// case: correct
	t.Run(
		"right_case", func(t *testing.T) {
			logURL := baseURL + "/cmds/loglevel?logger=default&output=1"
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
		},
	)
	t.Run("method not allowed", func(t *testing.T) {
		logURL := baseURL + "/cmds/loglevel?logger=default&output=1"
		respData, err := httpRequest(http.MethodDelete, logURL, "")
		require.Nil(t, err, "httpGetBody failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "testAdminServerLogLevel unmarshal failed")
		require.Equal(t, http.StatusMethodNotAllowed, res.Errcode)
	},
	)
	// case: Request parameter is empty
	t.Run(
		"nil_query_param_case", func(t *testing.T) {
			logURL := baseURL + "/cmds/loglevel"
			respData, err := httpRequest(http.MethodGet, logURL, "")
			require.Nil(t, err, "httpGetBody failed")

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "unmarshal failed")
			require.Equal(t, 0, res.Errcode)
			require.Equal(t, "debug", res.Level)
		},
	)

	// case: Failed to parse request parameters
	t.Run(
		"parse_form_err_case", func(t *testing.T) {
			logURL := baseURL + "/cmds/loglevel?logger%"
			respData, err := httpRequest(http.MethodGet, logURL, "")
			require.Nil(t, err, "httpGetBody failed:", err)

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "Unmarshal failed", err)
			require.Equal(t, ErrCodeServer, res.Errcode)
		},
	)

	// case: logger is invalid
	t.Run(
		"invalid_logger_err_case", func(t *testing.T) {
			logURL := baseURL + "/cmds/loglevel?logger=invalid"
			respData, err := httpRequest(http.MethodGet, logURL, "")
			require.Nil(t, err, "httpGetBody failed:", err)

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "Unmarshal failed", err)
			require.Equal(t, ErrCodeServer, res.Errcode)
			require.Equal(t, "logger not found", res.Message)
		},
	)
}

func TestCmdsConfig(t *testing.T) {
	versionURL := baseURL + "/cmds/config"
	res := struct {
		Errcode int         `json:"errorcode"`
		Message string      `json:"message"`
		Content interface{} `json:"content"`
	}{}

	// case: correct
	t.Run(
		"right_case", func(t *testing.T) {
			respData, err := httpRequest(http.MethodGet, versionURL, "")
			if err != nil {
				require.Nil(t, err, "httpGetBody failed")
				return
			}

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "unmarshal failed", err)
			require.Equal(t, 0, res.Errcode)
			require.NotNil(t, res.Content, "config content is empty")
		},
	)
	t.Run("method not allowed", func(t *testing.T) {
		respData, err := httpRequest(http.MethodDelete, versionURL, "")
		if err != nil {
			require.Nil(t, err, "httpGetBody failed")
			return
		}

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "unmarshal failed", err)
		require.Equal(t, http.StatusMethodNotAllowed, res.Errcode)
	},
	)

	// case: Failed to read configuration file
	t.Run(
		"read_file_fail_case", func(t *testing.T) {
			server := adminServer
			// Replace invalid config path
			server.config.configPath = "./invalid/invalid.yaml"
			respData, err := httpRequest(http.MethodGet, versionURL, "")
			// Adjust back to the correct path
			server.config.configPath = testConfigPath
			if err != nil {
				require.Nil(t, err, "httpGetBody failed")
				return
			}

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "unmarshal failed", err)
			require.Equal(t, ErrCodeServer, res.Errcode)
		},
	)

	// case: Failed to get unmarshaler
	t.Run(
		"get_unmarshaler_fail_case", func(t *testing.T) {
			// Replace invalid unmarshaler
			config.RegisterUnmarshaler("yaml", nil)
			respData, err := httpRequest(http.MethodGet, versionURL, "")
			// Adjust back to the correct unmarshaler
			config.RegisterUnmarshaler("yaml", &config.YamlUnmarshaler{})
			if err != nil {
				require.Nil(t, err, "httpGetBody failed")
				return
			}

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "unmarshal failed", err)
			require.Equal(t, ErrCodeServer, res.Errcode)
			require.Equal(t, "cannot find yaml unmarshaler", res.Message)
		},
	)

	// case: Failed to unmarshal configuration file
	t.Run(
		"unmarshal_file_fail_case", func(t *testing.T) {

			versionURL := baseURL + "/cmds/config"
			server := adminServer
			// Replace invalid config path
			server.config.configPath = "../testdata/greeter.trpc.go"
			respData, err := httpRequest(http.MethodGet, versionURL, "")
			// Adjust back to the correct path
			server.config.configPath = testConfigPath
			if err != nil {
				require.Nil(t, err, "httpGetBody failed")
				return
			}

			err = json.Unmarshal(respData, &res)
			require.Nil(t, err, "unmarshal failed", err)
			require.Equal(t, ErrCodeServer, res.Errcode)
		},
	)
}

func TestCmdsHealthCheck(t *testing.T) {
	rsp, err := http.Get(baseURL + "/is_healthy")
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = http.Get(baseURL + "/is_healthy/")
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = http.Get(baseURL + "/is_healthy/not_exist")
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	unregister, update, err := adminServer.RegisterHealthCheck("service")
	require.Nil(t, err)
	rsp, err = http.Get(baseURL + "/is_healthy")
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
	rsp, err = http.Get(baseURL + "/is_healthy/service")
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	update(healthcheck.Serving)
	rsp, err = http.Get(baseURL + "/is_healthy")
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	rsp, err = http.Get(baseURL + "/is_healthy/service")
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	update(healthcheck.NotServing)
	rsp, err = http.Get(baseURL + "/is_healthy")
	require.Nil(t, err)
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
	rsp, err = http.Get(baseURL + "/is_healthy/service")
	require.Nil(t, err)
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)

	unregister()
	rsp, err = http.Get(baseURL + "/is_healthy")
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	rsp, err = http.Get(baseURL + "/is_healthy/service")
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestCmds(t *testing.T) {
	res := struct {
		Errcode int      `json:"errorcode"`
		Message string   `json:"message"`
		Cmds    []string `json:"cmds"`
	}{}
	t.Run("ok", func(t *testing.T) {
		usercmdURL := baseURL + "/cmds"
		respData, err := httpRequest(http.MethodGet, usercmdURL, "")
		require.Nil(t, err, "cmds request failed")

		err = json.Unmarshal(respData, &res)
		require.Nil(t, err, "Unmarshal failed")
	})
	t.Run("method not allowed", func(t *testing.T) {
		versionURL := baseURL + "/version"
		rsp, err := httpRequest(http.MethodDelete, versionURL, "")
		require.Nil(t, err)
		err = json.Unmarshal(rsp, &res)
		require.Nil(t, err)
		require.Equal(t, http.StatusMethodNotAllowed, res.Errcode)
	})
}

func TestErrorOutput(t *testing.T) {
	usercmdURL := baseURL + "/errout"
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

func TestPanicHanle(t *testing.T) {
	usercmdURL := baseURL + "/panicHandle"
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
	s := NewTrpcAdminServer()

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
	err := adminServer.Close(nil)
	require.Nil(t, err)

	usercmdURL := baseURL + "/cmds"
	_, err = httpRequest(http.MethodGet, usercmdURL, "")
	var netErr *net.OpError
	require.ErrorAs(t, err, &netErr)

	startAdminServer()
}

func TestOptionsConfig(t *testing.T) {
	adminServer.Close(nil)

	WithTLS(true)(adminServer.config)
	err := adminServer.Serve()
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "not support")

	startAdminServer()
}

func httpRequest(method string, url string, body string) ([]byte, error) {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("content-type", "application/x-www-form-urlencoded")

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

func Test_init(t *testing.T) {
	t.Run("reset default serve mux to remove pprof registration at admin init func", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.Nil(t, err)
		go func() {
			if err := http.Serve(l, nil); err != nil {
				t.Logf("http serving: %v", err)
			}
		}()
		time.Sleep(200 * time.Millisecond)

		r, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/cmdline", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/profile", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/symbol", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/trace", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, r.StatusCode)
	})
	t.Run("register pprof handler explicitly after importing the admin package", func(t *testing.T) {
		http.DefaultServeMux.HandleFunc("/debug/pprof/", pprof.Index)
		http.DefaultServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		http.DefaultServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		http.DefaultServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		http.DefaultServeMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		t.Cleanup(func() {
			http.DefaultServeMux = http.NewServeMux()
		})
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.Nil(t, err)
		go func() {
			if err := http.Serve(l, nil); err != nil {
				t.Logf("http serving: %v", err)
			}
		}()
		time.Sleep(200 * time.Millisecond)

		r, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/cmdline", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/symbol", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, r.StatusCode)

		r, err = http.Get(fmt.Sprintf("http://%s/debug/pprof/trace", l.Addr().String()))
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, r.StatusCode)
	})
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
	s := NewTrpcAdminServer(WithAddr("invalid addr"))
	err := s.Serve()
	require.NotNil(t, err)

	s = NewTrpcAdminServer(WithAddr("127.0.0.1:9038"))
	err = s.Register(struct{}{}, struct{}{})
	require.Nil(t, err)

	go func() {
		_ = s.Serve()
	}()

	time.Sleep(200 * time.Millisecond)
	ch := make(chan struct{}, 1)
	err = s.Close(ch)
	closed := <-ch
	require.NotNil(t, closed)
	require.Nil(t, err)
	startAdminServer()
}
