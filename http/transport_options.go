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

package http

// OptServerTransport modifies ServerTransport.
type OptServerTransport func(*ServerTransport)

// WithReusePort returns an OptServerTransport which enables reuse port.
func WithReusePort() OptServerTransport {
	return func(st *ServerTransport) {
		st.reusePort = true
	}
}

// WithEnableH2C returns an OptServerTransport which enables H2C.
func WithEnableH2C() OptServerTransport {
	return func(st *ServerTransport) {
		st.enableH2C = true
	}
}
