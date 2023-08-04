package rpcz

// Config stores the config of rpcz.
type Config struct {
	Fraction     float64
	Capacity     uint32
	ShouldRecord ShouldRecord
	Exporter     SpanExporter
}

func (c *Config) shouldRecord() ShouldRecord {
	if c.ShouldRecord == nil {
		return AlwaysRecord
	}
	return c.ShouldRecord
}

// ShouldRecord determines if the Span should be recorded.
type ShouldRecord = func(Span) bool

// AlwaysRecord always records span.
func AlwaysRecord(_ Span) bool { return true }
