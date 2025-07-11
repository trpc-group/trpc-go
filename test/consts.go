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

package test

// The constants required for end-to-end testing.
const (
	defaultConfigPath = "./trpc_go.yaml"

	trpcServiceName      = "trpc.testing.end2end.TestTRPC"
	streamingServiceName = "trpc.testing.end2end.TestStreaming"
	httpServiceName      = "trpc.testing.end2end.TestHTTP"

	defaultServerAddress   = "localhost:0"
	defaultAdminListenAddr = "127.0.0.1:9028"

	// retUnsupportedPayload is the return code for unsupported payload type.
	retUnsupportedPayload = 1101

	validUserNameForAuth = "trpc-go-end2end-testing"
)
