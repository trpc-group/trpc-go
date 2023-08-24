// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package bannednodes defines a concurrent safe node list which may be bound to a context.
package bannednodes

import (
	"context"
	"sync"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// ctxKeyBannedNodes is the key of the context.
type ctxKeyBannedNodes struct{}

// bannedNodes is the value of the context.
type bannedNodes struct {
	mu        sync.Mutex
	nodes     *Nodes
	mandatory bool
}

// NewCtx creates a new context and sets its k-v.
func NewCtx(ctx context.Context, mandatory bool) context.Context {
	return context.WithValue(ctx, ctxKeyBannedNodes{}, &bannedNodes{mandatory: mandatory})
}

// FromCtx returns the node list and a boolean which indicate whether it is abandoned.
// FromCtx does not return a bannedNodes, but a read only linked list. The internal lock is not
// exported to external user.
func FromCtx(ctx context.Context) (nodes *Nodes, mandatory bool, ok bool) {
	bannedNodes, ok := ctx.Value(ctxKeyBannedNodes{}).(*bannedNodes)
	if !ok {
		return nil, false, false
	}

	bannedNodes.mu.Lock()
	defer bannedNodes.mu.Unlock()
	return bannedNodes.nodes, bannedNodes.mandatory, true
}

// Add adds a new node to ctx.
// Nothing would happen if there's no k-v.
func Add(ctx context.Context, nodes ...*registry.Node) {
	bannedNodes, ok := ctx.Value(ctxKeyBannedNodes{}).(*bannedNodes)
	if !ok {
		return
	}

	bannedNodes.mu.Lock()
	defer bannedNodes.mu.Unlock()
	for _, node := range nodes {
		bannedNodes.nodes = &Nodes{
			next: bannedNodes.nodes,
			node: node,
		}
	}
}

// Nodes is a linked list of registry.Node.
type Nodes struct {
	next *Nodes
	node *registry.Node
}

// Range likes the Range method of sync.Map.
// It calls f serially for each node in Nodes. Range stops on f failed.
// Range returns true after traversing all nodes, false otherwise.
// Users should not change n in f.
func (nodes *Nodes) Range(f func(n *registry.Node) bool) bool {
	if nodes == nil {
		return true
	}
	if f(nodes.node) {
		return nodes.next.Range(f)
	}
	return false
}
