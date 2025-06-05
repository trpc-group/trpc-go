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
	"net"
	"os"
	"strings"
)

// tlsFileSeparator is the file separator used for parsing TLS configuration files.
// The colon character is reserved in macOS and Windows for domain names, so we use it here.
const tlsFileSeparator = ":"

// MayLiftToTLSListener takes a listener and optional TLS configuration files, and returns
// a new listener that supports TLS encryption. If no TLS configuration files are provided,
// the original listener is returned. If provided, the function creates a new TLS listener
// using the configuration files and returns it for encrypted connections.
// The tlsCertFile and tlsKeyFile parameters support multiple file paths,
// with each file path separated by a colon `:`(tlsFileSeparator) and no spaces in between.
// For example:
//
//	caCertFile = "caA.pem:caB.pem"
//	tlsCertFile = "a.crt:b.crt"
//	tlsKeyFile = "a.key:b.key"
func MayLiftToTLSListener(ln net.Listener, caCertFile, tlsCertFile, tlsKeyFile string) (net.Listener, error) {
	if len(tlsKeyFile) == 0 || len(tlsCertFile) == 0 {
		return ln, nil
	}
	tlsConf, err := GetServerConfig(caCertFile, tlsCertFile, tlsKeyFile)
	if err != nil {
		return nil, fmt.Errorf("tls get server config failed: %w", err)
	}
	return tls.NewListener(ln, tlsConf), nil
}

// LoadTLSKeyPairs loads multiple TLS key pairs from the provided certificate and key files.
// The certFile and keyFile parameters should be strings containing file paths separated by tlsFileSeparator.
// The function returns a slice of tls.Certificate or an error if any of the files cannot be loaded.
func LoadTLSKeyPairs(certFile, keyFile string) ([]tls.Certificate, error) {
	certFiles := strings.Split(certFile, tlsFileSeparator)
	keyFiles := strings.Split(keyFile, tlsFileSeparator)
	// Files should be the same length.
	if len(certFiles) != len(keyFiles) {
		return nil, fmt.Errorf("cert file and key files should have the same length, but have %d and %d",
			len(certFiles), len(keyFiles))
	}

	certs := make([]tls.Certificate, 0, len(certFiles))
	for i := range certFiles {
		cert, err := tls.LoadX509KeyPair(certFiles[i], keyFiles[i])
		if err != nil {
			return nil, fmt.Errorf("tls load cert file{i: %d, cert: %s, key: %s} error: %w",
				i, certFiles[i], keyFiles[i], err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// GetServerConfig gets TLS config for server.
// If you do not need to verify the client's certificate, set the caCertFile to empty.
// CertFile and keyFile should not be empty.
// The certFile and keyFile parameters support multiple file paths,
// with each file path separated by a colon `:`(tlsFileSeparator) and no spaces in between.
// For example:
//
//	caCertFile = "caA.pem:caB.pem"
//	certFile = "a.crt:b.crt"
//	keyFile = "a.key:b.key"
func GetServerConfig(caCertFile, certFile, keyFile string) (*tls.Config, error) {
	certs, err := LoadTLSKeyPairs(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("server load cert file error: %w", err)
	}

	tlsConf := &tls.Config{
		Certificates: certs,
	}
	// Unnecessary to verify client certificate.
	if caCertFile == "" {
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
// The certFile and keyFile parameters support multiple file paths,
// with each file path separated by a colon `:`(tlsFileSeparator) and no spaces in between.
// For example:
//
//	caCertFile = "caA.pem:caB.pem"
//	certFile = "a.crt:b.crt"
//	keyFile = "a.key:b.key"
func GetClientConfig(serverName, caCertFile, certFile, keyFile string) (*tls.Config, error) {
	tlsConf := &tls.Config{}
	if caCertFile == "none" {
		// Unnecessary to verify server certificate.
		tlsConf.InsecureSkipVerify = true
	} else {
		// Necessary to verify server certification.
		certPool, err := GetCertPool(caCertFile)
		if err != nil {
			return nil, err
		}

		tlsConf.RootCAs = certPool
	}
	tlsConf.ServerName = serverName
	if certFile == "" || keyFile == "" {
		return tlsConf, nil
	}
	// Enable two-way authentication and needs to send the
	// client's own certificate to the server.
	certs, err := LoadTLSKeyPairs(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("client load cert file error: %w", err)
	}
	tlsConf.Certificates = certs
	return tlsConf, nil
}

// GetCertPool gets CertPool information.
func GetCertPool(caCertFile string) (*x509.CertPool, error) {
	// root means to use the root ca certificate installed on the machine to
	// verify the peer, if not root, use the input ca file to verify peer.
	if caCertFile == "root" {
		return nil, nil
	}
	if caCertFile == "" {
		return nil, errors.New("caCertFile is empty")
	}

	certs := strings.Split(caCertFile, tlsFileSeparator)
	certPool := x509.NewCertPool()
	for i, cert := range certs {
		c, err := os.ReadFile(cert)
		if err != nil {
			return nil, fmt.Errorf("read cert file{i: %d, ca: %s} error: %w", i, cert, err)
		}
		if !certPool.AppendCertsFromPEM(c) {
			return nil, fmt.Errorf("append certs file{i: %d, ca: %s} from PEM failed", i, cert)
		}
	}
	return certPool, nil
}
