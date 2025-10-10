//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package http

import (
	"context"
	"errors"

	"github.com/valyala/fasthttp"
	"trpc.group/trpc-go/trpc-go/server"
)

// CtxKey is used to store context.Context in requestCtx.
type CtxKey struct{}

// FastHTTPHandleFunc registers fasthttp handler with custom route.
// If handler need ctx (context.Context), users can get by requestCtx.UserValue(CtxKey{})
func FastHTTPHandleFunc(pattern string, handler func(requestCtx *fasthttp.RequestCtx)) {
	ServiceDesc.Methods = append(ServiceDesc.Methods, generateMethodFastHTTP(pattern, handler))
}

// generateMethod generates server method.
func generateMethodFastHTTP(pattern string, handler fasthttp.RequestHandler) server.Method {
	handlerFunc := func(_ interface{}, ctx context.Context, f server.FilterFunc) (rspBody interface{}, err error) {
		filters, err := f(nil)
		if err != nil {
			return nil, err
		}
		handleFunc := func(ctx context.Context, _ interface{}) (rspBody interface{}, err error) {
			requestCtx := RequestCtx(ctx)
			if requestCtx == nil {
				return nil, errors.New("fasthttp Handle missing requestCtx in context")
			}
			// Store context.Context.
			requestCtx.SetUserValue(CtxKey{}, ctx)
			// Handle error in handler.
			// fasthttp.RequestHandler will NOT return err.
			handler(requestCtx)
			return nil, nil
		}
		return filters.Filter(ctx, nil, handleFunc)
	}
	return server.Method{
		Name: pattern,
		Func: handlerFunc,
	}
}
