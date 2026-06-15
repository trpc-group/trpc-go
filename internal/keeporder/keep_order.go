// Package keeporder offers internal utilities for maintaining request order.
package keeporder

import (
	"context"
)

// PreDecodeExtractor defines a function type that extracts a key which is used to maintain the order of requests
// from the decoded results and the raw request body.
//
// It returns a keep-order key and a boolean.
//
// If the boolean is false, the keep-order feature is disabled for the request.
//
// When enabled, requests sharing the same keep-order key are processed serially within the same group.
// Requests from different groups, identified by different keys, are processed in parallel.
type PreDecodeExtractor func(ctx context.Context, reqBody []byte) (string, bool)

// PreUnmarshalExtractor defines a function type that extracts a key which is used to maintain the order of requests
// from the unmarshalled request body.
//
// It returns a keep-order key and a boolean.
//
// If the boolean is false, the keep-order feature is disabled for the request.
//
// When enabled, requests sharing the same keep-order key are processed serially within the same group.
// Requests from different groups, identified by different keys, are processed in parallel.
type PreUnmarshalExtractor func(ctx context.Context, reqBody interface{}) (string, bool)
