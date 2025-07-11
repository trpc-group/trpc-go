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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stretchr/testify/require"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/healthcheck"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/server"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestAdmin() {
	trpc.ServerConfigPath = "trpc_go_trpc_server_with_admin.yaml"

	sf := func(
		ctx context.Context,
		req interface{},
		next filter.ServerHandleFunc,
	) (rsp interface{}, err error) {
		span := rpcz.SpanFromContext(ctx)
		span.AddEvent("user's annotation at pre-filter")
		rsp, err = next(ctx, req)
		span.AddEvent("user's annotation at post-filter")
		return rsp, err
	}

	cf := func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
		return next(ctx, req, rsp)
	}

	s.startTRPCServerWithListener(
		&TRPCService{
			EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
				span := rpcz.SpanFromContext(ctx)
				span.AddEvent("handling EmptyCallF")
				c := testpb.NewTestTRPCClientProxy(client.WithTarget(s.serverAddress()))
				_, err := c.UnaryCall(
					ctx,
					s.defaultSimpleRequest,
					client.WithNamedFilter("filter1", cf),
					client.WithFilter(cf),
				)
				require.Nil(s.T(), err)

				cs, _ := span.NewChild("sleep")
				time.Sleep(100 * time.Millisecond)
				cs.AddEvent("awake")

				_, err = c.UnaryCall(
					ctx,
					s.defaultSimpleRequest,
					client.WithNamedFilter("filter2", cf),
					client.WithFilter(cf),
				)
				require.Nil(s.T(), err)
				return &testpb.Empty{}, nil
			},
		},
		server.WithNamedFilter("filter1", sf),
		server.WithNamedFilter("filter2", sf),
		server.WithFilter(sf),
	)
	// wait a while until admin server has started.
	time.Sleep(200 * time.Millisecond)

	s.Run("cmds", s.testCmds)
	s.Run("cmds-config", s.testCmdsConfig)
	s.Run("cmds-loglevel", s.testCmdsLogLevel)
	s.Run("CustomHandleFunc", s.testCustomHandleFunc)
	s.Run("is-healthy", s.testIsHealthy)
	s.Run("cmd-rpcz-BriefSpans", s.testRPCZBriefSpansOk)
	s.Run("cmd-rpcz-DetailedSpan", s.testDetailedSpanOk)
}

func (s *TestSuite) testCmds() {
	resp, err := httpRequest(http.MethodGet, fmt.Sprintf("http://%s/cmds", defaultAdminListenAddr), "")
	require.Nil(s.T(), err)

	r := struct {
		Errcode int      `json:"errorcode"`
		Message string   `json:"message"`
		Cmds    []string `json:"cmds"`
	}{}
	require.Nil(s.T(), json.Unmarshal(resp, &r), "Unmarshal failed")
	require.ElementsMatch(
		s.T(),
		[]string{
			"/cmds",
			"/version",
			"/debug/pprof/",
			"/debug/pprof/symbol",
			"/debug/pprof/trace",
			"/cmds/loglevel",
			"/cmds/config",
			"/is_healthy/",
			"/debug/pprof/cmdline",
			"/debug/pprof/profile",
			"/cmds/rpcz/spans/",
			"/cmds/rpcz/spans",
		},
		r.Cmds,
	)
}

func (s *TestSuite) testCmdsConfig() {
	resp, err := httpRequest(http.MethodGet, fmt.Sprintf("http://%s/cmds/config", defaultAdminListenAddr), "")
	require.Nil(s.T(), err)

	r := struct {
		Errcode int         `json:"errorcode"`
		Message string      `json:"message"`
		Content interface{} `json:"content"`
	}{}
	require.Nil(s.T(), json.Unmarshal(resp, &r), "Unmarshal failed")

	buf, err := os.ReadFile(trpc.ServerConfigPath)
	require.Nil(s.T(), err)

	conf := map[interface{}]interface{}{}
	unmarshaler := config.GetUnmarshaler("yaml")
	require.NotNil(s.T(), unmarshaler)
	require.Nil(s.T(), unmarshaler.Unmarshal(buf, &conf))
	require.Equal(s.T(), fmt.Sprint(conf), fmt.Sprint(r.Content))
}

func (s *TestSuite) testCmdsLogLevel() {
	logURL := fmt.Sprintf("http://%s/cmds/loglevel", defaultAdminListenAddr)
	resp, err := httpRequest(
		http.MethodGet,
		logURL,
		"",
	)
	require.Nil(s.T(), err)

	r := struct {
		Errcode  int    `json:"errorcode"`
		Message  string `json:"message"`
		Level    string `json:"level"`
		PreLevel string `json:"prelevel"`
	}{}
	require.Nil(s.T(), json.Unmarshal(resp, &r), "Unmarshal failed")
	require.Equal(s.T(), "debug", r.Level)

	resp, err = httpRequest(http.MethodPut, logURL, "value=info")
	require.Nil(s.T(), err)
	require.Nil(s.T(), json.Unmarshal(resp, &r), "Unmarshal failed")
	require.Equal(s.T(), "info", r.Level)
}

func (s *TestSuite) testIsHealthy() {
	isHealthyURL := fmt.Sprintf("http://%s/is_healthy", defaultAdminListenAddr)
	resp, err := http.Get(isHealthyURL)
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)

	resp, err = http.Get(fmt.Sprintf("http://%s/is_healthy/not_exist", defaultAdminListenAddr))
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusNotFound, resp.StatusCode)

	adminServer, err := trpc.GetAdminService(s.server)
	require.Nil(s.T(), err)
	_, update, err := adminServer.RegisterHealthCheck("service")
	require.Nil(s.T(), err)

	update(healthcheck.NotServing)
	resp, err = http.Get(isHealthyURL)
	require.Nil(s.T(), err)
	require.Equal(s.T(), http.StatusServiceUnavailable, resp.StatusCode)
}

func (s *TestSuite) testCustomHandleFunc() {
	if as, ok := s.server.Service(admin.ServiceName).(*admin.Server); ok {
		as.HandleFunc(
			"/customHandle",
			func(http.ResponseWriter, *http.Request) {
				panic("panic error handle")
			},
		)
	} else {
		s.T().Fatal("config admin handle function failed")
	}

	resp, err := httpRequest(
		http.MethodGet,
		fmt.Sprintf("http://%s/customHandle", defaultAdminListenAddr),
		"",
	)
	require.Nil(s.T(), err)

	r := struct {
		Errcode int    `json:"errorcode"`
		Message string `json:"message"`
	}{}
	require.Nil(s.T(), json.Unmarshal(resp, &r), "Unmarshal failed")
	require.Equal(s.T(), http.StatusInternalServerError, r.Errcode)
	require.Contains(s.T(), r.Message, "panic error handle")
}

func httpRequest(method string, url string, body string) ([]byte, error) {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

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
