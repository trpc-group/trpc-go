package registry

// Options is the node register options.
type Options struct {
	Address string
	Event   EventType
}

// Option modifies the Options.
type Option func(*Options)

// WithAddress returns an Option which sets the server address. The format of address is "IP:Port" or
// just ":Port".
func WithAddress(s string) Option {
	return func(opts *Options) {
		opts.Address = s
	}
}

// EventType defines the event types.
type EventType int

// GracefulRestart represents the hot restart event.
const GracefulRestart = EventType(iota)

// WithEvent returns an Option which sets the event type.
func WithEvent(e EventType) Option {
	return func(opts *Options) {
		opts.Event = e
	}
}
