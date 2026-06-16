//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

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
