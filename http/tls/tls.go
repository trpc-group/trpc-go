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

// Package tls supports custom certificate providers for HTTP TLS.
package tls

import "sync"

// CertificateProvider loads certificate and key material by ID.
type CertificateProvider interface {
	LoadCertFile(certID string) ([]byte, error)
	LoadKeyFile(keyID string) ([]byte, error)
}

var providers sync.Map

// RegisterCertificateProvider registers a CertificateProvider by name.
func RegisterCertificateProvider(name string, provider CertificateProvider) {
	providers.Store(name, provider)
}

// UnregisterCertificateProvider unregisters a CertificateProvider by name.
func UnregisterCertificateProvider(name string) {
	providers.Delete(name)
}

// GetCertificateProvider returns a registered CertificateProvider by name.
func GetCertificateProvider(name string) CertificateProvider {
	provider, ok := providers.Load(name)
	if !ok {
		return nil
	}
	return provider.(CertificateProvider)
}
