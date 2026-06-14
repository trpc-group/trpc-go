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

package tls_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httptls "trpc.group/trpc-go/trpc-go/http/tls"
	"trpc.group/trpc-go/trpc-go/internal/tls"
)

func TestGetServerConfig(t *testing.T) {
	_, err := tls.GetServerConfig("../../testdata/ca.pem", "../../testdata/server.crt", "../../testdata/server.key", "")
	assert.Nil(t, err)
	_, err = tls.GetServerConfig("", "../../testdata/server.crt", "../../testdata/server.key", "")
	assert.Nil(t, err)
	_, err = tls.GetServerConfig("", "", "", "")
	assert.NotNil(t, err)
}

func TestGetClientConfig(t *testing.T) {
	_, err := tls.GetClientConfig(
		"localhost",
		"../../testdata/ca.pem",
		"../../testdata/client.crt",
		"../../testdata/client.key",
		"",
	)
	assert.Nil(t, err)
	_, err = tls.GetClientConfig("localhost", "none", "../../testdata/client.crt", "../../testdata/client.key", "")
	assert.Nil(t, err)
	_, err = tls.GetClientConfig("localhost", "../../testdata/ca.pem", "", "", "")
	assert.Nil(t, err)
	_, err = tls.GetClientConfig("localhost", "root", "", "", "")
	assert.Nil(t, err)
}

func TestGetConfigWithProvider(t *testing.T) {
	const providerName = "internal_tls_provider_test"
	provider := mapProvider{
		"ca":     readFile(t, "../../testdata/ca.pem"),
		"server": readFile(t, "../../testdata/server.crt"),
		"skey":   readFile(t, "../../testdata/server.key"),
		"client": readFile(t, "../../testdata/client.crt"),
		"ckey":   readFile(t, "../../testdata/client.key"),
	}
	httptls.RegisterCertificateProvider(providerName, provider)
	defer httptls.UnregisterCertificateProvider(providerName)

	serverConfig, err := tls.GetServerConfig("ca", "server", "skey", providerName)
	require.NoError(t, err)
	require.Len(t, serverConfig.Certificates, 1)
	require.NotNil(t, serverConfig.ClientCAs)

	clientConfig, err := tls.GetClientConfig("localhost", "ca", "client", "ckey", providerName)
	require.NoError(t, err)
	require.Len(t, clientConfig.Certificates, 1)
	require.NotNil(t, clientConfig.RootCAs)
}

type mapProvider map[string][]byte

func (p mapProvider) LoadCertFile(certID string) ([]byte, error) {
	b, ok := p[certID]
	if !ok {
		return nil, fmt.Errorf("cert %q not found", certID)
	}
	return b, nil
}

func (p mapProvider) LoadKeyFile(keyID string) ([]byte, error) {
	b, ok := p[keyID]
	if !ok {
		return nil, fmt.Errorf("key %q not found", keyID)
	}
	return b, nil
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	require.NoError(t, err)
	return b
}
