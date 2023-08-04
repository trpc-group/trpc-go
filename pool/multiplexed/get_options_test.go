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

	assert.Equal(t, opts.FP, fp)
	assert.Equal(t, opts.VID, id)
	assert.Equal(t, opts.CACertFile, caFile)
	assert.Equal(t, opts.TLSKeyFile, keyFile)
	assert.Equal(t, opts.TLSServerName, serverName)
	assert.Equal(t, opts.TLSCertFile, certFile)
	assert.Equal(t, opts.LocalAddr, localAddr)
}

type emptyFrameParser struct{}

func (efp *emptyFrameParser) Parse(rc io.Reader) (vid uint32, buf []byte, err error) {
	return 0, nil, nil
}
