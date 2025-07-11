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

package multiplexed

// GetOptions get conn configuration.
type GetOptions struct {
	FP  FrameParser
	VID uint32

	CACertFile    string // CA certificate.
	TLSCertFile   string // Client certificate.
	TLSKeyFile    string // Client secret key.
	TLSServerName string // The client verifies the server's service name,
	// if not filled in, it defaults to the http hostname.

	LocalAddr string

	network  string
	address  string
	isStream bool
	nodeKey  string
}

// NewGetOptions creates GetOptions.
func NewGetOptions() GetOptions {
	return GetOptions{}
}

// WithFrameParser sets the FrameParser of a single Get.
func (o *GetOptions) WithFrameParser(fp FrameParser) {
	o.FP = fp
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

func (o *GetOptions) update(network, address string) error {
	if o.FP == nil {
		return ErrFrameParserNil
	}
	isStream, err := isStream(network)
	if err != nil {
		return err
	}
	o.isStream = isStream
	o.address = address
	o.network = network
	o.nodeKey = makeNodeKey(o.network, o.address)
	return nil
}
