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

// Package overloadctrl 定义了过载保护接口。
package overloadctrl

import (
	"context"
)

// OverloadController 定义了过载保护接口。
type OverloadController interface {
	Acquire(ctx context.Context, addr string) (Token, error)
}

// Token 定义了过载保护返回的 token 的接口。
type Token interface {
	OnResponse(ctx context.Context, err error)
}

// NoopOC 是 OverloadController 的空实现。
type NoopOC struct{}

// Acquire 总是放行，并返回一个空 Token。
func (NoopOC) Acquire(context.Context, string) (Token, error) {
	return NoopToken{}, nil
}

// NoopToken 是 Token 的空实现。
type NoopToken struct{}

// OnResponse 什么都不做。
func (NoopToken) OnResponse(context.Context, error) {}

// IsNoop checks whether the given overload controller is noop.
func IsNoop(oc OverloadController) bool {
	if impl, ok := oc.(*Impl); ok {
		if impl.OverloadController == nil {
			return true
		}
		_, ok := impl.OverloadController.(NoopOC)
		return ok
	}
	_, ok := oc.(NoopOC)
	return ok
}
