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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/http/tls"
	"trpc.group/trpc-go/trpc-go/log"
)

// SimpleCertProvider is a minimal example of implementing a custom certificate provider
type SimpleCertProvider struct{}

// LoadCertFile implements the CertificateProvider interface
func (s *SimpleCertProvider) LoadCertFile(certID string) ([]byte, error) {
	log.Infof("SimpleCertProvider loading certificate file: certID=%q (empty=%v)", certID, certID == "")

	// Handle special IDs - simulate KMS/vault behavior
	var actualPath string
	switch certID {
	case "":
		log.Info("certID is empty, using default client certificate")
		actualPath = "../../../../testdata/client.crt"
	case "my-client-cert", "my-client-cert-2":
		log.Infof("Recognized custom cert ID: %s, fetching from KMS...", certID)
		actualPath = "../../../../testdata/client.crt"
	default:
		// Treat as file path
		actualPath = certID
	}

	// In this simple example, we just load from files
	// In production, you would:
	// 1. Fetch from your secure storage (KMS, vault, database, etc.)
	// 2. Decrypt if necessary
	// 3. Apply access control checks
	// 4. Log audit events
	// 5. Implement caching and rotation

	certPEM, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert from %s (mapped from %s): %w", actualPath, certID, err)
	}

	// Here you could decrypt or transform the data before returning
	// certPEM = decrypt(certPEM)

	log.Infof("Certificate loaded successfully via SimpleCertProvider (certID=%q, size=%d bytes)", certID, len(certPEM))
	return certPEM, nil
}

// LoadKeyFile implements the CertificateProvider interface
func (s *SimpleCertProvider) LoadKeyFile(keyID string) ([]byte, error) {
	log.Infof("SimpleCertProvider loading key file: keyID=%q (empty=%v)", keyID, keyID == "")

	// Handle special IDs - simulate KMS/vault behavior
	var actualPath string
	switch keyID {
	case "":
		log.Info("keyID is empty, using default client key")
		actualPath = "../../../../testdata/client.key"
	case "my-client-key", "my-client-key-2":
		log.Infof("Recognized custom key ID: %s, fetching from KMS and decrypting...", keyID)
		actualPath = "../../../../testdata/client.key"
	default:
		// Treat as file path
		actualPath = keyID
	}

	keyPEM, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key from %s (mapped from %s): %w", actualPath, keyID, err)
	}

	// Here you could decrypt the key
	// keyPEM = decrypt(keyPEM)

	log.Infof("Key loaded successfully via SimpleCertProvider (keyID=%q, size=%d bytes)", keyID, len(keyPEM))
	return keyPEM, nil
}

// KMSCertProvider demonstrates integration with Key Management Service
type KMSCertProvider struct {
	kmsEndpoint string
}

// LoadCertFile fetches certificate from KMS
func (k *KMSCertProvider) LoadCertFile(certID string) ([]byte, error) {
	log.Infof("KMSCertProvider loading cert from %s: cert=%s", k.kmsEndpoint, certID)

	// In production, you would:
	// 1. Call KMS API to fetch certificate
	// 2. Authenticate using service credentials
	// 3. Handle caching

	// For demonstration, fall back to file loading
	certPEM, err := os.ReadFile(certID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cert from KMS: %w", err)
	}

	log.Infof("Certificate loaded successfully from KMS")
	return certPEM, nil
}

// LoadKeyFile fetches private key from KMS
func (k *KMSCertProvider) LoadKeyFile(keyID string) ([]byte, error) {
	log.Infof("KMSCertProvider loading key from %s: key=%s", k.kmsEndpoint, keyID)

	// In production, you would:
	// 1. Call KMS API to fetch and decrypt the key
	// 2. The key might never leave the KMS in plaintext
	// 3. Handle key rotation

	// For demonstration, fall back to file loading
	keyPEM, err := os.ReadFile(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch key from KMS: %w", err)
	}

	log.Infof("Key loaded successfully from KMS")
	return keyPEM, nil
}

