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

// Package tls provides some utility functions to get TLS config.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	httptls "trpc.group/trpc-go/trpc-go/http/tls"
)

const (
	defaultProviderName = "default"
	tlsFileSeparator    = ":"
)

type defaultProvider struct{}

func (defaultProvider) LoadCertFile(certID string) ([]byte, error) {
	return os.ReadFile(certID)
}

func (defaultProvider) LoadKeyFile(keyID string) ([]byte, error) {
	return os.ReadFile(keyID)
}

func init() {
	httptls.RegisterCertificateProvider(defaultProviderName, defaultProvider{})
}

// GetServerConfig gets TLS config for server.
// If you do not need to verify the client's certificate, set the caCertFile to empty.
// CertFile and keyFile should not be empty.
func GetServerConfig(caCertFile, certFile, keyFile, providerName string) (*tls.Config, error) {
	tlsConf := &tls.Config{}
	certs, err := LoadTLSKeyPairs(certFile, keyFile, providerName)
	if err != nil {
		return nil, fmt.Errorf("server load cert file error: %w", err)
	}
	tlsConf.Certificates = certs

	if caCertFile == "" { // no need to verify client certificate.
		return tlsConf, nil
	}
	tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
	pool, err := GetCertPool(caCertFile, providerName)
	if err != nil {
		return nil, err
	}
	tlsConf.ClientCAs = pool
	return tlsConf, nil
}

// GetClientConfig gets TLS config for client.
// If you do not need to verify the server's certificate, set the caCertFile to "none".
// If only one-way authentication, set the certFile and keyFile to empty.
func GetClientConfig(serverName, caCertFile, certFile, keyFile, providerName string) (*tls.Config, error) {
	tlsConf := &tls.Config{}
	if caCertFile == "none" { // no need to verify server certificate.
		tlsConf.InsecureSkipVerify = true
	} else {
		// need to verify server certification.
		tlsConf.ServerName = serverName
		certPool, err := GetCertPool(caCertFile, providerName)
		if err != nil {
			return nil, err
		}
		tlsConf.RootCAs = certPool
	}
	if certFile == "" || keyFile == "" {
		return tlsConf, nil
	}
	// enable two-way authentication and needs to send the
	// client's own certificate to the server.
	certs, err := LoadTLSKeyPairs(certFile, keyFile, providerName)
	if err != nil {
		return nil, fmt.Errorf("client load cert file error: %w", err)
	}
	tlsConf.Certificates = certs
	return tlsConf, nil
}

// LoadTLSKeyPairs loads X.509 key pairs from the provider.
func LoadTLSKeyPairs(certFile, keyFile, providerName string) ([]tls.Certificate, error) {
	provider, err := getProvider(providerName)
	if err != nil {
		return nil, err
	}
	certs := strings.Split(certFile, tlsFileSeparator)
	keys := strings.Split(keyFile, tlsFileSeparator)
	if len(certs) != len(keys) {
		return nil, errors.New("certificate file count does not match key file count")
	}

	certificates := make([]tls.Certificate, 0, len(certs))
	for i := range certs {
		certPEMBlock, err := provider.LoadCertFile(certs[i])
		if err != nil {
			return nil, fmt.Errorf("load cert file error: %w", err)
		}
		keyPEMBlock, err := provider.LoadKeyFile(keys[i])
		if err != nil {
			return nil, fmt.Errorf("load key file error: %w", err)
		}
		cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, cert)
	}
	return certificates, nil
}

// GetCertPool gets CertPool information.
func GetCertPool(caCertFile, providerName string) (*x509.CertPool, error) {
	// root means to use the root ca certificate installed on the machine to
	// verify the peer, if not root, use the input ca file to verify peer.
	if caCertFile == "root" {
		return nil, nil
	}
	provider, err := getProvider(providerName)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	for _, file := range strings.Split(caCertFile, tlsFileSeparator) {
		ca, err := provider.LoadCertFile(file)
		if err != nil {
			return nil, fmt.Errorf("read ca file error: %w", err)
		}
		if !certPool.AppendCertsFromPEM(ca) {
			return nil, errors.New("AppendCertsFromPEM fail")
		}
	}
	return certPool, nil
}

func getProvider(providerName string) (httptls.CertificateProvider, error) {
	if providerName == "" {
		providerName = defaultProviderName
	}
	provider := httptls.GetCertificateProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("tls certificate provider %q not found", providerName)
	}
	return provider, nil
}
