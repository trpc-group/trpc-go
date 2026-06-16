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
)

// Default is the default global actors.
var Default = NewActors()

// Actors keeps the order of requests by the given key for each jobs.
type Actors struct {
	mu     sync.RWMutex
	actors map[string]*Actor
}

// NewActors creates a new Actors.
func NewActors() *Actors {
	return &Actors{
		actors: make(map[string]*Actor),
	}
}

// Add adds a function with the given key to the Actors.
//
// If the speed of Add is higher than the capabilities of the actors, this function
// will block.
func (as *Actors) Add(key string, fn func()) {
	as.mu.RLock()
	a, ok := as.actors[key]
	as.mu.RUnlock()
	if !ok {
		as.mu.Lock()
		// Double check the existence.
		a, ok = as.actors[key]
		if !ok {
			a = NewActor(key,
				func() {
					as.mu.Lock()
					delete(as.actors, key)
					as.mu.Unlock()
				},
				nil,
			)
			as.actors[key] = a
		}
		as.mu.Unlock()
	}
	a.Add(fn)
}

// Remove safely removes an actor from the map.
func (as *Actors) Remove(key string) {
	as.mu.RLock()
	// Check if the actor is still in the map (it might have been removed already).
	a, ok := as.actors[key]
	as.mu.RUnlock()
	if ok {
		// Stop will remove the actor from the map.
		a.Stop()
	}
}

// Stop will stop all the current active actors.
func (as *Actors) Stop() {
	as.mu.RLock()
	for _, a := range as.actors {
		a.Stop()
	}
	as.mu.RUnlock()
}
