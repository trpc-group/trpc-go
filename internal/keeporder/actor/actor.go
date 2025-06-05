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

// Package actor provides the implementation for actor model.
package actor

import (
	"sync"
	"time"
)

// The default values for actor.
const (
	defaultIdleGroupTimeout = 30 * time.Second
	defaultMaxElementCount  = 1024
)

// Actor is a single actor that handle the jobs in order.
type Actor struct {
	key     string
	jobs    chan func()
	cleanup func()
	once    sync.Once

	idleGroupTimeout time.Duration
}

// NewActor creates a new Actor.
//
// It handles the jobs in order, if no job is received after a certain timeout,
// the actor will quit.
func NewActor(
	key string,
	cleanup func(),
	opts *Options,
) *Actor {
	if opts == nil {
		opts = &Options{}
	}
	opts.fixDefault()
	a := &Actor{
		key:              key,
		jobs:             make(chan func(), opts.MaxElementCount),
		cleanup:          cleanup,
		idleGroupTimeout: opts.IdleGroupTimeout,
	}
	a.start()
	return a
}

func (a *Actor) start() {
	go func() {
		timer := time.NewTimer(a.idleGroupTimeout)
		defer timer.Stop()
		defer a.cleanup()
		for {
			// Quoted from the comments of timer.Reset:
			//  Before Go 1.23, the only safe way to use Reset was to [Stop] and explicitly drain the timer first.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(a.idleGroupTimeout)
			select {
			case fn, ok := <-a.jobs:
				if !ok {
					// The job queue is closed, return to clean up it.
					return
				}
				fn()
			case <-timer.C:
				// Still no request for this key after idle timeout,
				// return to cleanup it.
				return
			}
		}
	}()
}

// Add adds a function to the job queue.
func (a *Actor) Add(fn func()) {
	a.jobs <- fn
}

// Stop stops the actor.
func (a *Actor) Stop() {
	a.once.Do(func() {
		close(a.jobs)
	})
}
