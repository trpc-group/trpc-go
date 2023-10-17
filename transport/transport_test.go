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

package transport_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
)

type mockSvrTransport struct{}

func (t *mockSvrTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) error {
	return nil
}

type mockClientTransport struct{}

func (c *mockClientTransport) RoundTrip(ctx context.Context, req []byte,
	opts ...transport.RoundTripOption) ([]byte, error) {
	return nil, nil
}

type mockFramerBuilder struct{}

func (f *mockFramerBuilder) New(reader io.Reader) codec.Framer {
	return nil
}

func TestListenAndServe(t *testing.T) {
	var err error
	err = transport.ListenAndServe()
	assert.NotNil(t, err)
}

func TestRoundTrip(t *testing.T) {
	_, err := transport.RoundTrip(context.Background(), nil)
	assert.NotNil(t, err)
}

func TestGetFramerBuilder(t *testing.T) {
	transport.RegisterFramerBuilder("mock", &mockFramerBuilder{})
	f := transport.GetFramerBuilder("mock")
	assert.NotNil(t, f)
	assert.Equal(t, &mockFramerBuilder{}, f)
}

func TestRegisterFramerBuilder_BuilderNil(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	transport.RegisterFramerBuilder("mock", nil)
}

func TestRegisterFramerBuilder_BuilderNilPointer(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	var fb *mockFramerBuilder
	transport.RegisterFramerBuilder("mock", fb)
}

func TestRegisterFramerBuilder_NameNil(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	transport.RegisterFramerBuilder("", &mockFramerBuilder{})
}

func TestGetServerTransport(t *testing.T) {
	transport.RegisterServerTransport("mock", &mockSvrTransport{})
	ts := transport.GetServerTransport("mock")
	assert.NotNil(t, ts)
	assert.Equal(t, &mockSvrTransport{}, ts)
}

func TestGetClientTransport(t *testing.T) {
	transport.RegisterClientTransport("mock", &mockClientTransport{})
	tc := transport.GetClientTransport("mock")
	assert.NotNil(t, tc)
	assert.Equal(t, &mockClientTransport{}, tc)

	// Test ClientTransport nil.
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterClientTransport("mock", nil)
}

func TestClientTransportNilPointer(t *testing.T) {
	// Test ClientTransport nil.
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	var c *mockClientTransport
	transport.RegisterClientTransport("mock", c)
}

func TestRegisterClientTransport_NameNil(t *testing.T) {
	// Test name nil.
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	transport.RegisterClientTransport("", &mockClientTransport{})
}

func TestRegisterServerTransport_Nil(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterServerTransport("mock", nil)
}

func TestRegisterServerTransport_NilPointer(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	var ts *mockSvrTransport
	transport.RegisterServerTransport("mock", ts)
}

func TestRegisterServerTransport_EmptyName(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterServerTransport("", &mockSvrTransport{})
}

func TestRegisterNilSvrTransport(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterServerTransport("mock", nil)
}

func TestRemoteAddrFromContext(t *testing.T) {
	addr := transport.RemoteAddrFromContext(context.Background())
	assert.Nil(t, addr)
}
