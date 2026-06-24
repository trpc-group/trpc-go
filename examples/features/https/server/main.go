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

// Package main is the server main package for https demo.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/http/tls"
	"trpc.group/trpc-go/trpc-go/log"
)

// CustomCertProvider demonstrates custom provider for server-side certificates
type CustomCertProvider struct{}

// LoadCertFile implements the CertificateProvider interface
func (c *CustomCertProvider) LoadCertFile(certID string) ([]byte, error) {
	log.Infof("CustomCertProvider loading certificate file: certID=%q (empty=%v)", certID, certID == "")

	// Handle special IDs - simulate KMS/vault behavior
	var actualPath string
	switch certID {
	case "":
		log.Info("certID is empty, using default server certificate")
		actualPath = "../../../../testdata/server.crt"
	case "my-server-cert", "my-server-cert-2":
		log.Infof("Recognized custom cert ID: %s, mapping to actual certificate", certID)
		actualPath = "../../../../testdata/server.crt"
	default:
		// Treat as file path
		actualPath = certID
	}

	// In production, you would load from KMS, vault, encrypted storage, etc.
	certPEM, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert from %s (mapped from %s): %w", actualPath, certID, err)
	}

	log.Infof("Server certificate loaded successfully via CustomCertProvider (certID=%q, size=%d bytes)", certID, len(certPEM))
	return certPEM, nil
}

// LoadKeyFile implements the CertificateProvider interface
func (c *CustomCertProvider) LoadKeyFile(keyID string) ([]byte, error) {
	log.Infof("CustomCertProvider loading key file: keyID=%q (empty=%v)", keyID, keyID == "")

	// Handle special IDs - simulate KMS/vault behavior
	var actualPath string
	switch keyID {
	case "":
		log.Info("keyID is empty, using default server key")
		actualPath = "../../../../testdata/server.key"
	case "my-server-key", "my-server-key-2":
		log.Infof("Recognized custom key ID: %s, mapping to actual key (decrypting...)", keyID)
		actualPath = "../../../../testdata/server.key"
	default:
		// Treat as file path
		actualPath = keyID
	}

	// In production, you would decrypt the key from secure storage
	keyPEM, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key from %s (mapped from %s): %w", actualPath, keyID, err)
	}

	log.Infof("Server key loaded successfully via CustomCertProvider (keyID=%q, size=%d bytes)", keyID, len(keyPEM))
	return keyPEM, nil
}

func init() {
	// Register custom certificate provider with a name
	// The server will use this provider to load certificates
	provider := &CustomCertProvider{}
	tls.RegisterCertificateProvider("server-custom", provider)
	log.Info("Custom certificate provider 'server-custom' registered")
}

// handle is a function that processes HTTPS requests.
// Its implementation is consistent with the standard HTTP library.
func handle(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Infof("Received HTTPS request: Method=%s, URL=%s, Body=%s", r.Method, r.URL.String(), string(body))

	// Finally, use 'w' to send the response.
	w.Header().Set("Content-type", "application/text")
	w.Header().Set("reply", "https-response-head")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("https response body"))
	return nil
}

func main() {
	filter.Register("info-request-head", infoReqHead, nil)
	// Init server. Use -conf flag to specify config file: go run main.go -conf trpc_go_with_provider.yaml
	s := trpc.NewServer()

	// Register the handle function for the "/v1/hello" endpoint.
	thttp.HandleFunc("/v1/hello", handle)

	// When registering the NoProtocolService, the parameter passed must match the service name in the configuration.
	// The service name here should be: s.Service("trpc.app.server.stdhttps").
	thttp.RegisterNoProtocolService(s.Service("trpc.app.server.stdhttps"))

	// Start serving and listening.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}

func infoReqHead(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
	msg := codec.Message(ctx)
	rsp, err = next(ctx, req)
	log.Info("ClientReqHead:", msg.ClientReqHead())
	log.Info("ClientRspHead:", msg.ClientRspHead())
	log.Info("ServerReqHead:", msg.ServerReqHead())
	log.Info("ServerRspHead:", msg.ServerRspHead())
	return rsp, err
}
