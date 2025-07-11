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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShardMap(t *testing.T) {
	var (
		id uint32 = 1
		vc        = &virtualConnection{}
		m         = newShardMap(4)
	)
	_, loaded := m.loadOrStore(id, vc)
	require.False(t, loaded)
	_, loaded = m.loadOrStore(id, vc)
	require.True(t, loaded)
	require.Equal(t, uint32(1), m.length())
	require.Equal(t, 1, len(m.loadAll()))
	_, ok := m.load(id)
	require.True(t, ok)

	m.delete(id)
	_, ok = m.load(id)
	require.False(t, ok)
}

func BenchmarkMutexMap(b *testing.B) {
	var mu sync.RWMutex
	m := make(map[uint32]*virtualConnection)
	var (
		id uint32
		vc virtualConnection
	)
	b.SetParallelism(128)
	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			key := atomic.AddUint32(&id, 1)
			mu.Lock()
			m[key] = &vc
			mu.Unlock()

			mu.RLock()
			_ = m[key]
			mu.RUnlock()

			mu.Lock()
			delete(m, key)
			mu.Unlock()

			mu.RLock()
			_ = len(m)
			mu.RUnlock()
		}
	})
}

func BenchmarkShardMap(b *testing.B) {
	m := newShardMap(32)
	var (
		id uint32
		vc virtualConnection
	)
	b.SetParallelism(128)
	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			key := atomic.AddUint32(&id, 1)
			m.store(key, &vc)
			m.load(key)
			m.delete(key)
			m.length()
		}
	})
}

func BenchmarkSyncMap(b *testing.B) {
	var m sync.Map
	var (
		id uint32
		vc virtualConnection
	)
	b.SetParallelism(128)
	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			key := atomic.AddUint32(&id, 1)
			m.Store(key, &vc)
			m.Load(key)
			m.Delete(key)
		}
	})
}
