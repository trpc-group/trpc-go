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

package keeporder_test

import (
	"context"
	"reflect"
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/keeporder"
)

func TestNewContextWithPreUnmarshal(t *testing.T) {
	ctx := context.Background()
	info := &keeporder.PreUnmarshalInfo{
		Stored:  true,
		ReqBody: "test request body",
	}

	// Create a new context with pre-unmarshal information.
	newCtx := keeporder.NewContextWithPreUnmarshal(ctx, info)

	// Retrieve the info back from the context.
	retrievedInfo, ok := keeporder.PreUnmarshalInfoFromContext(newCtx)
	if !ok {
		t.Fatalf("Expected pre-unmarshal info to be present in the context.")
	}

	// Check if the retrieved information is the same as what was added.
	if !reflect.DeepEqual(info, retrievedInfo) {
		t.Errorf("Expected retrieved info to be %+v, got %+v", info, retrievedInfo)
	}
}

func TestPreUnmarshalInfoFromContext_NoInfo(t *testing.T) {
	ctx := context.Background()

	// Attempt to retrieve pre-unmarshal info from a context that does not have it.
	_, ok := keeporder.PreUnmarshalInfoFromContext(ctx)
	if ok {
		t.Errorf("Expected no pre-unmarshal info to be present in the context.")
	}
}
