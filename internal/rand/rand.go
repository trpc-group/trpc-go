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

// Package rand provides public goroutine-safe random function.
// The implementation is similar to grpc random functions. Additionally,
// the seed function is provided to be called from the outside, and
// the random functions are provided as a struct's methods.
package rand

import (
	"math/rand"
	"sync"
)

// SafeRand is the safe random functions struct.
type SafeRand struct {
	r  *rand.Rand
	mu sync.Mutex
}

// NewSafeRand creates a SafeRand using the given seed.
func NewSafeRand(seed int64) *SafeRand {
	c := &SafeRand{
		r: rand.New(rand.NewSource(seed)),
	}
	return c
}

// Int63n provides a random int64.
func (c *SafeRand) Int63n(n int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := c.r.Int63n(n)
	return res
}

// Intn provides a random int.
func (c *SafeRand) Intn(n int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := c.r.Intn(n)
	return res
}

// Float64 provides a random float64.
func (c *SafeRand) Float64() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := c.r.Float64()
	return res
}
