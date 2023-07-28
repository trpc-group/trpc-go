// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

import (
	"runtime"
	"sync"
	"sync/atomic"
)

var defaultShardSize = uint32(runtime.GOMAXPROCS(0))

// shardMap is a concurrent safe <id,*virtualConn> map.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type shardMap struct {
	size   uint32
	len    uint32
	shards []*shard
}

// shard is a concurrent safe map.
type shard struct {
	idToVirtualConn map[uint32]*virtualConn
	mu              sync.RWMutex
}

// newShardMap creates a new shardMap.
func newShardMap(size uint32) *shardMap {
	m := &shardMap{
		size:   size,
		shards: make([]*shard, size),
	}
	for i := range m.shards {
		m.shards[i] = &shard{
			idToVirtualConn: make(map[uint32]*virtualConn),
		}
	}
	return m
}

// getShard returns shard of given id.
func (m *shardMap) getShard(id uint32) *shard {
	return m.shards[id%m.size]
}

// loadOrStore returns the existing virtual connection for the id if present.
// Otherwise, it stores and returns the given vc. The loaded result is true if
// the vc was loaded, false if stored.
func (m *shardMap) loadOrStore(id uint32, vc *virtualConn) (actual *virtualConn, loaded bool) {
	shard := m.getShard(id)
	// Generally the ids are always different, here directly add the write lock.
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if actual, ok := shard.idToVirtualConn[id]; ok {
		return actual, true
	}
	atomic.AddUint32(&m.len, 1)
	shard.idToVirtualConn[id] = vc
	return vc, false
}

// store stores virtualConn.
func (m *shardMap) store(id uint32, vc *virtualConn) {
	shard := m.getShard(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.idToVirtualConn[id]; !ok {
		atomic.AddUint32(&m.len, 1)
	}
	shard.idToVirtualConn[id] = vc
}

// load loads the virtualConn of the given id.
func (m *shardMap) load(id uint32) (*virtualConn, bool) {
	shard := m.getShard(id)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	vc, ok := shard.idToVirtualConn[id]
	return vc, ok
}

// delete deletes the virtualConn of the given id.
func (m *shardMap) delete(id uint32) {
	shard := m.getShard(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.idToVirtualConn[id]; !ok {
		return
	}
	atomic.AddUint32(&m.len, ^uint32(0))
	delete(shard.idToVirtualConn, id)
}

// reset deletes all virtualConns in the shardMap.
func (m *shardMap) reset() {
	if m.length() == 0 {
		return
	}
	atomic.StoreUint32(&m.len, 0)
	for _, shard := range m.shards {
		shard.mu.Lock()
		shard.idToVirtualConn = make(map[uint32]*virtualConn)
		shard.mu.Unlock()
	}
}

// length returns number of all virtualConns in the shardMap.
func (m *shardMap) length() uint32 {
	return atomic.LoadUint32(&m.len)
}

// loadAll returns all virtualConns in the shardMap.
func (m *shardMap) loadAll() []*virtualConn {
	var conns []*virtualConn
	for _, shard := range m.shards {
		shard.mu.RLock()
		for _, v := range shard.idToVirtualConn {
			conns = append(conns, v)
		}
		shard.mu.RUnlock()
	}
	return conns
}
