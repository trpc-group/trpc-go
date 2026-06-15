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
	"bytes"
	"errors"
	"testing"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/require"
)

type errorWriter struct{}

func (e errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestWriteSSE(t *testing.T) {
	event := sse.Event{
		Comment: []byte("comment"),
		ID:      []byte("1"),
		Event:   []byte("message"),
		Retry:   []byte("1000"),
		Data:    []byte("hello"),
	}
	var buf bytes.Buffer
	require.NoError(t, WriteSSE(&buf, event))
	require.Equal(t, ":comment\nid:1\nevent:message\nretry:1000\ndata:hello\n\n", buf.String())
}

func TestWriteSSEError(t *testing.T) {
	require.Error(t, WriteSSE(errorWriter{}, sse.Event{Data: []byte("hello")}))
}
