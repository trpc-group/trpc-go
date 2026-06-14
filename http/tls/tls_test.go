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

package tls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testProvider struct{}

func (testProvider) LoadCertFile(string) ([]byte, error) { return nil, nil }
func (testProvider) LoadKeyFile(string) ([]byte, error)  { return nil, nil }

func TestCertificateProvider(t *testing.T) {
	const providerName = "test_provider"
	UnregisterCertificateProvider(providerName)
	require.Nil(t, GetCertificateProvider(providerName))

	provider := testProvider{}
	RegisterCertificateProvider(providerName, provider)
	require.Equal(t, provider, GetCertificateProvider(providerName))

	UnregisterCertificateProvider(providerName)
	require.Nil(t, GetCertificateProvider(providerName))
}
