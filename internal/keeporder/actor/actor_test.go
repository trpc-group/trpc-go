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

package actor

import (
	"sync"
	"testing"
	"time"
)

func TestActorJobProcessingWithOptions(t *testing.T) {
	var wg sync.WaitGroup
	key := "testActorWithOptions"
	jobHandled := false

	cleanup := func() {
		if !jobHandled {
			t.Error("Cleanup called before job was handled")
		}
		wg.Done()
	}

	opts := &Options{
		IdleGroupTimeout: 100 * time.Millisecond,
		MaxElementCount:  10,
	}

	a := NewActor(key, cleanup, opts)
	wg.Add(1)

	job := func() {
		jobHandled = true
	}

	a.Add(job)

	wg.Wait()

	if !jobHandled {
		t.Errorf("The job was not handled as expected")
	}
}

func TestActorIdleTimeoutWithOptions(t *testing.T) {
	var wg sync.WaitGroup
	key := "idleActorWithOptions"
	cleanupCalled := false

	cleanup := func() {
		cleanupCalled = true
		wg.Done()
	}

	opts := &Options{
		IdleGroupTimeout: 50 * time.Millisecond, // Short timeout for testing.
	}

	NewActor(key, cleanup, opts)
	wg.Add(1)

	wg.Wait()

	if !cleanupCalled {
		t.Errorf("Cleanup was not called after the idle timeout")
	}
}

func TestActorExplicitStopWithOptions(t *testing.T) {
	var wg sync.WaitGroup
	key := "stoppedActorWithOptions"
	cleanupCalled := false

	cleanup := func() {
		cleanupCalled = true
		wg.Done()
	}

	opts := &Options{
		IdleGroupTimeout: 1 * time.Second,
		MaxElementCount:  5,
	}

	a := NewActor(key, cleanup, opts)
	wg.Add(1)

	a.Stop()

	wg.Wait()

	if !cleanupCalled {
		t.Errorf("Cleanup was not called after actor was stopped")
	}
}
