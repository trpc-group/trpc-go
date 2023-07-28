// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

// GetOptions get conn configuration.
type GetOptions struct {
	FrameParser FrameParser
	VID         uint32

	// CA certificate.
	CACertFile string
	// Client certificate.
	TLSCertFile string
	// Client secret key.
	TLSKeyFile string
	// The client verifies the server's service name,
	// if not filled in, it defaults to the http hostname.
	TLSServerName string

	LocalAddr string
}

// NewGetOptions creates GetOptions.
func NewGetOptions() GetOptions {
	return GetOptions{}
}

// WithFrameParser sets the FrameParser of a single Get.
func (o *GetOptions) WithFrameParser(fp FrameParser) {
	o.FrameParser = fp
}

// WithDialTLS returns an Option which sets the client to support TLS.
func (o *GetOptions) WithDialTLS(certFile, keyFile, caFile, serverName string) {
	o.TLSCertFile = certFile
	o.TLSKeyFile = keyFile
	o.CACertFile = caFile
	o.TLSServerName = serverName
}

// WithVID returns an Option which sets virtual connection ID.
func (o *GetOptions) WithVID(vid uint32) {
	o.VID = vid
}

// WithLocalAddr returns an Option which sets the local address when
// establishing a connection, and it is randomly selected by default
// when there are multiple network cards.
func (o *GetOptions) WithLocalAddr(addr string) {
	o.LocalAddr = addr
}
