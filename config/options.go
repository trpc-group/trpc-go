package config

// WithCodec returns an option which sets the codec's name.
func WithCodec(name string) LoadOption {
	return func(c *TrpcConfig) {
		c.decoder = GetCodec(name)
	}
}

// WithProvider returns an option which sets the provider's name.
func WithProvider(name string) LoadOption {
	return func(c *TrpcConfig) {
		c.p = GetProvider(name)
	}
}

// options is config option.
type options struct{}

// Option is the option for config provider sdk.
type Option func(*options)
