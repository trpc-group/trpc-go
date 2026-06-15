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

// Package prewarm implements client connection prewarming.
package prewarm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	"trpc.group/trpc-go/trpc-go/transport"
)

var supportedNetworks = map[string]struct{}{
	"tcp":  {},
	"tcp4": {},
	"tcp6": {},
	"unix": {},
}

// Enabled checks if prewarming is enabled.
func Enabled(_ context.Context, opts ...transport.RoundTripOption) bool {
	roundTripOpts := &transport.RoundTripOptions{}
	for _, o := range opts {
		o(roundTripOpts)
	}
	return roundTripOpts.PreWarm != nil && roundTripOpts.PreWarm.ConnsPerNode > 0
}

// ValidateConfig validates the configuration for prewarming.
func ValidateConfig(opts *transport.RoundTripOptions) error {
	if opts == nil {
		return errors.New("prewarm options empty")
	}
	if _, ok := supportedNetworks[opts.Network]; !ok {
		return fmt.Errorf("network %s not supported", opts.Network)
	}
	if opts.DisableConnectionPool {
		return errors.New("prewarm is not supported when connection pool is disabled")
	}
	if opts.PreWarm == nil || opts.PreWarm.ConnsPerNode <= 0 {
		return errors.New("prewarm connections per node must be greater than zero")
	}
	if opts.EnableMultiplexed {
		if opts.Multiplexed == nil {
			return errors.New("prewarm multiplexed pool empty")
		}
		if opts.FramerBuilder == nil {
			return errors.New("prewarm framer builder empty")
		}
		return nil
	}
	if opts.Pool == nil {
		return errors.New("prewarm connection pool empty")
	}
	return nil
}

// PreWarm prewarms the multiplexed or connection pool.
func PreWarm(ctx context.Context, nodeList []*registry.Node, opts *transport.RoundTripOptions) error {
	if len(nodeList) == 0 {
		return nil
	}
	if err := ValidateConfig(opts); err != nil {
		return err
	}
	if opts.EnableMultiplexed {
		return preWarm[muxConn](ctx, nodeList, opts, &muxPool{pool: opts.Multiplexed})
	}
	return preWarm[net.Conn](ctx, nodeList, opts, &connPool{pool: opts.Pool})
}

type prewarmConn interface {
	Close() error
}

type prewarmPool[C prewarmConn] interface {
	Get(ctx context.Context, network, address string, opts *transport.RoundTripOptions) (C, error)
}

func preWarm[C prewarmConn](
	ctx context.Context,
	nodeList []*registry.Node,
	opts *transport.RoundTripOptions,
	pool prewarmPool[C],
) error {
	parallelism := runtime.GOMAXPROCS(0)
	if parallelism <= 0 {
		parallelism = 1
	}
	sem := make(chan struct{}, parallelism)
	errs := make(chan error, len(nodeList)*opts.PreWarm.ConnsPerNode)
	var wg sync.WaitGroup

	for _, node := range nodeList {
		if node == nil || node.Address == "" {
			continue
		}
		for i := 0; i < opts.PreWarm.ConnsPerNode; i++ {
			wg.Add(1)
			go func(node *registry.Node) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					errs <- fmt.Errorf("prewarm canceled before establish connection for %s: %w", node.Address, ctx.Err())
					return
				}
				if err := establish(ctx, node, opts, pool); err != nil {
					errs <- err
				}
			}(node)
		}
	}

	wg.Wait()
	close(errs)

	var err error
	for e := range errs {
		err = multierror.Append(err, e)
	}
	return err
}

func establish[C prewarmConn](
	ctx context.Context,
	node *registry.Node,
	opts *transport.RoundTripOptions,
	pool prewarmPool[C],
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("prewarm canceled before establish connection for %s: %w", node.Address, err)
	}

	taskCtx, msg := codec.WithCloneContextAndMessage(ctx)
	defer codec.PutBackMessage(msg)
	if deadline, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithDeadline(taskCtx, deadline)
		defer cancel()
	}

	network := opts.Network
	if node.Network != "" {
		network = node.Network
	}
	conn, err := pool.Get(taskCtx, network, node.Address, opts)
	if err != nil {
		return fmt.Errorf("establish connection for %s/%s: %w", network, node.Address, err)
	}
	if err := taskCtx.Err(); err != nil {
		closeErr := conn.Close()
		if closeErr != nil {
			return multierror.Append(
				fmt.Errorf("prewarm canceled after establish connection for %s/%s: %w", network, node.Address, err),
				fmt.Errorf("close connection for %s/%s: %w", network, node.Address, closeErr),
			)
		}
		return fmt.Errorf("prewarm canceled after establish connection for %s/%s: %w", network, node.Address, err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close connection for %s/%s: %w", network, node.Address, err)
	}
	return nil
}

type connPool struct {
	pool connpool.Pool
}

func (p *connPool) Get(
	ctx context.Context,
	network string,
	address string,
	opts *transport.RoundTripOptions,
) (net.Conn, error) {
	getOpts := connpool.NewGetOptions()
	getOpts.WithContext(ctx)
	getOpts.WithFramerBuilder(opts.FramerBuilder)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithCertProvider(opts.TLSCertProvider)
	getOpts.WithLocalAddr(opts.LocalAddr)
	getOpts.WithDialTimeout(opts.DialTimeout)
	getOpts.WithProtocol(opts.Protocol)
	return p.pool.Get(network, address, getOpts)
}

type muxConn struct {
	multiplexed.MuxConn
}

func (c muxConn) Close() error {
	c.MuxConn.Close()
	return nil
}

type muxPool struct {
	pool multiplexed.Pool
	vid  uint32
}

func (p *muxPool) Get(
	ctx context.Context,
	network string,
	address string,
	opts *transport.RoundTripOptions,
) (muxConn, error) {
	getOpts := multiplexed.NewGetOptions()
	getOpts.WithVID(atomic.AddUint32(&p.vid, 1))
	fp, ok := opts.FramerBuilder.(multiplexed.FrameParser)
	if !ok {
		return muxConn{}, errors.New("frame builder does not implement multiplexed.FrameParser")
	}
	getOpts.WithFrameParser(fp)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithCertProvider(opts.TLSCertProvider)
	getOpts.WithLocalAddr(opts.LocalAddr)
	conn, err := p.pool.GetMuxConn(ctx, network, address, getOpts)
	if err != nil {
		return muxConn{}, err
	}
	return muxConn{MuxConn: conn}, nil
}
