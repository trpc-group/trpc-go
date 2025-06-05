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

package errs_test

import (
	"errors"
	"testing"

	"trpc.group/trpc-go/trpc-go/errs"
	ierrs "trpc.group/trpc-go/trpc-go/transport/internal/errs"
)

func TestWrapReadFrameError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		errCode       int
		wantErrorCode int
	}{
		{
			"nil",
			nil,
			errs.RetClientConnectFail,
			errs.RetOK,
		},
		{
			"net timeout",
			&timeoutError{},
			errs.RetClientConnectFail,
			errs.RetClientTimeout,
		},
		{
			"other error",
			errors.New("something failed"),
			errs.RetClientNetErr,
			errs.RetClientNetErr,
		},
		{
			"other error",
			errors.New("something failed"),
			errs.RetClientReadFrameErr,
			errs.RetClientReadFrameErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errCode := errs.Code(ierrs.WrapAsClientTimeoutErrOr(tt.err, tt.errCode, "")); errCode != tt.wantErrorCode {
				t.Errorf("WrapAsClientTimeoutErrOr() error code = %v, wantErrorCode %v", errCode, tt.wantErrorCode)
			}
		})
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
