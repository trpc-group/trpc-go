// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOptions(t *testing.T) {

	opts := NewGetOptions()
	fp := &emptyFrameParser{}
	caFile := "caFile"
	keyFile := "keyFile"
	serverName := "serverName"
	certFile := "certFile"
	localAddr := "127.0.0.1:8080"
	var id uint32 = 2

	opts.WithFrameParser(fp)
	opts.WithVID(id)
	opts.WithDialTLS(certFile, keyFile, caFile, serverName)
	opts.WithLocalAddr(localAddr)

	assert.Equal(t, opts.FrameParser, fp)
	assert.Equal(t, opts.VID, id)
	assert.Equal(t, opts.CACertFile, caFile)
	assert.Equal(t, opts.TLSKeyFile, keyFile)
	assert.Equal(t, opts.TLSServerName, serverName)
	assert.Equal(t, opts.TLSCertFile, certFile)
	assert.Equal(t, opts.LocalAddr, localAddr)
}

type emptyFrameParser struct{}

func (efp *emptyFrameParser) Parse(r io.Reader) (vid uint32, buf []byte, err error) {
	return 0, nil, nil
}
