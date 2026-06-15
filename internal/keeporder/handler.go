package keeporder

import (
	"context"
)

// PreDecodeHandler extends the Handler interface to provide PreDecode functionality.
// It is typically used by the keep-order feature.
type PreDecodeHandler interface {
	PreDecode(ctx context.Context, reqBuf []byte) (reqBodyBuf []byte, err error)
}

// PreUnmarshalHandler extends the Handler interface to provide PreUnmarshal functionality.
// It is typically used by the keep-order feature.
type PreUnmarshalHandler interface {
	PreUnmarshal(ctx context.Context, reqBuf []byte) (reqBody interface{}, err error)
}
