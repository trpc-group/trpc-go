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

// Package errs provides common function for error handling.
package errs

import (
	"errors"
	"net"

	"trpc.group/trpc-go/trpc-go/errs"
)

// WrapAsClientTimeoutErrOr wraps err as ClientTimeout error or returns a new error with errCode and msg.
// If err is nil, return the original nil err.
func WrapAsClientTimeoutErrOr(err error, errCode int, msg string) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return errs.WrapFrameError(err, errs.RetClientTimeout, msg)
	}
	return errs.WrapFrameError(err, errCode, msg)
}

// ErrListenerNotFound indicates that the requested listener was not found in the transport layer.
var ErrListenerNotFound = errors.New("listener not found")
