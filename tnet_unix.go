//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package trpc

import (
	// register tnet transport by default on unix system.
	_ "trpc.group/trpc-go/trpc-go/transport/tnet"
)
