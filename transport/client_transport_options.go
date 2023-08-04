package transport

// ClientTransportOptions is the client transport options.
type ClientTransportOptions struct {
	DisableHTTPEncodeTransInfoBase64 bool
}

// ClientTransportOption modifies the ClientTransportOptions.
type ClientTransportOption func(*ClientTransportOptions)

// WithDisableEncodeTransInfoBase64 returns a ClientTransportOption indicates disable
// encoding the transinfo value by base64 in HTTP.
func WithDisableEncodeTransInfoBase64() ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.DisableHTTPEncodeTransInfoBase64 = true
	}
}
