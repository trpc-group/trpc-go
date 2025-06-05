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

package multiplexed

import "trpc.group/trpc-go/trpc-go/codec"

// GetOptions gets conn configuration.
type GetOptions struct {
	FramerBuilder codec.FramerBuilder
	Msg           codec.Msg

	CACertFile    string // CA certificate.
	TLSCertFile   string // Client certificate.
	TLSKeyFile    string // Client secret key.
	TLSServerName string // The client verifies the server's service name,
	// if not filled in, it defaults to the http hostname.

	LocalAddr string

	network       string
	address       string
	virtualConnID uint32
	isStream      bool
	nodeKey       string
}

// NewGetOptions creates GetOptions.
func NewGetOptions() GetOptions {
	return GetOptions{}
}

// WithFramerBuilder returns an Option which sets the FramerBuilder.
func (o *GetOptions) WithFramerBuilder(fb codec.FramerBuilder) {
	o.FramerBuilder = fb
}

// WithDialTLS returns an Option which sets the client to support TLS.
func (o *GetOptions) WithDialTLS(certFile, keyFile, caFile, serverName string) {
	o.TLSCertFile = certFile
	o.TLSKeyFile = keyFile
	o.CACertFile = caFile
	o.TLSServerName = serverName
}

// WithMsg returns an Option which sets Msg.
func (o *GetOptions) WithMsg(msg codec.Msg) {
	o.Msg = msg
}

// WithLocalAddr returns an Option which sets the local address when
// establishing a connection, and it is randomly selected by default
// when there are multiple network cards.
func (o *GetOptions) WithLocalAddr(addr string) {
	o.LocalAddr = addr
}

func (o *GetOptions) update(network, address string) error {
	o.virtualConnID = o.Msg.RequestID()
	if o.FramerBuilder == nil {
		return ErrFrameBuilderNil
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
