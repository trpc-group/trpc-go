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
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/transport"
)

type mockSvrStreamTransport struct{}

// ListenAndServe mocks stream Transport listener interface.
func (t *mockSvrStreamTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) error {
	return nil
}

// Send mocks stream Transport send method.
func (t *mockSvrStreamTransport) Send(ctx context.Context, req []byte) error {
	return nil
}

func (t *mockSvrStreamTransport) Close(ctx context.Context) {
	return
}

type mockClientStreamTransport struct{}

// RoundTrip mocks ClientStreamTransport roundtrip method.
func (c *mockClientStreamTransport) RoundTrip(ctx context.Context, req []byte,
	opts ...transport.RoundTripOption) ([]byte, error) {
	return nil, nil
}

// Send is a noop implementation which mocks ClientStreamTransport.Send.
func (c *mockClientStreamTransport) Send(ctx context.Context, req []byte, opts ...transport.RoundTripOption) error {
	return nil
}

// Recv is a noop implementation which mocks ClientStreamTransport.Recv.
func (c *mockClientStreamTransport) Recv(ctx context.Context, opts ...transport.RoundTripOption) ([]byte, error) {
	return nil, nil
}

// Init is a noop implementation which mocks ClientStreamTransport.Init.
func (c *mockClientStreamTransport) Init(ctx context.Context, opts ...transport.RoundTripOption) error {
	return nil
}

// Close is a noop implementation which mocks ClientStreamTransport.Close.
func (c *mockClientStreamTransport) Close(ctx context.Context) {
	return
}

// TestGetServerStreamTransport tests register and get of server stream transport.
func TestGetServerStreamTransport(t *testing.T) {
	transport.RegisterServerStreamTransport("mock", &mockSvrStreamTransport{})
	ts := transport.GetServerStreamTransport("mock")
	assert.NotNil(t, ts)
	assert.Equal(t, &mockSvrStreamTransport{}, ts)
}

// TestGetClientStreamTransport tests register and get client stream transport.
func TestGetClientStreamTransport(t *testing.T) {
	transport.RegisterClientStreamTransport("mock", &mockClientStreamTransport{})
	tc := transport.GetClientStreamTransport("mock")
	assert.NotNil(t, tc)
	assert.Equal(t, &mockClientStreamTransport{}, tc)

	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterClientStreamTransport("mock", nil)
}

// TestClientStreamTransportNilPointer tests register nil client stream transport.
func TestClientStreamTransportNilPointer(t *testing.T) {
	// Test ClientTransport nil.
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	var c *mockClientStreamTransport
	transport.RegisterClientStreamTransport("mock", c)
}

// TestRegisterClientTransport_NameNil tests register of an empty named client stream transport.
func TestRegisterClientStreamTransport_NameNil(t *testing.T) {
	// Test name nil.
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	transport.RegisterClientStreamTransport("", &mockClientStreamTransport{})
}

// TestRegisterServerTransport_Nil tests register a nil server stream transport.
func TestRegisterServerStreamTransport_Nil(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterServerStreamTransport("mock", nil)
}

// TestRegisterServerTransport_NilPointer tests register a nil server stream transport.
func TestRegisterServerStreamTransport_NilPointer(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()

	var ts *mockSvrStreamTransport
	transport.RegisterServerStreamTransport("mock", ts)
}

// TestRegisterServerStreamTransport_EmptyName tests register of a empty named server stream
// transport.
func TestRegisterServerStreamTransport_EmptyName(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterServerStreamTransport("", &mockSvrStreamTransport{})
}

// TestRegisterNilSvrStreamTransport tests nil stream transport.
func TestRegisterNilSvrStreamTransport(t *testing.T) {
	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	transport.RegisterServerStreamTransport("mock", nil)
}
