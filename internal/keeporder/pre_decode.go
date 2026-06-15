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

import "context"

type preDecodeKey struct{}

// PreDecodeInfo contains request body buffer that is a part of the decoded result.
type PreDecodeInfo struct {
	ReqBodyBuf []byte
}

// NewContextWithPreDecode returns a new context with pre-decoded information.
func NewContextWithPreDecode(ctx context.Context, info *PreDecodeInfo) context.Context {
	return context.WithValue(ctx, preDecodeKey{}, info)
}

// PreDecodeInfoFromContext returns the pre-decoded info from the context.
func PreDecodeInfoFromContext(ctx context.Context) (*PreDecodeInfo, bool) {
	info, ok := ctx.Value(preDecodeKey{}).(*PreDecodeInfo)
	return info, ok
}
