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

// Package tls provides some utility functions to get TLS config.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// GetServerConfig gets TLS config for server.
// If you do not need to verify the client's certificate, set the caCertFile to empty.
// CertFile and keyFile should not be empty.
func GetServerConfig(caCertFile, certFile, keyFile string) (*tls.Config, error) {
	tlsConf := &tls.Config{}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("server load cert file error: %w", err)
	}
	tlsConf.Certificates = []tls.Certificate{cert}

	if caCertFile == "" { // no need to verify client certificate.
		return tlsConf, nil
	}
	tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
	pool, err := GetCertPool(caCertFile)
	if err != nil {
		return nil, err
	}
	tlsConf.ClientCAs = pool
	return tlsConf, nil
}

// GetClientConfig gets TLS config for client.
// If you do not need to verify the server's certificate, set the caCertFile to "none".
// If only one-way authentication, set the certFile and keyFile to empty.
func GetClientConfig(serverName, caCertFile, certFile, keyFile string) (*tls.Config, error) {
	tlsConf := &tls.Config{}
	if caCertFile == "none" { // no need to verify server certificate.
		tlsConf.InsecureSkipVerify = true
		return tlsConf, nil
	}
	// need to verify server certification.
	tlsConf.ServerName = serverName
	certPool, err := GetCertPool(caCertFile)
	if err != nil {
		return nil, err
	}
	tlsConf.RootCAs = certPool
	if certFile == "" {
		return tlsConf, nil
	}
	// enable two-way authentication and needs to send the
	// client's own certificate to the server.
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("client load cert file error: %w", err)
	}
	tlsConf.Certificates = []tls.Certificate{cert}
	return tlsConf, nil
}

// GetCertPool gets CertPool information.
func GetCertPool(caCertFile string) (*x509.CertPool, error) {
	// root means to use the root ca certificate installed on the machine to
	// verify the peer, if not root, use the input ca file to verify peer.
	if caCertFile == "root" {
		return nil, nil
	}
	ca, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("read ca file error: %w", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(ca) {
		return nil, errors.New("AppendCertsFromPEM fail")
	}
	return certPool, nil
}
