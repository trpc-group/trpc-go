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

// Package overloadctrl defines overload control interfaces.
package overloadctrl

import (
	"context"
)

// OverloadController defines an overload controller.
type OverloadController interface {
	Acquire(ctx context.Context, addr string) (Token, error)
}

// Token is returned by overload control and observes request completion.
type Token interface {
	OnResponse(ctx context.Context, err error)
}

// NoopOC is a no-op OverloadController.
type NoopOC struct{}

// Acquire always allows the request and returns a no-op token.
func (NoopOC) Acquire(context.Context, string) (Token, error) {
	return NoopToken{}, nil
}

// NoopToken is a no-op Token.
type NoopToken struct{}

// OnResponse does nothing.
func (NoopToken) OnResponse(context.Context, error) {}

// IsNoop checks whether the given overload controller is noop.
func IsNoop(oc OverloadController) bool {
	if oc == nil {
		return true
	}
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
