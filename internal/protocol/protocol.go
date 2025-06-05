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

// Package protocol provides name constants for protocols.
package protocol

// Name constants for protocols.
const (
	HTTP            = "http"
	HTTP2           = "http2"
	HTTPS           = "https"
	HTTPNoProtocol  = "http_no_protocol"
	HTTP2NoProtocol = "http2_no_protocol"
	HTTPSNoProtocol = "https_no_protocol"

	FastHTTP           = "fasthttp"
	FastHTTPNoProtocol = "fasthttp_no_protocol"

	TRPC = "trpc"
	TNET = "tnet"

	TCP  = "tcp"
	TCP4 = "tcp4"
	TCP6 = "tcp6"
	UDP  = "udp"
	UDP4 = "udp4"
	UDP6 = "udp6"
	UNIX = "unix"
)
