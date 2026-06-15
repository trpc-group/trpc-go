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

package actor_test

import (
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/internal/keeporder/actor"
)

func TestActors(t *testing.T) {
	actors := actor.NewActors()
	key := "testKey"
	called := false

	// Define a function to be added.
	fn := func() {
		called = true
	}

	// Add the function to the actors under the specified key.
	actors.Add(key, fn)

	// Allow some time for the actor to process the function.
	time.Sleep(100 * time.Millisecond)

	// Check if the function was called.
	if !called {
		t.Errorf("The function was not called.")
	}

	// Remove the actor from the actors.
	actors.Remove(key)

	// Stop all the actors.
	// This should not cause panic when executed after remove.
	actors.Stop()
}
