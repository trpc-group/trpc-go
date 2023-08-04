package rpcz

// SpanID is a unique identity of a span.
// a valid span ID is always a non-negative integer
type SpanID int64

// nilSpanID is a invalid span ID.
const nilSpanID SpanID = -1
