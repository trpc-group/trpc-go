package metrics

// NoopSink defines the noop Sink.
type NoopSink struct{}

// Name returns noop.
func (n *NoopSink) Name() string {
	return "noop"
}

// Report does nothing.
func (n *NoopSink) Report(rec Record, opts ...Option) error {
	return nil
}
