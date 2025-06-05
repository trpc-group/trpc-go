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

package bytes_test

import (
	"testing"

	ibytes "trpc.group/trpc-go/trpc-go/internal/bytes"
	"github.com/stretchr/testify/require"
)

func TestNopCloserBuffer(t *testing.T) {
	b := ibytes.GetNopCloserBuffer()
	require.Nil(t, b.Close())
	ibytes.PutNopCloserBuffer(b)
}
