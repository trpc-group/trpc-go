//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package multiplex

import (
	"runtime"
	"sync"
	"sync/atomic"
)

var defaultShardSize = uint32(runtime.GOMAXPROCS(0))

// shardMap is a concurrent safe <id,*virConn> map.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type shardMap struct {
	size   uint32
	len    uint32
	shards []*shard
}

// shard is a concurrent safe map.
type shard struct {
	idToVirConn map[uint32]*virtualConnection
	mu          sync.RWMutex
}

// newShardMap creates a new shardMap.
func newShardMap(size uint32) *shardMap {
	m := &shardMap{
		size:   size,
		shards: make([]*shard, size),
	}
	for i := range m.shards {
		m.shards[i] = &shard{
			idToVirConn: make(map[uint32]*virtualConnection),
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
func (m *shardMap) loadOrStore(id uint32, vc *virtualConnection) (actual *virtualConnection, loaded bool) {
	shard := m.getShard(id)
	// Generally the ids are always different, here directly add the write lock.
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if actual, ok := shard.idToVirConn[id]; ok {
		return actual, true
	}
	atomic.AddUint32(&m.len, 1)
	shard.idToVirConn[id] = vc
	return vc, false
}

// store stores virConn.
func (m *shardMap) store(id uint32, vc *virtualConnection) {
	shard := m.getShard(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.idToVirConn[id]; !ok {
		atomic.AddUint32(&m.len, 1)
	}
	shard.idToVirConn[id] = vc
}

// load loads the virConn of the given id.
func (m *shardMap) load(id uint32) (*virtualConnection, bool) {
	shard := m.getShard(id)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	vc, ok := shard.idToVirConn[id]
	return vc, ok
}

// delete deletes the virConn of the given id.
func (m *shardMap) delete(id uint32) {
	shard := m.getShard(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok := shard.idToVirConn[id]; !ok {
		return
	}
	atomic.AddUint32(&m.len, ^uint32(0))
	delete(shard.idToVirConn, id)
}

// reset deletes all virConns in the shardMap.
func (m *shardMap) reset() {
	if m.length() == 0 {
		return
	}
	atomic.StoreUint32(&m.len, 0)
	for _, shard := range m.shards {
		shard.mu.Lock()
		shard.idToVirConn = make(map[uint32]*virtualConnection)
		shard.mu.Unlock()
	}
}

// length returns number of all virConns in the shardMap.
func (m *shardMap) length() uint32 {
	return atomic.LoadUint32(&m.len)
}

// loadAll returns all virConns in the shardMap.
func (m *shardMap) loadAll() []*virtualConnection {
	var conns []*virtualConnection
	for _, shard := range m.shards {
		shard.mu.RLock()
		for _, v := range shard.idToVirConn {
			conns = append(conns, v)
		}
		shard.mu.RUnlock()
	}
	return conns
}
