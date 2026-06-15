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
	"context"
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestClientTransportGetRequestDoesNotSetTLSServerNameFromHost(t *testing.T) {
	ct := NewClientTransport(false).(*ClientTransport)
	_, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName("/test")

	opts := &transport.RoundTripOptions{Address: "127.0.0.1:443"}
	req, err := ct.getRequest(&ClientReqHeader{
		Method: stdhttp.MethodGet,
		Host:   "example.com",
	}, nil, msg, opts)

	require.NoError(t, err)
	require.Equal(t, "example.com", req.Host)
	require.Empty(t, opts.TLSServerName)
}
