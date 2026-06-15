package transport

import "context"

// KeepOrderPreDecodeExtractor extracts a key for keeping request order from the decoded request body.
type KeepOrderPreDecodeExtractor func(ctx context.Context, reqBody []byte) (string, bool)

// KeepOrderPreUnmarshalExtractor extracts a key for keeping request order from the unmarshaled request body.
type KeepOrderPreUnmarshalExtractor func(ctx context.Context, reqBody interface{}) (string, bool)

// OrderedGroups keeps requests ordered by key.
type OrderedGroups interface {
	Add(key string, fn func())
	Remove(key string)
	Stop()
}
