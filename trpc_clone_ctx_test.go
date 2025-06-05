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

package trpc_test

import (
	"context"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go"
)

func TestContextCloning(t *testing.T) {
	// Define a timeout duration for the test.
	timeoutDuration := 50 * time.Millisecond

	t.Run("CloneContextWithoutTimeout", func(t *testing.T) {
		originalCtx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Sleep to simulate work, but less than the timeout duration.
		time.Sleep(10 * time.Millisecond)

		// Clone the context without retaining the timeout.
		clonedCtx := trpc.CloneContext(originalCtx)

		// Wait for the original context to reach its deadline.
		<-originalCtx.Done()
		if err := originalCtx.Err(); err != context.DeadlineExceeded {
			t.Errorf("Expected original context to be canceled due to deadline exceeded, got %v", err)
		}

		// Check that the cloned context is not canceled.
		select {
		case <-clonedCtx.Done():
			t.Errorf("Cloned context should not be canceled")
		case <-time.After(timeoutDuration):
			// Expected result, cloned context is not canceled.
		}
	})

	t.Run("CloneContextWithTimeout", func(t *testing.T) {
		originalCtx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Sleep to simulate work, but less than the timeout duration.
		time.Sleep(10 * time.Millisecond)

		// Clone the context while retaining the timeout.
		clonedCtx := trpc.CloneContextWithTimeout(originalCtx)

		// Wait for the original context to reach its deadline.
		<-originalCtx.Done()
		if err := originalCtx.Err(); err != context.DeadlineExceeded {
			t.Errorf("Expected original context to be canceled due to deadline exceeded, got %v", err)
		}

		// Check that the cloned context is also canceled due to the deadline.
		select {
		case <-clonedCtx.Done():
			if err := clonedCtx.Err(); err != context.DeadlineExceeded {
				t.Errorf("Expected cloned context to be canceled due to deadline exceeded, got %v", err)
			}
		case <-time.After(timeoutDuration):
			t.Errorf("Expected cloned context to be canceled due to deadline exceeded, but it was not")
		}
	})
}
