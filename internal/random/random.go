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

// Package random provides goroutine-safe high performance random number generator.
package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"
)

func newSeed() (seed int64) {
	if crand.Reader != nil {
		if err := binary.Read(crand.Reader, binary.BigEndian, &seed); err == nil {
			return seed
		}
	}
	return time.Now().UnixNano()
}

func newSource() interface{} {
	return rand.NewSource(newSeed()) // nolint:gosec
}

type poolSource struct {
	sync.Pool
}

func (p *poolSource) Int63() int64 {
	v := p.Pool.Get()
	defer p.Pool.Put(v)
	return v.(rand.Source).Int63()
}

// Seed is a no-op.
// It is provided for compatibility with the rand.Source interface.
// Source is pooled, so it is not safe to seed it.
func (p *poolSource) Seed(seed int64) {
}

func (p *poolSource) Uint64() uint64 {
	v := p.Pool.Get()
	defer p.Pool.Put(v)
	return v.(rand.Source64).Uint64()
}

func newPoolSource() *poolSource {
	var p = &poolSource{}
	p.New = newSource
	return p
}

// New returns a new goroutine-safe pseudo-random source.
func New() *rand.Rand {
	return rand.New(newPoolSource())
}

var defaultRand = New()

// Int returns a non-negative pseudo-random int.
func Int() int {
	return defaultRand.Int()
}

// Intn returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n).
// It panics if n <= 0.
func Intn(n int) int {
	return defaultRand.Intn(n)
}

// Int31 returns a non-negative pseudo-random 31-bit integer as an int32.
func Int31() int32 {
	return defaultRand.Int31()
}

// Int31n returns, as an int32, a non-negative pseudo-random number in the half-open interval [0,n).
// It panics if n <= 0.
func Int31n(n int32) int32 {
	return defaultRand.Int31n(n)
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func Int63() int64 {
	return defaultRand.Int63()
}

// Int63n returns, as an int64, a non-negative pseudo-random number in the half-open interval [0,n).
// It panics if n <= 0.
func Int63n(n int64) int64 {
	return defaultRand.Int63n(n)
}

// Uint32 returns a pseudo-random 32-bit value as a uint32.
func Uint32() uint32 {
	return defaultRand.Uint32()
}

// Uint64 returns a pseudo-random 64-bit value as a uint64.
func Uint64() uint64 {
	return defaultRand.Uint64()
}

// Float64 returns, as a float64, a pseudo-random number in the half-open interval [0.0,1.0).
func Float64() float64 {
	return defaultRand.Float64()
}

// Float32 returns, as a float32, a pseudo-random number in the half-open interval [0.0,1.0).
func Float32() float32 {
	return defaultRand.Float32()
}

// ExpFloat64 returns an exponentially distributed float64 in the range
// (0, +math.MaxFloat64] with an exponential distribution whose rate parameter
// (lambda) is 1 and whose mean is 1/lambda (1).
// To produce a distribution with a different rate parameter,
// callers can adjust the output using:
//
//	sample = ExpFloat64() / desiredRateParameter
func ExpFloat64() float64 {
	return defaultRand.ExpFloat64()
}

// NormFloat64 returns a normally distributed float64 in
// the range -math.MaxFloat64 through +math.MaxFloat64 inclusive,
// with standard normal distribution (mean = 0, stddev = 1).
// To produce a different normal distribution, callers can
// adjust the output using:
//
//	sample = NormFloat64() * desiredStdDev + desiredMean
func NormFloat64() float64 {
	return defaultRand.NormFloat64()
}

// Perm returns, as a slice of n ints, a pseudo-random permutation of the integers
// in the half-open interval [0,n).
func Perm(n int) []int {
	return defaultRand.Perm(n)
}

// Read generates len(p) random bytes and writes them into p. It
// always returns len(p) and a nil error.
// Read should not be called concurrently with any other Rand method.
func Read(p []byte) (n int, err error) {
	return defaultRand.Read(p)
}

// Shuffle pseudo-randomizes the order of elements.
// n is the number of elements. Shuffle panics if n < 0.
// swap swaps the elements with indexes i and j.
func Shuffle(n int, swap func(i, j int)) {
	defaultRand.Shuffle(n, swap)
}

// True returns true with probability of input `prob`(interval [0.0, 1.0]).
// It returns false if prob lower or equal than 0.
// It returns true if prob larger or equal than 1.
func True(prob float64) (ok bool) {
	if prob <= 0 {
		return false
	}
	if prob >= 1 {
		return true
	}
	ok = defaultRand.Float64() < prob
	return
}
