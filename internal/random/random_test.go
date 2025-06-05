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

package random

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func assertRate(t *testing.T, want, total, got float64) {
	t.Helper()
	r := got / total
	diff := math.Abs(r - want)
	if diff > 0.01 {
		t.Fatalf("expect rate %.4f, got %.4f, diff %.4f", want, r, diff)
	}
}

func TestTrue(t *testing.T) {
	var (
		mu         sync.Mutex
		trueCount  float64
		falseCount float64
		wg         sync.WaitGroup
	)

	for i := 0; i < 65536; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := True(0.5)
			mu.Lock()
			defer mu.Unlock()
			if b {
				trueCount++
			} else {
				falseCount++
			}
		}()
	}
	wg.Wait()
	assertRate(t, 0.5, trueCount+falseCount, trueCount)

	if True(0) {
		t.Fatal("should NOT be true")
	}
	if !True(1) {
		t.Fatal("should always be true")
	}
}

func TestIntn(t *testing.T) {
	if i := Intn(10); i > 10 {
		t.Fatal(i)
	}
}

func TestFloat64(t *testing.T) {
	if f := Float64(); f > 1.0 || f < 0.0 {
		t.Fatal(f)
	}
}

func TestPanic(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			require.NotPanics(t, func() {
				defer wg.Done()
				testPanic(r)
			})
		}()
	}
	wg.Wait()
}

func testPanic(r *rand.Rand) {
	_ = r.Int()
	_ = r.Intn(32)
	_ = r.Int31()
	_ = r.Int31n(32)
	_ = r.Int63()
	_ = r.Int63n(32)
	_ = r.ExpFloat64()
	_ = r.NormFloat64()
	_ = r.Float32()
	_ = r.Float64()
	_, _ = r.Read(make([]byte, 10))
	_ = r.Perm(10)
	r.Shuffle(10, func(i, j int) {})
	_ = r.Uint32()
	_ = r.Uint64()
	r.Seed(10)
}

// benchSink prevents the compiler from optimizing away benchmark loops.
var benchSink int32

func BenchmarkStandard(b *testing.B) {
	b.RunParallel(func(p *testing.PB) {
		var s int
		for p.Next() {
			s += rand.Intn(10)
		}
		atomic.AddInt32(&benchSink, int32(s))
	})
}

func BenchmarkRandom(b *testing.B) {
	b.RunParallel(func(p *testing.PB) {
		var s int
		for p.Next() {
			s += Intn(10)
		}
		atomic.AddInt32(&benchSink, int32(s))
	})
}
