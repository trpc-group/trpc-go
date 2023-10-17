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

package loadbalance

import (
	"time"

	"trpc.group/trpc-go/trpc-go/internal/rand"
	"trpc.group/trpc-go/trpc-go/naming/bannednodes"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func init() {
	Register(LoadBalanceRandom, NewRandom())
}

// Random is the random load balance algorithm.
type Random struct {
	safeRand *rand.SafeRand
}

// NewRandom creates a new Random.
func NewRandom() *Random {
	return &Random{
		safeRand: rand.NewSafeRand(time.Now().UnixNano()),
	}
}

// Select picks a node from nodes randomly. Select tries its best to choose a node not in
// bannedNodes of context.
func (b *Random) Select(
	serviceName string,
	nodes []*registry.Node,
	opts ...Option,
) (node *registry.Node, err error) {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}

	if o.Ctx == nil {
		return b.chooseOne(nodes)
	}

	bans, mandatory, ok := bannednodes.FromCtx(o.Ctx)
	if !ok {
		return b.chooseOne(nodes)
	}

	defer func() {
		if err == nil {
			bannednodes.Add(o.Ctx, node)
		}
	}()

	node, err = b.chooseUnbanned(nodes, bans)
	if !mandatory && err == ErrNoServerAvailable {
		return b.chooseOne(nodes)
	}
	return node, err
}

func (b *Random) chooseOne(nodes []*registry.Node) (*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, ErrNoServerAvailable
	}
	return nodes[b.safeRand.Intn(len(nodes))], nil
}

func (b *Random) chooseUnbanned(
	nodes []*registry.Node,
	bans *bannednodes.Nodes,
) (*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, ErrNoServerAvailable
	}
	i := b.safeRand.Intn(len(nodes))
	if !bans.Range(func(n *registry.Node) bool {
		return n.Address != nodes[i].Address
	}) {
		return b.chooseUnbanned(append(nodes[:i], nodes[i+1:]...), bans)
	}
	return nodes[i], nil
}
