package rpcz

// SpanExporter Exports ReadOnlySpan to external receivers.
type SpanExporter interface {
	// Export exports the span.
	// Even if the span is exported, a copy of it is stored in GlobalRPCZ.
	Export(span *ReadOnlySpan)
}
