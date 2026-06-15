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

package keeporder_test

import (
	"context"
	"reflect"
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/keeporder"
)

func TestNewContextWithPreDecode(t *testing.T) {
	ctx := context.Background()
	info := &keeporder.PreDecodeInfo{
		ReqBodyBuf: []byte("test data"),
	}

	// Create a new context with pre-decoded information.
	newCtx := keeporder.NewContextWithPreDecode(ctx, info)

	// Retrieve the info back from the context.
	retrievedInfo, ok := keeporder.PreDecodeInfoFromContext(newCtx)
	if !ok {
		t.Fatalf("Expected pre-decoded info to be present in the context.")
	}

	// Check if the retrieved information is the same as what was added.
	if !reflect.DeepEqual(info, retrievedInfo) {
		t.Errorf("Expected retrieved info to be %+v, got %+v", info, retrievedInfo)
	}
}

func TestPreDecodeInfoFromContext_NoInfo(t *testing.T) {
	ctx := context.Background()

	// Attempt to retrieve pre-decoded info from a context that does not have it.
	_, ok := keeporder.PreDecodeInfoFromContext(ctx)
	if ok {
		t.Errorf("Expected no pre-decoded info to be present in the context.")
	}
}
