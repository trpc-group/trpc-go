//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet

import (
	"runtime"
	"time"

	"trpc.group/trpc-go/tnet"
)

// SetNumPollers sets the number of tnet pollers. Generally it is not actively used.
func SetNumPollers(n int) error {
	return tnet.SetNumPollers(n)
}

// ServerTransportOption is server transport option.
type ServerTransportOption func(o *ServerTransportOptions)

// ServerTransportOptions is server transport options struct.
type ServerTransportOptions struct {
	KeepAlivePeriod time.Duration
	ReusePort       bool
}

// WithKeepAlivePeriod sets the TCP keep alive interval.
func WithKeepAlivePeriod(d time.Duration) ServerTransportOption {
	return func(opts *ServerTransportOptions) {
		opts.KeepAlivePeriod = d
	}
}

// WithReusePort returns a ServerTransportOption which enables reuse port or not.
func WithReusePort(reuse bool) ServerTransportOption {
	return func(opts *ServerTransportOptions) {
		opts.ReusePort = reuse
		if runtime.GOOS == "windows" {
			opts.ReusePort = false
		}
	}
}
