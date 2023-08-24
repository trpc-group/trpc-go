// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package consistenthash

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/spaolacci/murmur3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// Test whether key takes effect.
// The returned node should not change for the same key in the same node list.
func TestConsistentHashGetOne(t *testing.T) {
	ch := NewConsistentHash()

	// test list 1
	n, err := ch.Select("test", list1, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	expectAddr := n.Address
	n, err = ch.Select("test", list1, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list1, loadbalance.WithKey("123456"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list1, loadbalance.WithKey("123456"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list1, loadbalance.WithKey("12315"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list1, loadbalance.WithKey("12315"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	// test list 4
	n, err = ch.Select("test", list4, loadbalance.WithKey("Pony"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list4, loadbalance.WithKey("Pony"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list4, loadbalance.WithKey("John"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list4, loadbalance.WithKey("John"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list4, loadbalance.WithKey("Jack"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list4, loadbalance.WithKey("Jack"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)
}

// Test whether key takes effect using custom.
// The returned node should not change for the same key in the same node list.
func TestCustomConsistentHashGetOne(t *testing.T) {
	ch := NewCustomConsistentHash(murmur3.Sum64)

	// test list 1
	n, err := ch.Select("test", list1, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	expectAddr := n.Address
	n, err = ch.Select("test", list1, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list1, loadbalance.WithKey("123456"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list1, loadbalance.WithKey("123456"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list1, loadbalance.WithKey("12315"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list1, loadbalance.WithKey("12315"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	// test list 4
	n, err = ch.Select("test", list4, loadbalance.WithKey("Pony"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list4, loadbalance.WithKey("Pony"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list4, loadbalance.WithKey("John"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list4, loadbalance.WithKey("John"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)

	n, err = ch.Select("test", list4, loadbalance.WithKey("Jack"))
	assert.Nil(t, err)
	expectAddr = n.Address
	n, err = ch.Select("test", list4, loadbalance.WithKey("Jack"))
	assert.Nil(t, err)
	assert.Equal(t, expectAddr, n.Address)
}

// Test hash-collision.
// The returned node should not equal for the different key.
func TestHashCollision(t *testing.T) {
	const magicKey = "magic_key"
	ch := NewCustomConsistentHash(func(data []byte) uint64 {
		if id := strings.Index(string(data), magicKey); id != -1 {
			// the hash value is determined by the byte after magic key
			return uint64(data[id+len(magicKey)])
		}
		// hash must collide if missing magic key.
		return 0
	})

	nodes := []*registry.Node{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
	}

	// consistent hash has two replicas, select three different keys(three different hashes), there's no possible that
	// all of them shares the same address.
	addresses := make(map[string]struct{})

	n, err := ch.Select("test", nodes, loadbalance.WithKey(magicKey+"a"), loadbalance.WithReplicas(2))
	require.Nil(t, err)
	log.Debug(n.Address)
	addresses[n.Address] = struct{}{}
	n, err = ch.Select("test", nodes, loadbalance.WithKey(magicKey+"b"), loadbalance.WithReplicas(2))
	require.Nil(t, err)
	log.Debug(n.Address)
	addresses[n.Address] = struct{}{}
	n, err = ch.Select("test", nodes, loadbalance.WithKey(magicKey+"c"), loadbalance.WithReplicas(2))
	require.Nil(t, err)
	log.Debug(n.Address)
	addresses[n.Address] = struct{}{}
	require.Less(t, 1, len(addresses))
}

// Test empty node list.
// Should return an expected error.
func TestNilList(t *testing.T) {
	ch := NewConsistentHash()
	n, err := ch.Select("test", nil, loadbalance.WithKey("123"))
	assert.Nil(t, n)
	assert.Equal(t, loadbalance.ErrNoServerAvailable, err)
}

// Test empty opt.
// WithKey of opt must be provided.
// Should return an expected error.
func TestNilOpts(t *testing.T) {
	ch := NewConsistentHash()

	n, err := ch.Select("test", list1)
	assert.Nil(t, n)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "missing key")

	n, err = ch.Select("test", list1, loadbalance.WithKey("whatever"))
	assert.Nil(t, err)
	assert.NotNil(t, n)
}

// Test node list with only one node.
// Should return the same result each time.
func TestSingleNode(t *testing.T) {
	ch := NewConsistentHash()
	n, err := ch.Select("test", list2, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	assert.Equal(t, list2[0].Address, n.Address)

	n, err = ch.Select("test", list2, loadbalance.WithKey("456"))
	assert.Nil(t, err)
	assert.Equal(t, list2[0].Address, n.Address)

	n, err = ch.Select("test", list2, loadbalance.WithKey("12306"))
	assert.Nil(t, err)
	assert.Equal(t, list2[0].Address, n.Address)

	n, err = ch.Select("test", list2, loadbalance.WithKey("JackChen"))
	assert.Nil(t, err)
	assert.Equal(t, list2[0].Address, n.Address)
}

// Hash ring should be updated once length of node list changed.
func TestInterval(t *testing.T) {
	ch := NewConsistentHash()

	// On list length changed, recalculate hash ring immediately.
	n, err := ch.Select("test", list2, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	assert.Equal(t, list2[0].Address, n.Address)

	n, err = ch.Select("test", list4, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	assert.Equal(t, false, isInList(n.Address, list2))
	assert.Equal(t, true, isInList(n.Address, list4))
}

// Test the influence to object mapping position if node is deleted.
func TestSubNode(t *testing.T) {
	ch := NewConsistentHash()

	var address1, address2, address3 string
	n, err := ch.Select("test", list1, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	address1 = n.Address

	n, err = ch.Select("test", list1, loadbalance.WithKey("123456"))
	assert.Nil(t, err)
	address2 = n.Address

	n, err = ch.Select("test", list1, loadbalance.WithKey("12315"))
	assert.Nil(t, err)
	address3 = n.Address

	deletedAddress := address1

	// Delete deletedAddress of list1.
	// No key is effected except the key influenced by deletedAddress.
	listTmp := deleteNode(deletedAddress, list1)

	n, err = ch.Select("test", listTmp, loadbalance.WithKey("123"))
	assert.Nil(t, err)
	if address1 != deletedAddress {
		assert.Equal(t, address1, n.Address)
	} else {
		assert.NotEqual(t, address1, n.Address)
	}

	n, err = ch.Select("test", listTmp, loadbalance.WithKey("123456"))
	assert.Nil(t, err)
	if address2 != deletedAddress {
		assert.Equal(t, address2, n.Address)
	} else {
		assert.NotEqual(t, address2, n.Address)
	}

	n, err = ch.Select("test", listTmp, loadbalance.WithKey("12315"))
	assert.Nil(t, err)
	if address3 != deletedAddress {
		assert.Equal(t, address3, n.Address)
	} else {
		assert.NotEqual(t, address3, n.Address)
	}
}

// Test balance.
func TestBalance(t *testing.T) {
	ch := NewConsistentHash()
	counter := make(map[string]int)
	for i := 0; i < 200; i++ {
		n, err := ch.Select("test", list1, loadbalance.WithKey(fmt.Sprintf("%d", i)), loadbalance.WithReplicas(100))
		assert.Nil(t, err)
		if _, ok := counter[n.Address]; !ok {
			counter[n.Address] = 0
		} else {
			counter[n.Address]++
		}
	}
	for _, v := range counter {
		assert.NotEqual(t, 0, v)
		fmt.Println(v)
	}
}

func TestIsNodeSliceEqualBCE(t *testing.T) {
	isEqual := isNodeSliceEqualBCE(list1, list2)
	assert.Equal(t, false, isEqual)
	isEqual = isNodeSliceEqualBCE(list1, list1)
	assert.Equal(t, true, isEqual)
	isEqual = isNodeSliceEqualBCE(list1, nil)
	assert.Equal(t, false, isEqual)
}

// Test concurrency safety. List changes every visit.
// This is an extreme situation. In fact, in most cases, node list of a service does not change
// frequently, but will only change on service scaling.
func TestParallel(t *testing.T) {
	var wg sync.WaitGroup
	ch := NewConsistentHash()
	var lists [][]*registry.Node
	var keys []string
	var results []string

	n, err := ch.Select("test", list1, loadbalance.WithKey("1"))
	assert.Nil(t, err)
	results = append(results, n.Address)
	lists = append(lists, list1)
	keys = append(keys, "1")

	n, err = ch.Select("test", list2, loadbalance.WithKey("2"))
	assert.Nil(t, err)
	results = append(results, n.Address)
	lists = append(lists, list2)
	keys = append(keys, "2")

	n, err = ch.Select("test", list3, loadbalance.WithKey("3"))
	assert.Nil(t, err)
	results = append(results, n.Address)
	lists = append(lists, list3)
	keys = append(keys, "3")

	n, err = ch.Select("test", list4, loadbalance.WithKey("4"))
	assert.Nil(t, err)
	results = append(results, n.Address)
	lists = append(lists, list4)
	keys = append(keys, "4")

	n, err = ch.Select("test", list5, loadbalance.WithKey("5"))
	assert.Nil(t, err)
	results = append(results, n.Address)
	lists = append(lists, list5)
	keys = append(keys, "5")

	// To simulate a large concurrent goroutine.

	// This is an extreme situation. In fact, in most cases, node list of a service does not change
	// frequently, but will only change on service scaling.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n, err := ch.Select("test0", lists[i%5], loadbalance.WithKey(keys[i%5]))
			assert.Nil(t, err)
			assert.Equal(t, results[i%5], n.Address)
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n, err := ch.Select("test1", lists[0], loadbalance.WithKey(keys[0]))
			assert.Nil(t, err)
			assert.Equal(t, results[0], n.Address)
		}(i)
	}
	wg.Wait()
}

// Test performance on current visit.
func BenchmarkParallel(b *testing.B) {
	ch := NewConsistentHash()
	b.SetParallelism(10) // 10 concurrency
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = ch.Select("test", list1, loadbalance.WithKey("HelloWorld"))
		}
	})
}

var list1 = []*registry.Node{
	{
		Address: "list1.ip.1:8080",
	},
	{
		Address: "list1.ip.2:8080",
	},
	{
		Address: "list1.ip.3:8080",
	},
	{
		Address: "list1.ip.4:8080",
	},
}

var list2 = []*registry.Node{
	{
		Address: "list2.ip.1:8080",
	},
}

var list3 = []*registry.Node{
	{
		Address: "list3.ip.2:8080",
	},
	{
		Address: "list3.ip.4:8080",
	},
	{
		Address: "list3.ip.1:8080",
	},
}

var list4 = []*registry.Node{
	{
		Address: "list4.ip.168:8080",
	},
	{
		Address: "list4.ip.167:8080",
	},
	{
		Address: "list4.ip.15:8080",
	},
	{
		Address: "list4.ip.15:8081",
	},
}

var list5 = []*registry.Node{
	{
		Address: "list5.ip.2:8080",
	},
}

func deleteNode(address string, list []*registry.Node) []*registry.Node {
	ret := make([]*registry.Node, 0, len(list))
	for _, n := range list {
		if n.Address != address {
			ret = append(ret, n)
		}
	}
	return ret
}

func isInList(address string, list []*registry.Node) bool {
	for _, n := range list {
		if n.Address == address {
			return true
		}
	}
	return false
}
