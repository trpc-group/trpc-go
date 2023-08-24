// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptServerTransport(t *testing.T) {
	st := NewServerTransport(
		func() *http.Server { return &http.Server{} },
		WithReusePort(),
		WithEnableH2C())
	require.True(t, st.reusePort)
	require.True(t, st.enableH2C)
}
