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

package graceful_test

import (
	"testing"

	. "trpc.group/trpc-go/trpc-go/internal/graceful/internal"
	"github.com/stretchr/testify/require"
)

func TestUnwrap(t *testing.T) {
	require.Equal(t, 1, Unwrap(1))
	var null wrapper
	var w wrapper = &wrap{wrapped: null}
	require.Equal(t, null, Unwrap(w))
}

type wrapper interface {
	wrap()
}

type wrap struct {
	wrapper
	wrapped wrapper
}

func (w *wrap) Unwrap() wrapper {
	return w.wrapped
}
