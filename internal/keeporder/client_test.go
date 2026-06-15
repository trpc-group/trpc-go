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
	"errors"
	"testing"

	"trpc.group/trpc-go/trpc-go/internal/keeporder"
)

func TestNewContextWithClientInfo(t *testing.T) {
	// Create a ClientInfo and embed it into a context.
	clientInfo := &keeporder.ClientInfo{
		SendError: make(chan error, 1),
	}
	ctx := keeporder.NewContextWithClientInfo(context.Background(), clientInfo)

	// Retrieve the ClientInfo from the context.
	retrievedInfo, ok := keeporder.ClientInfoFromContext(ctx)
	if !ok {
		t.Errorf("ClientInfo was not found in the context")
	}

	// Check that the retrieved ClientInfo is the same as the original.
	if retrievedInfo != clientInfo {
		t.Errorf("Retrieved ClientInfo is not the same as the original")
	}

	// Check that the SendError channel works.
	testErr := errors.New("test error")
	clientInfo.SendError <- testErr
	receivedErr := <-retrievedInfo.SendError
	if receivedErr != testErr {
		t.Errorf("Error sent through SendError channel was not received correctly")
	}
}

func TestClientInfoFromContext_NoClientInfo(t *testing.T) {
	// Create a context without ClientInfo.
	ctx := context.Background()

	// Try to retrieve ClientInfo from the context.
	_, ok := keeporder.ClientInfoFromContext(ctx)
	if ok {
		t.Errorf("ClientInfo should not be found in the context")
	}
}
