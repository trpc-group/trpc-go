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
	"net/http"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Test 1: protocol: https with ca_cert: "none"
	log.Info("=== Test 1: protocol: https with ca_cert: none ===")
	httpsWithCaCert()

	// Test 2: protocol: https without ca_cert (should work with explicit HTTPS)
	log.Info("=== Test 2: protocol: https without ca_cert ===")
	httpsWithoutCaCert()

	// Test 3: protocol: http with ca_cert: "none" (should also work)
	log.Info("=== Test 3: protocol: http with ca_cert: none ===")
	httpWithCaCert()
}

// httpsWithCaCert sends an HTTPS POST request using explicit HTTPS protocol with ca_cert: "none".
func httpsWithCaCert() {
	// Create a ClientProxy with target and TLS enabled via ca_cert: "none".
	httpCli := thttp.NewClientProxy("trpc.app.server.stdhttps",
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:9443"),
		client.WithTLS("", "", "none", "localhost"),
	)

	// Create a ClientReqHeader with the specified HTTP method (POST)
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}

	// Add a custom "request" header to the HTTP request header
	reqHeader.AddHeader("request", "https-test-with-ca-cert")

	// Create ClientRspHeader to store the response header
	rspHead := &thttp.ClientRspHeader{}

	// Create a Body containing the request data
	req := &codec.Body{Data: []byte("Hello, I am HTTPS client with ca_cert!")}

	// Create an empty Body to store the response data
	rsp := &codec.Body{}

	// Send a HTTPS POST request
	if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Errorf("Error getting HTTPS response with ca_cert: %v", err)
		return
	}

	// Get the "reply" field from the HTTP response header
	replyHead := rspHead.Response.Header.Get("reply")
	log.Infof("HTTPS with ca_cert - Data: \"%s\", Response head: \"%s\"", string(rsp.Data), replyHead)
}

// httpsWithoutCaCert sends an HTTPS POST request using explicit HTTPS protocol without ca_cert.
// This should work if explicit HTTPS is properly implemented.
func httpsWithoutCaCert() {
	// Create a ClientProxy with HTTPS but no ca_cert to test behavior.
	httpCli := thttp.NewClientProxy("trpc.app.server.stdhttps-no-ca",
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:9443"),
	)

	// Create a ClientReqHeader with the specified HTTP method (POST)
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}

	// Add a custom "request" header to the HTTP request header
	reqHeader.AddHeader("request", "https-test-without-ca-cert")

	// Create ClientRspHeader to store the response header
	rspHead := &thttp.ClientRspHeader{}

	// Create a Body containing the request data
	req := &codec.Body{Data: []byte("Hello, I am HTTPS client without ca_cert!")}

	// Create an empty Body to store the response data
	rsp := &codec.Body{}

	// Send a HTTPS POST request
	if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Errorf("Error getting HTTPS response without ca_cert: %v", err)
		return
	}

	// Get the "reply" field from the HTTP response header
	replyHead := rspHead.Response.Header.Get("reply")
	log.Infof("HTTPS without ca_cert - Data: \"%s\", Response head: \"%s\"", string(rsp.Data), replyHead)
}

// httpWithCaCert sends an HTTPS POST request using HTTP protocol with ca_cert: "none".
// This should work because ca_cert triggers HTTPS inference.
func httpWithCaCert() {
	// Create a ClientProxy with HTTP protocol but TLS enabled via ca_cert: "none".
	httpCli := thttp.NewClientProxy("trpc.app.server.stdhttps-http-ca",
		client.WithSerializationType(codec.SerializationTypeNoop),
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		client.WithTarget("ip://127.0.0.1:9443"),
		client.WithTLS("", "", "none", "localhost"),
	)

	// Create a ClientReqHeader with the specified HTTP method (POST)
	reqHeader := &thttp.ClientReqHeader{
		Method: http.MethodPost,
	}

	// Add a custom "request" header to the HTTP request header
	reqHeader.AddHeader("request", "http-test-with-ca-cert")

	// Create ClientRspHeader to store the response header
	rspHead := &thttp.ClientRspHeader{}

	// Create a Body containing the request data
	req := &codec.Body{Data: []byte("Hello, I am HTTP client with ca_cert!")}

	// Create an empty Body to store the response data
	rsp := &codec.Body{}

	// Send a HTTPS POST request
	if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
		client.WithReqHead(reqHeader),
		client.WithRspHead(rspHead),
	); err != nil {
		log.Errorf("Error getting HTTP response with ca_cert: %v", err)
		return
	}

	// Get the "reply" field from the HTTP response header
	replyHead := rspHead.Response.Header.Get("reply")
	log.Infof("HTTP with ca_cert - Data: \"%s\", Response head: \"%s\"", string(rsp.Data), replyHead)
}
