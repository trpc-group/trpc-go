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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

// Package tnet provides tRPC-Go transport implementation for tnet networking framework.
package tnet

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"trpc.group/trpc-go/tnet"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/addrutil"
	"trpc.group/trpc-go/trpc-go/internal/keeporder/actor"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

const transportName = protocol.TNET

func init() {
	transport.RegisterServerTransport(transportName, DefaultServerTransport)
}

type serverTransport struct {
	addrToConn map[string]*tcpConn
	m          sync.RWMutex
	opts       *ServerTransportOptions
}

// DefaultServerTransport is the default implementation of tnet server transport.
var DefaultServerTransport = NewServerTransport(WithReusePort(true))

// NewServerTransport creates tnet server transport.
func NewServerTransport(opts ...ServerTransportOption) transport.ServerTransport {
	option := &ServerTransportOptions{}
	for _, o := range opts {
		o(option)
	}
	return &serverTransport{addrToConn: make(map[string]*tcpConn), opts: option}
}

// ListenAndServe begins listen and serve.
func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) error {
	lsOpts, err := buildListenServeOptions(opts...)
	if err != nil {
		return err
	}
	networks := strings.Split(lsOpts.Network, ",")
	for _, network := range networks {
		lsOpts.Network = network
		if err := s.switchNetworkToServe(ctx, lsOpts); err != nil {
			log.Info("switch to gonet default transport, ", err)
			opts = append(opts, transport.WithListenNetwork(network))
			return transport.DefaultServerTransport.ListenAndServe(ctx, opts...)
		}
	}
	return nil
}

// Send implements ServerStreamTransport, sends stream messages.
func (s *serverTransport) Send(ctx context.Context, req []byte) error {
	msg := codec.Message(ctx)
	raddr := msg.RemoteAddr()
	laddr := msg.LocalAddr()
	if raddr == nil || laddr == nil {
		return errs.NewFrameError(errs.RetServerSystemErr,
			fmt.Sprintf("Address is invalid, local: %s, remote: %s", laddr, raddr))
	}
	tc, ok := s.loadConn(addrutil.AddrToKey(laddr, raddr))
	if !ok {
		return errs.NewFrameError(errs.RetServerSystemErr, "can't find conn by addr")
	}
	if _, err := tc.rawConn.Write(req); err != nil {
		tc.close()
		s.Close(ctx)
		return err
	}
	return nil
}

// Close closes transport, and cleans up cached connections.
func (s *serverTransport) Close(ctx context.Context) {
	msg := codec.Message(ctx)
	raddr := msg.RemoteAddr()
	laddr := msg.LocalAddr()
	s.deleteConn(addrutil.AddrToKey(laddr, raddr))
}

func (s *serverTransport) switchNetworkToServe(ctx context.Context, opts *transport.ListenServeOptions) error {
	switch opts.Network {
	case protocol.TCP, protocol.TCP4, protocol.TCP6:
		log.Infof("service: %s is using tnet tcp transport, current number of pollers: %d",
			opts.ServiceName, tnet.NumPollers())
		if err := s.listenAndServeTCP(ctx, opts); err != nil {
			return fmt.Errorf("tnet: listen and serve tcp: %w", err)
		}
	case protocol.UDP, protocol.UDP4, protocol.UDP6:
		log.Infof("service: %s is using tnet udp transport, current number of pollers: %d",
			opts.ServiceName, tnet.NumPollers())
		if err := s.listenAndServeUDP(ctx, opts); err != nil {
			return fmt.Errorf("tnet: listen and serve udp: %w", err)
		}
	default:
		return fmt.Errorf("tnet server transport doesn't support network type [%s]", opts.Network)
	}
	return nil
}

func (s *serverTransport) deleteConn(addr string) {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.addrToConn, addr)
}

func (s *serverTransport) loadConn(addr string) (*tcpConn, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	tc, ok := s.addrToConn[addr]
	return tc, ok
}

func (s *serverTransport) storeConn(addr string, tc *tcpConn) {
	s.m.Lock()
	defer s.m.Unlock()
	s.addrToConn[addr] = tc
}

func buildListenServeOptions(opts ...transport.ListenServeOption) (*transport.ListenServeOptions, error) {
	lsOpts := &transport.ListenServeOptions{}
	for _, o := range opts {
		o(lsOpts)
	}
	if lsOpts.OrderedGroups == nil {
		// Use actor.Default as the default implementation for ordered groups.
		lsOpts.OrderedGroups = actor.Default
	}
	if lsOpts.FramerBuilder == nil {
		return nil, errors.New("transport FramerBuilder empty")
	}
	return lsOpts, nil
}
