package keeporder

import "context"

type preUnmarshalKey struct{}

// PreUnmarshalInfo stores the unmarshaled request body and a boolean indicating
// the current state of the request body.
type PreUnmarshalInfo struct {
	Stored  bool
	ReqBody interface{}
}

// NewContextWithPreUnmarshal creates a new context that carries the provided PreUnmarshalInfo.
func NewContextWithPreUnmarshal(ctx context.Context, info *PreUnmarshalInfo) context.Context {
	return context.WithValue(ctx, preUnmarshalKey{}, info)
}

// PreUnmarshalInfoFromContext return the pre-unmarshal info from the context.
func PreUnmarshalInfoFromContext(ctx context.Context) (*PreUnmarshalInfo, bool) {
	info, ok := ctx.Value(preUnmarshalKey{}).(*PreUnmarshalInfo)
	return info, ok
}
