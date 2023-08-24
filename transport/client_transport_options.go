// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package transport

// ClientTransportOptions is the client transport options.
type ClientTransportOptions struct {
	DisableHTTPEncodeTransInfoBase64 bool
}

// ClientTransportOption modifies the ClientTransportOptions.
type ClientTransportOption func(*ClientTransportOptions)

// WithDisableEncodeTransInfoBase64 returns a ClientTransportOption indicates disable
// encoding the transinfo value by base64 in HTTP.
func WithDisableEncodeTransInfoBase64() ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.DisableHTTPEncodeTransInfoBase64 = true
	}
}
