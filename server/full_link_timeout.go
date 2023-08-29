// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package server

import (
	"context"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
)

// mayConvert2FullLinkTimeout infers whether an error is caused by a full-link
// timeout. If so, it returns the full-link timeout error.
func mayConvert2FullLinkTimeout(
	ctx context.Context,
	req interface{},
	next filter.ServerHandleFunc,
) (interface{}, error) {
	rsp, err := next(ctx, req)
	if e, ok := err.(*errs.Error); ok &&
		e.IsTimeout(errs.ErrorTypeFramework) &&
		e.Code != errs.RetClientTimeout {
		e.Code = errs.RetServerFullLinkTimeout
	}
	return rsp, err
}

// mayConvert2NormalTimeout infers whether an error is caused by a server
// timeout. If so, it returns the server timeout error.
func mayConvert2NormalTimeout(
	ctx context.Context,
	req interface{},
	next filter.ServerHandleFunc,
) (interface{}, error) {
	rsp, err := next(ctx, req)
	if e, ok := err.(*errs.Error); ok &&
		e.IsTimeout(errs.ErrorTypeFramework) &&
		e.Code != errs.RetClientTimeout {
		e.Code = errs.RetServerTimeout
	}
	return rsp, err
}
