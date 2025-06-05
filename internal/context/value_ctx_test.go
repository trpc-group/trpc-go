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

package context_test

import (
	"context"
	"testing"

	icontext "trpc.group/trpc-go/trpc-go/internal/context"
	"github.com/stretchr/testify/require"
)

func TestWithValues(t *testing.T) {
	type testKey struct{}
	testValue := "value"
	ctx := context.WithValue(context.TODO(), testKey{}, testValue)
	ctx1 := icontext.NewContextWithValues(context.TODO(), ctx)
	require.NotNil(t, ctx1.Value(testKey{}))
	type notExist struct{}
	require.Nil(t, ctx1.Value(notExist{}))
}
