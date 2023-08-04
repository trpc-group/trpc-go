// Package consistenthash provides consistent hash utilities.
package consistenthash

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cespare/xxhash"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// defaultReplicas is the default virtual node coefficient.
const (
	defaultReplicas int = 100
	prime               = 16777619
)

// Hash is the hash function type.
type Hash func(data []byte) uint64

// defaultHashFunc uses CRC32 as the default.
var defaultHashFunc Hash = xxhash.Sum64

func init() {
	loadbalance.Register("consistent_hash", NewConsistentHash())
}

// NewConsistentHash creates a new ConsistentHash.
func NewConsistentHash() *ConsistentHash {
	return &ConsistentHash{
		pickers:  new(sync.Map),
		hashFunc: defaultHashFunc,
	}
}

// NewCustomConsistentHash creates a new ConsistentHash with custom hash function.
func NewCustomConsistentHash(hashFunc Hash) *ConsistentHash {
	return &ConsistentHash{
		pickers:  new(sync.Map),
		hashFunc: hashFunc,
	}
}

// ConsistentHash defines the consistent hash.
type ConsistentHash struct {
	pickers  *sync.Map
	interval time.Duration
	hashFunc Hash
}

// Select implements loadbalance.LoadBalancer.
func (ch *ConsistentHash) Select(serviceName string, list []*registry.Node,
	opt ...loadbalance.Option) (*registry.Node, error) {
	opts := &loadbalance.Options{}
	for _, o := range opt {
		o(opts)
	}
	p, ok := ch.pickers.Load(serviceName)
	if ok {
		return p.(*chPicker).Pick(list, opts)
	}

	newPicker := &chPicker{
		interval: ch.interval,
		hashFunc: ch.hashFunc,
	}
	v, ok := ch.pickers.LoadOrStore(serviceName, newPicker)
	if !ok {
		return newPicker.Pick(list, opts)
	}
	return v.(*chPicker).Pick(list, opts)
}

// chPicker is the picker of the consistent hash.
type chPicker struct {
	list     []*registry.Node
	hashFunc Hash
	keys     Uint64Slice                 // a hash slice of sorted node list, it's length is #(node)*replica
	hashMap  map[uint64][]*registry.Node // a map which keeps hash-nodes maps
	mu       sync.Mutex
	interval time.Duration
}

// Pick picks a node.
func (p *chPicker) Pick(list []*registry.Node, opts *loadbalance.Options) (*registry.Node, error) {
	if len(list) == 0 {
		return nil, loadbalance.ErrNoServerAvailable
	}
	// Returns error if opts.Key is not provided.
	if opts.Key == "" {
		return nil, errors.New("missing key")
	}
	tmpKeys, tmpMap, err := p.updateState(list, opts.Replicas)
	if err != nil {
		return nil, err
	}
	hash := p.hashFunc([]byte(opts.Key))
	// Find the best matched node by binary search. Node A is better than B if A's hash value is
	// greater than B's.
	idx := sort.Search(len(tmpKeys), func(i int) bool { return tmpKeys[i] >= hash })
	if idx == len(tmpKeys) {
		idx = 0
	}
	nodes, ok := tmpMap[tmpKeys[idx]]
	if !ok {
		return nil, loadbalance.ErrNoServerAvailable
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	default:
		innerIndex := p.hashFunc(innerRepr(opts.Key))
		pos := int(innerIndex % uint64(len(nodes)))
		return nodes[pos], nil
	}
}

// updateState recalculates list every so often if nodes changed.
func (p *chPicker) updateState(list []*registry.Node, replicas int) (Uint64Slice, map[uint64][]*registry.Node, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// if node list is the same as last update, there is no need to update hash ring.
	if isNodeSliceEqualBCE(p.list, list) {
		return p.keys, p.hashMap, nil
	}
	actualReplicas := replicas
	if actualReplicas <= 0 {
		actualReplicas = defaultReplicas
	}
	// update node list.
	p.list = list
	p.hashMap = make(map[uint64][]*registry.Node)
	p.keys = make(Uint64Slice, len(list)*actualReplicas)
	for i, node := range list {
		if node == nil {
			// node must not be nil.
			return nil, nil, errors.New("list contains nil node")
		}
		for j := 0; j < actualReplicas; j++ {
			hash := p.hashFunc([]byte(strconv.Itoa(j) + node.Address))
			p.keys[i*(actualReplicas)+j] = hash
			p.hashMap[hash] = append(p.hashMap[hash], node)
		}
	}
	sort.Sort(p.keys)
	return p.keys, p.hashMap, nil
}

// Uint64Slice defines uint64 slice.
type Uint64Slice []uint64

// Len returns the length of the slice.
func (s Uint64Slice) Len() int {
	return len(s)
}

// Less returns whether the value at i is less than j.
func (s Uint64Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

// Swap swaps values between i and j.
func (s Uint64Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// isNodeSliceEqualBCE check whether two node list is equal by BCE.
func isNodeSliceEqualBCE(a, b []*registry.Node) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	b = b[:len(a)]
	for i, v := range a {
		if (v == nil) != (b[i] == nil) {
			return false
		}
		if v.Address != b[i].Address {
			return false
		}
	}
	return true
}

func innerRepr(key interface{}) []byte {
	return []byte(fmt.Sprintf("%d:%v", prime, key))
}
