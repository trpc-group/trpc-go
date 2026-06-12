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
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestOptServerTransport(t *testing.T) {
	st := NewServerTransport(
		func() *http.Server { return &http.Server{} },
		WithReusePort(),
		WithEnableH2C(),
		WithHTTP2Config(&transport.HTTP2Config{MaxConcurrentStreams: 1}))
	require.True(t, st.reusePort)
	require.True(t, st.enableH2C)
	require.Equal(t, 1, st.http2Config.MaxConcurrentStreams)
}
