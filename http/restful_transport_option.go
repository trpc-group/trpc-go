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

// RESTServerTransportOption modifies ServerTransport.
type RESTServerTransportOption func(*restServerTransportOptions)

type restServerTransportOptions struct {
	reusePort bool
}

// WithReusePort returns an RESTServerTransportOption which enables reuse port.
func WithReusePort() RESTServerTransportOption {
	return func(o *restServerTransportOptions) {
		o.reusePort = true
	}
}