func init() {
	// Register multiple certificate providers with different names
	// This demonstrates the registry pattern where each provider has a unique name

	simpleProvider := &SimpleCertProvider{}
	tls.RegisterCertificateProvider("simple", simpleProvider)
	log.Info("Registered certificate provider: 'simple'")

	kmsProvider := &KMSCertProvider{kmsEndpoint: "https://kms.example.com"}
	tls.RegisterCertificateProvider("kms", kmsProvider)
	log.Info("Registered certificate provider: 'kms'")
}

func main() {
	log.Info("=== Testing Custom Certificate Provider with client.WithCertProvider ===")
	log.Info("This example demonstrates using client.WithCertProvider option")
	log.Info("Please start the HTTPS server first:")
	log.Info("  cd ../server && go run main.go")
	log.Info("")

	// Example 1: Using Simple Certificate Provider with normal paths
	log.Info("--- Example 1: Using 'simple' Provider with normal paths ---")
	makeHTTPSRequestWithOptions("simple", "../../../../testdata/client.crt", "../../../../testdata/client.key")

	// Example 2: Using Simple Certificate Provider with EMPTY paths
	log.Info("\n--- Example 2: Using 'simple' Provider with EMPTY paths (provider handles default) ---")
	makeHTTPSRequestWithOptions("simple", "", "")

	// Example 3: Using Simple Certificate Provider with MULTIPLE paths
	log.Info("\n--- Example 3: Using 'simple' Provider with MULTIPLE paths (xxx:xxx) ---")
	makeHTTPSRequestWithOptions("simple",
		"../../../../testdata/client.crt:../../../../testdata/client.crt",
		"../../../../testdata/client.key:../../../../testdata/client.key")

	// Example 4: Using Simple Certificate Provider with NON-FILE custom IDs (xxx:xxx)
	log.Info("\n--- Example 4: Using 'simple' Provider with CUSTOM IDs (my-client-cert:my-client-cert-2) ---")
	makeHTTPSRequestWithOptions("simple",
		"my-client-cert:my-client-cert-2",
		"my-client-key:my-client-key-2")

	// Example 5: Using KMS Certificate Provider
	log.Info("\n--- Example 5: Using 'kms' Provider ---")
	makeHTTPSRequestWithOptions("kms", "../../../../testdata/client.crt", "../../../../testdata/client.key")

	// Example 6: Using default (no provider)
	log.Info("\n--- Example 6: Using default file loading (no provider) ---")
	makeHTTPSRequestWithOptions("", "../../../../testdata/client.crt", "../../../../testdata/client.key")

	log.Info("\n=== All tests completed successfully! ===")
}

func makeHTTPSRequestWithOptions(providerName, certFile, keyFile string) {
	log.Infof("Creating HTTPS client with certFile=%q, keyFile=%q, provider=%q", certFile, keyFile, providerName)

	// Create HTTPS client with certificate provider option
	opts := []client.Option{
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:9443"),
		client.WithTLS(
			certFile,
			keyFile,
			"none",
			"localhost",
		),
	}

	// Add provider option if specified
	if providerName != "" {
		log.Infof("Using certificate provider: %s", providerName)
		opts = append(opts, client.WithCertProvider(providerName))
	} else {
		log.Info("Using default file-based certificate loading")
	}

	httpCli := thttp.NewClientProxy("trpc.app.server.stdhttps", opts...)

	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}
	reqHeader.AddHeader("request", "custom-cert-provider-test")

	rspHead := &thttp.ClientRspHeader{}
	req := &codec.Body{Data: []byte("Hello from custom cert provider client!")}
	rsp := &codec.Body{}

	// Send HTTPS POST request to the existing server
	if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Errorf("Request failed: %v", err)
		return
	}

	log.Infof("Request successful!")
	log.Infof("  Response: %s", string(rsp.Data))
	log.Infof("  Reply header: %s", rspHead.Response.Header.Get("reply"))
}
