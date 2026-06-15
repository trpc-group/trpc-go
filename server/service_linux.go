//go:build linux && amd64
// +build linux,amd64

package server

import (
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
	"trpc.group/trpc-go/trpc-go/transport/tnet"
)

func attemptSwitchingTransport(o *Options) transport.ServerTransport {
	// If the user doesn't explicitly set the transport (which is usually the case for trpc protocol),
	// attempt to switch to the tnet transport.
	if o.Transport == nil {
		// Only use tnet transport with TCP and trpc.
		if (o.network == protocol.TCP ||
			o.network == protocol.TCP4 ||
			o.network == protocol.TCP6) &&
			o.protocol == protocol.TRPC {
			log.Infof("service %s with network %s and protocol %s is empowered with tnet! 🤩 "+
				"you can set 'transport: go-net' in your trpc_go.yaml's service configuration "+
				"to switch to the golang net framework",
				o.ServiceName, o.network, o.protocol)
			return tnet.DefaultServerTransport
		}
		log.Infof("service: %s, tnet is not enabled by default for the network %s and protocol %s, 🧐 "+
			"fallback to go-net transport, it is either because tnet does not support them or "+
			"we haven't fully test for some third-party protocols, you can set 'transport: tnet' "+
			"in your service configuration to force using tnet and test it at your own risk",
			o.ServiceName, o.network, o.protocol)
		return transport.DefaultServerStreamTransport
	}
	return o.Transport
}
