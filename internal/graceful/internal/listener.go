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

//go:build !windows

package graceful

import (
	"fmt"
	"net"
	"sync"

	iprotocol "trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/reuseport"
)

var inheritListeners = NewSafe[map[string]map[string]net.Listener](nil)
var listeners = NewSafe[map[string]map[string]net.Listener](nil)

// Listen creates a net.Listener on network address.
// If we have inherited a Listener from parent process, then return the inherited one.
// Otherwise, create a new net.Listener by net.Listen or reuseport.Listen.
// In either case, the listener is stored to a global variable listeners and is ready
// to pass to child process on next graceful restart.
func Listen(network, address string, reusePort bool) (net.Listener, error) {
	inheritListeners.Lock()
	if ls, ok := inheritListeners.T[network]; ok {
		if l, ok := ls[address]; ok {
			listeners.Lock()
			listeners.T = appendMap(listeners.T, network, address, l)
			listeners.Unlock()
			delete(ls, address)
			inheritListeners.Unlock()
			return l, nil
		}
	}
	inheritListeners.Unlock()

	var l net.Listener
	var err error

	if reusePort && network != iprotocol.UNIX {
		l, err = reuseport.Listen(network, address)
		if err != nil {
			return nil, fmt.Errorf("%s reuseport error: %v", network, err)
		}
	} else {
		l, err = net.Listen(network, address)
	}
	if err != nil {
		return nil, err
	}
	conns := make(chan net.Conn)
	close(conns)
	l = NewListener(l, network, address, conns)
	listeners.Lock()
	listeners.T = appendMap(listeners.T, network, address, l)
	listeners.Unlock()
	return l, nil
}

// NewListener creates a new Listener based on net.Listener.
// connReceiver is used to receive subsequent connections from parent process.
// Closing connReceiver indicates that all parent connections, that belongs
// to this Listener have been transmitted.
func NewListener(l net.Listener, network, address string, connReceiver chan net.Conn) *Listener {
	return &Listener{
		network: network,
		address: address,
		l:       l,
		conns:   connReceiver,
		accepts: make(chan Result[net.Conn]),
	}
}

// Listener accepts connections, which may comes from parent process or a new connection from kernel.
type Listener struct {
	// we explicitly store network and address here. Though net.Listener can return the address,
	// but it may be different from Listen function. For graceful restart, the listener is
	// distinguished by Listen network and address, not net.Listener.
	network string
	address string
	l       net.Listener
	conns   chan net.Conn

	mu        sync.Mutex
	recvState recvState
	accepts   chan Result[net.Conn]
	dangling  int // Number of connections pending consumption, originating from kernel Accept.
	receiving int
}

type recvState = uint32

const (
	receiving recvState = iota
	draining
	received
)

// Accept accepts a new net.Conn, which may comes from parent process or a new connection from kernel.
// Accept is concurrent safe.
func (l *Listener) Accept() (conn net.Conn, err error) {
	defer func() {
		if err == nil {
			if _, ok := conn.(*Conn); ok {
				return
			}
			conn = NewConn(conn, newConnOnClosed(l.network, l.address))
		}
	}()

	// this mutex protect recvState and is unlocked in acceptReceiving.
	l.mu.Lock()
	switch l.recvState {
	case receiving:
		return l.acceptReceiving()
	case draining:
		return l.acceptDraining()
	case received:
		l.mu.Unlock()
		return l.l.Accept()
	default:
		panic("unreachable")
	}
}

// Close closes the underlying net.Listener.
func (l *Listener) Close() error {
	listeners.Lock()
	deleteMap(listeners.T, l.network, l.address)
	listeners.Unlock()
	return l.l.Close()
}

// Addr returns the address of underlying net.Listener.
func (l *Listener) Addr() net.Addr {
	return l.l.Addr()
}

func (l *Listener) acceptReceiving() (net.Conn, error) {
	l.receiving++
	// Plan to consume one connection from l.accepts.
	if l.dangling > 0 {
		l.dangling--
	} else {
		go func() {
			conn, err := l.l.Accept()
			l.accepts <- Result[net.Conn]{Ok: conn, Err: err}
		}()
	}
	l.mu.Unlock()

	select {
	case conn, ok := <-l.conns:
		l.mu.Lock()
		if !ok {
			l.recvState = draining
			l.receiving--
			l.mu.Unlock()
			res := <-l.accepts
			return res.Ok, res.Err
		}
		// Compensate by incrementing dangling if no connection was consumed from l.accepts.
		l.dangling++
		l.receiving--
		l.mu.Unlock()
		return conn, nil
	case res := <-l.accepts:
		l.mu.Lock()
		l.receiving--
		l.mu.Unlock()
		return res.Ok, res.Err
	}
}

func (l *Listener) acceptDraining() (net.Conn, error) {
	// Prioritize consuming connections from the accepts channel.
	if l.receiving == 0 && l.dangling > 0 {
		l.dangling--
		l.mu.Unlock()
		res := <-l.accepts
		return res.Ok, res.Err
	}
	if l.receiving == 0 && l.dangling == 0 {
		l.recvState = received
	}
	l.mu.Unlock()
	return l.l.Accept()
}

// Unwrap unwraps and giving the underlying net.Listener.
func (l *Listener) Unwrap() net.Listener { return l.l }

// Result represents either ok or err.
type Result[T any] struct {
	Ok  T
	Err error
}
