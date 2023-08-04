package transport

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
)

// DefaultClientTransport is the default client transport.
var DefaultClientTransport = NewClientTransport()

// NewClientTransport creates a new ClientTransport.
func NewClientTransport(opt ...ClientTransportOption) ClientTransport {
	r := newClientTransport(opt...)
	return &r
}

// newClientTransport creates a new clientTransport.
func newClientTransport(opt ...ClientTransportOption) clientTransport {
	// the default options.
	opts := &ClientTransportOptions{}

	// use opt to modify the opts.
	for _, o := range opt {
		o(opts)
	}

	return clientTransport{opts: opts}
}

// clientTransport is the implementation details of client transport, such as tcp/udp roundtrip.
type clientTransport struct {
	// Transport has two kinds of options.
	// One is ClientTransportOptions, which is the option for transport, and is valid for all
	// RoundTrip requests. The framework does not care about the parameters required for specific
	// implementation.
	// The other is RoundTripOptions, which is the option of the current request, such as address,
	// which has different values for different requests. It can be configured and passed in by the
	// upper layer of the framework.
	opts *ClientTransportOptions
}

// RoundTrip sends client requests.
func (c *clientTransport) RoundTrip(ctx context.Context, req []byte,
	roundTripOpts ...RoundTripOption) (rsp []byte, err error) {
	// default value.
	opts := &RoundTripOptions{
		Pool:        connpool.DefaultConnectionPool,
		Multiplexed: multiplexed.DefaultMultiplexedPool,
	}

	// Use roundTripOpts to modify opts.
	for _, o := range roundTripOpts {
		o(opts)
	}

	if opts.EnableMultiplexed {
		return c.multiplexed(ctx, req, opts)
	}

	switch opts.Network {
	case "tcp", "tcp4", "tcp6", "unix":
		return c.tcpRoundTrip(ctx, req, opts)
	case "udp", "udp4", "udp6":
		return c.udpRoundTrip(ctx, req, opts)
	default:
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			fmt.Sprintf("client transport: network %s not support", opts.Network))
	}
}
