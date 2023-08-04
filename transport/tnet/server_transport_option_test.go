//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func TestSetNumPollers(t *testing.T) {
	err := tnettrans.SetNumPollers(2)
	assert.Nil(t, err)
}

func TestOptions(t *testing.T) {
	opts := &tnettrans.ServerTransportOptions{}
	tnettrans.WithKeepAlivePeriod(time.Second)(opts)
	assert.Equal(t, time.Second, opts.KeepAlivePeriod)
}
