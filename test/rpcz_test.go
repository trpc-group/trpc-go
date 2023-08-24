// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package test

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"trpc.group/trpc-go/trpc-go"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) testRPCZBriefSpansOk() {
	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	require.Nil(s.T(), err)
	_, err = c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})
	rpczURL := fmt.Sprintf("http://%s/cmds/rpcz/spans?num=4", defaultAdminListenAddr)
	_, err = httpRequest(
		http.MethodGet,
		rpczURL,
		"",
	)
	require.Nil(s.T(), err)
}

func (s *TestSuite) testDetailedSpanOk() {
	c := s.newTRPCClient()
	_, err := c.EmptyCall(trpc.BackgroundContext(), &testpb.Empty{})

	rpczURL := fmt.Sprintf("http://%s/cmds/rpcz/spans?num=10", defaultAdminListenAddr)
	resp, err := httpRequest(
		http.MethodGet,
		rpczURL,
		"",
	)
	_, err = html.Parse(strings.NewReader(string(resp)))
	require.Nil(s.T(), err)

	strs := strings.Split(string(resp), "\n")
	if len(strs) <= 2 {
		return
	}
	spanID := strings.TrimSuffix(strings.TrimPrefix(strs[5], "  span: (client, "), ")")
	spanID = strings.TrimSuffix(strings.TrimPrefix(spanID, "  span: (server, "), ")")
	func() {
		rpczURL := fmt.Sprintf("http://%s/cmds/rpcz/spans/%s", defaultAdminListenAddr, spanID)
		_, err := httpRequest(
			http.MethodGet,
			rpczURL,
			"",
		)
		require.Nil(s.T(), err)
	}()
}
