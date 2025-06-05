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

// Package lru provides an implementation of LRU cache.
package lru

import (
	"sync"
	"time"
)

// NewLRU returns a new LRU. A value is scavenged if it is not actived for ttl.
func NewLRU[T any](ttl time.Duration, newVal func() T) *LRU[T] {
	pool := getPool(func() *node[T] { return &node[T]{} })

	sentinel := pool.Get()
	sentinel.prev = sentinel
	sentinel.next = sentinel
	return &LRU[T]{
		pool:     pool,
		ttl:      ttl,
		newVal:   newVal,
		nodes:    make(map[string]*node[T]),
		sentinel: sentinel,
	}
}

// LRU is a least recently used cache.
type LRU[T any] struct {
	pool    *pool[*node[T]]
	zeroVal T

	ttl    time.Duration
	newVal func() T

	mu       sync.Mutex
	nodes    map[string]*node[T]
	sentinel *node[T]
}

type node[T any] struct {
	activeAt time.Time
	prev     *node[T]
	next     *node[T]
	key      string
	val      T
}

var pools = sync.Map{}

func getPool[T any](newT func() T) *pool[T] {
	var zeroT T
	val, _ := pools.LoadOrStore(zeroT, &sync.Pool{New: func() interface{} {
		return newT()
	}})
	return (*pool[T])(val.(*sync.Pool))
}

type pool[T any] sync.Pool

func (p *pool[T]) Get() T {
	return (*sync.Pool)(p).Get().(T)
}

func (p *pool[T]) Put(t T) {
	(*sync.Pool)(p).Put(t)
}

// Get always returns an value. If key does not exist, Get creates one.
// Expired values is scavenged when Get is called.
func (l *LRU[T]) Get(key string) T {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if node, ok := l.nodes[key]; ok {
		node.activeAt = now
		remove(node)
		addAfter(l.sentinel, node)
		l.scavenge(now)
		return node.val
	}

	node := l.pool.Get()
	node.activeAt = now
	node.key = key
	addAfter(l.sentinel, node)
	node.val = l.newVal()
	l.nodes[key] = node

	l.scavenge(now)
	return node.val
}

func (l *LRU[T]) scavenge(now time.Time) {
	for curr := l.sentinel.prev; curr != l.sentinel && now.Sub(curr.activeAt) > l.ttl; {
		delete(l.nodes, curr.key)
		remove(curr)
		curr.val = l.zeroVal
		curr.next = nil
		nextCurr := curr.prev
		curr.prev = nil
		l.pool.Put(curr)
		curr = nextCurr
	}
}

func remove[T any](n *node[T]) {
	n.prev.next = n.next
	n.next.prev = n.prev
}

func addAfter[T any](at, n *node[T]) {
	n.next = at.next
	n.prev = at
	at.next.prev = n
	at.next = n
}
