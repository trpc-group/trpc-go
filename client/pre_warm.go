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

package client

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/prewarm"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (c *client) init(ctx context.Context, opt ...Option) error {
	ctx, msg := codec.EnsureMessage(ctx)
	opts, err := c.getOptions(msg, opt...)
	if err != nil {
		return err
	}
	c.updateMsg(msg, opts)
	ctx = contextWithOptions(ctx, opts)
	if !prewarm.Enabled(ctx, opts.CallOptions...) {
		return nil
	}
	if err := c.preWarm(ctx, msg, opts); err != nil {
		return fmt.Errorf("prewarm: %w", err)
	}
	return nil
}

// Init performs initialization operations on the client using the provided options.
func (c *client) Init(ctx context.Context, opt ...Option) error {
	return c.init(ctx, opt...)
}

func (c *client) preWarm(ctx context.Context, msg codec.Msg, opts *Options) error {
	roundTripOpts := &transport.RoundTripOptions{
		Pool:        connpool.DefaultConnectionPool,
		Multiplexed: multiplexed.DefaultMultiplexedPool,
		Msg:         codec.Message(ctx),
	}
	for _, o := range opts.CallOptions {
		o(roundTripOpts)
	}
	if err := prewarm.ValidateConfig(roundTripOpts); err != nil {
		return fmt.Errorf("validate prewarm config: %w", err)
	}

	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok && roundTripOpts.PreWarm.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, roundTripOpts.PreWarm.Timeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("prewarm context error before get node list: %w", err)
	}

	nodeList, err := c.preWarmNodeList(ctx, msg, opts)
	if err != nil {
		return fmt.Errorf("get prewarm node list: %w", err)
	}
	if err := prewarm.PreWarm(ctx, nodeList, roundTripOpts); err != nil {
		return err
	}
	return nil
}

func (c *client) preWarmNodeList(ctx context.Context, msg codec.Msg, opts *Options) ([]*registry.Node, error) {
	var err error
	if _, ok := opts.Selector.(*selector.TrpcSelector); ok {
		var nodeList []*registry.Node
		nodeList, err = listNodes(ctx, opts)
		if err == nil && len(nodeList) > 0 {
			return nodeList, nil
		}
	}

	selectOpts := opts.clone()
	selectOpts.rebuildSliceCapacity()
	node, selectErr := selectNode(ctx, msg, selectOpts)
	if selectErr != nil {
		if err != nil {
			return nil, fmt.Errorf("%v; fallback select node: %w", err, selectErr)
		}
		return nil, selectErr
	}
	return []*registry.Node{node}, nil
}

func listNodes(ctx context.Context, opts *Options) ([]*registry.Node, error) {
	selectorOpts := selector.Options{
		Discovery:     discovery.DefaultDiscovery,
		ServiceRouter: servicerouter.DefaultServiceRouter,
	}
	selectOptions := append([]selector.Option{}, opts.SelectOptions...)
	selectOptions = append(selectOptions, selector.WithContext(ctx))
	for _, opt := range selectOptions {
		opt(&selectorOpts)
	}
	if selectorOpts.Discovery == nil {
		return nil, errs.NewFrameError(errs.RetClientRouteErr, "client prewarm: discovery not exists")
	}
	nodeList, err := selectorOpts.Discovery.List(opts.endpoint, selectorOpts.DiscoveryOptions...)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientRouteErr, "client prewarm discovery List: "+err.Error())
	}
	if selectorOpts.ServiceRouter != nil {
		nodeList, err = selectorOpts.ServiceRouter.Filter(opts.endpoint, nodeList, selectorOpts.ServiceRouterOptions...)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientRouteErr, "client prewarm service router Filter: "+err.Error())
		}
	}
	if len(nodeList) == 0 {
		return nil, errs.NewFrameError(errs.RetClientRouteErr, "client prewarm: node list is empty")
	}
	return nodeList, nil
}
