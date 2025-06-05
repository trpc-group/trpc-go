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

package tls_test

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	itls "trpc.group/trpc-go/trpc-go/internal/tls"
)

const (
	tlsFileSeparator = ":"

	caPem       = "../../testdata/ca.pem"
	notExistPem = "not_exist.pem"

	serverCert   = "../../testdata/server.crt"
	serverKey    = "../../testdata/server.key"
	clientCert   = "../../testdata/client.crt"
	clientKey    = "../../testdata/client.key"
	notExistCert = "not_exist.crt"
	notExistKey  = "not_exist.key"
)

func TestGetServerConfig(t *testing.T) {

	t.Run("Single cert and key file", func(t *testing.T) {
		_, err := itls.GetServerConfig(caPem, serverCert, serverKey)
		assert.NoError(t, err)
		_, err = itls.GetServerConfig("", serverCert, serverKey)
		assert.NoError(t, err)
		_, err = itls.GetServerConfig("", "", "")
		assert.Error(t, err)

		// Multiple ca files.
		_, err = itls.GetServerConfig(
			strings.Join([]string{caPem, caPem}, tlsFileSeparator), serverCert, serverKey)
		assert.NoError(t, err)
		_, err = itls.GetServerConfig(
			strings.Join([]string{caPem, "root"}, tlsFileSeparator), serverCert, serverKey)
		assert.Error(t, err)
		_, err = itls.GetServerConfig(
			strings.Join([]string{caPem, "none"}, tlsFileSeparator), serverCert, serverKey)
		assert.Error(t, err)
		_, err = itls.GetServerConfig(
			strings.Join([]string{caPem, notExistPem}, tlsFileSeparator), serverCert, serverKey)
		assert.Error(t, err)
	})

	t.Run("Files not have the same length", func(t *testing.T) {
		for _, ca := range []string{caPem, ""} {
			_, err := itls.GetServerConfig(ca,
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				serverKey)
			assert.Error(t, err)
		}
	})
	t.Run("Files not exist", func(t *testing.T) {
		for _, ca := range []string{caPem, ""} {
			_, err := itls.GetServerConfig(ca,
				strings.Join([]string{serverCert, notExistCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, notExistKey}, tlsFileSeparator),
			)
			assert.Error(t, err)
		}
	})
	t.Run("Multiple files normal case", func(t *testing.T) {
		for _, ca := range []string{caPem, ""} {
			_, err := itls.GetServerConfig(ca,
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
			)
			assert.NoError(t, err)
		}
	})
}

func TestGetClientConfig(t *testing.T) {
	const localhost = "localhost"
	t.Run("Single cert and key file", func(t *testing.T) {
		_, err := itls.GetClientConfig(localhost, caPem, clientCert, clientKey)
		assert.NoError(t, err)
		_, err = itls.GetClientConfig(localhost, notExistPem, clientCert, clientKey)
		assert.Error(t, err)
		_, err = itls.GetClientConfig(localhost, "none", clientCert, clientKey)
		assert.NoError(t, err)
		_, err = itls.GetClientConfig(localhost, caPem, "", "")
		assert.NoError(t, err)
		_, err = itls.GetClientConfig(localhost, caPem, notExistCert, notExistKey)
		assert.Error(t, err)
		_, err = itls.GetClientConfig(localhost, "root", "", "")
		assert.NoError(t, err)

		// Multiple ca files.
		_, err = itls.GetClientConfig(localhost,
			strings.Join([]string{caPem, caPem}, tlsFileSeparator), clientCert, clientKey)
		assert.NoError(t, err)
		_, err = itls.GetClientConfig(localhost,
			strings.Join([]string{caPem, "root"}, tlsFileSeparator), clientCert, clientKey)
		assert.Error(t, err)
		_, err = itls.GetClientConfig(localhost,
			strings.Join([]string{caPem, "none"}, tlsFileSeparator), clientCert, clientKey)
		assert.Error(t, err)
		_, err = itls.GetClientConfig(localhost,
			strings.Join([]string{caPem, notExistPem}, tlsFileSeparator), clientCert, clientKey)
		assert.Error(t, err)
	})

	t.Run("Files not have the same length", func(t *testing.T) {
		for _, ca := range []string{caPem, "root", "none"} {
			_, err := itls.GetClientConfig(localhost, ca,
				strings.Join([]string{clientCert, clientCert}, tlsFileSeparator),
				clientKey)
			assert.Error(t, err)
		}
	})
	t.Run("Files not exist", func(t *testing.T) {
		for _, ca := range []string{caPem, "root", "none"} {
			_, err := itls.GetClientConfig(localhost, ca,
				strings.Join([]string{clientCert, notExistCert}, tlsFileSeparator),
				strings.Join([]string{clientKey, notExistKey}, tlsFileSeparator),
			)
			assert.Error(t, err)
		}
	})
	t.Run("Multiple files normal case", func(t *testing.T) {
		for _, ca := range []string{caPem, "root", "none"} {
			_, err := itls.GetClientConfig(localhost, ca,
				strings.Join([]string{clientCert, clientCert}, tlsFileSeparator),
				strings.Join([]string{clientKey, clientKey}, tlsFileSeparator),
			)
			assert.NoError(t, err)
		}
	})
}

func TestMayLiftToTLSListener(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Log(err)
		}
	})
	t.Run("No TLS configuration files provided", func(t *testing.T) {
		newLn, err := itls.MayLiftToTLSListener(ln, "", "", "")
		require.NoErrorf(t, err, "unexpected error: %v", err)
		require.Equalf(t, ln, newLn, "expected original listener, got %v", newLn)
	})
	t.Run("Valid TLS configuration files provided", func(t *testing.T) {
		newLn, err := itls.MayLiftToTLSListener(ln, caPem, serverCert, serverKey)
		require.NoErrorf(t, err, "expected error, got %v", err)
		_, ok := newLn.(net.Listener)
		require.Truef(t, ok, "expected TLS listener, got %v, type: %T", newLn, newLn)
	})
	t.Run("Invalid TLS configuration files provided", func(t *testing.T) {
		newLn, err := itls.MayLiftToTLSListener(ln, notExistPem, notExistCert, notExistKey)
		require.Error(t, err, "expected error, got nil")
		require.Nil(t, newLn)
	})

	t.Run("Files not have the same length", func(t *testing.T) {
		for _, ca := range []string{"", caPem} {
			_, err := itls.MayLiftToTLSListener(ln, ca,
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				serverKey)
			assert.Error(t, err)
		}
	})
	t.Run("Files not exist", func(t *testing.T) {
		for _, ca := range []string{"", caPem} {
			_, err := itls.MayLiftToTLSListener(ln, ca,
				strings.Join([]string{serverCert, notExistCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, notExistKey}, tlsFileSeparator),
			)
			assert.Error(t, err)
		}
	})
	t.Run("Multiple files normal case", func(t *testing.T) {
		for _, ca := range []string{caPem, ""} {
			_, err := itls.MayLiftToTLSListener(ln, ca,
				strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
				strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
			)
			assert.NoError(t, err)
		}
		_, err := itls.MayLiftToTLSListener(ln, notExistPem,
			strings.Join([]string{serverCert, serverCert}, tlsFileSeparator),
			strings.Join([]string{serverKey, serverKey}, tlsFileSeparator),
		)
		assert.Error(t, err)
	})
}
