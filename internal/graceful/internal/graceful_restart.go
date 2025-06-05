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

// Package graceful restarts a new process and pass all
// tcp and unix domain socket to it.
//
// This package uses global variables. Because we are starting
// a new process and pass original network sockets to it as many
// as possible, it has no meaning to restart multiple times.
package graceful

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"

	"trpc.group/trpc-go/trpc-go/internal/atomic"
)

const gracefulRestartFdEnvKey = "graceful_restart_fd"

// writerToChildProcess is stored after Restart is called and used when closing conn.
// We can not avoid this global variable, since Listen and Restart are
// package level functions, and this variable is a link between them.
var writerToChildProcess atomic.Pointer[Safe[*Writer]]

func init() {
	init1()
}

// init1 is defined for unit test.
var init1 = func() {
	initProtocols()

	gracefulRestartFdEnv := os.Getenv(gracefulRestartFdEnvKey)
	if gracefulRestartFdEnv == "" {
		return
	}
	if err := initInherit(gracefulRestartFdEnv); err != nil {
		panic(fmt.Sprintf("graceful start: init inherit: %+v", err))
	}
}

func initInherit(gracefulRestartFdEnv string) error {
	r, w, err := newRPCReaderWriter(gracefulRestartFdEnv)
	if err != nil {
		return fmt.Errorf("failed to init rpc: %w", err)
	}
	rls, err := receiveAllListeners(r, w)
	if err != nil {
		return fmt.Errorf("recive all listeners: %w", err)
	}
	var listenerConns map[string]map[string]chan net.Conn
	for _, rl := range rls {
		switch rl.Network {
		case "tcp", "tcp4", "tcp6", "unix":
			file := os.NewFile(uintptr(rl.Fd), "")
			l, err := net.FileListener(file)
			if err != nil {
				return fmt.Errorf("convert file to net listener: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close file: %w", err)
			}
			conns := make(chan net.Conn)
			listenerConns = appendMap(listenerConns, rl.Network, rl.Address, conns)
			inheritListeners.Lock()
			inheritListeners.T = appendMap(inheritListeners.T, rl.Network, rl.Address,
				net.Listener(NewListener(l, rl.Network, rl.Address, conns)))
			inheritListeners.Unlock()
		case "udp", "udp4", "udp6":
			file := os.NewFile(uintptr(rl.Fd), "")
			conn, err := net.FilePacketConn(file)
			if err != nil {
				return fmt.Errorf("convert file to net.PacketConn: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close file: %w", err)
			}
			conn, err = NewPacketConn(conn, rl.Network, rl.Address)
			if err != nil {
				return fmt.Errorf("new Packet conn: %w", err)
			}
			inheritPacketConns.Lock()
			inheritPacketConns.T = appendMap(inheritPacketConns.T, rl.Network, rl.Address, conn)
			inheritPacketConns.Unlock()
		default:
			return fmt.Errorf("unexpected network %v", rl.Network)
		}
	}

	go func() {
		receivingConnections(r, listenerConns)
		for _, addrConns := range listenerConns {
			for _, conns := range addrConns {
				close(conns)
			}
		}
	}()

	return nil
}

// Restart restarts a new process.
func Restart(files []uintptr) error {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("failed to create unix domain socket: %w", err)
	}

	procAttr := syscall.ProcAttr{
		Env:   append(os.Environ(), fmt.Sprintf(gracefulRestartFdEnvKey+"=%d", len(files))),
		Files: append(files, uintptr(fds[1])),
	}
	_, err = forkExec(os.Args[0], os.Args, &procAttr)
	if err != nil {
		return fmt.Errorf("failed to ForkExec: %w", err)
	}

	w := NewRpcWriter(fds[0])
	if err := sendListenersWaitAck(w, NewRpcReader(fds[0])); err != nil {
		return fmt.Errorf("failed to sendListenersWaitAck: %w", err)
	}

	// Transfer connection to child process only after Restart returns.
	writerToChildProcess.Store(NewSafe(w))
	return nil
}

func newRPCReaderWriter(gracefulRestartFdEnv string) (*Reader, *Writer, error) {
	fd, err := strconv.Atoi(gracefulRestartFdEnv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse graceful restart fd env %s: %w", gracefulRestartFdEnv, err)
	}
	return NewRpcReader(fd), NewRpcWriter(fd), nil
}

func receiveAllListeners(r *Reader, w *Writer) ([]receivedListener, error) {
	rls, err := receiveListeners(r)
	if err != nil {
		return nil, err
	}

	var req protocol = AckListeners{Cnt: len(rls)}
	if err := w.Encode(&req); err != nil {
		return nil, fmt.Errorf("failed to ack listeners: %w", err)
	}
	if err := w.Flush(nil); err != nil {
		return nil, fmt.Errorf("failed to flush ack listeners: %w", err)
	}
	return rls, nil
}

func receiveListeners(r *Reader) ([]receivedListener, error) {
	var request protocol
	if err := r.Decode(&request); err != nil {
		return nil, fmt.Errorf("failed to decode init listeners: %w", err)
	}

	var rls []receivedListener
	switch req := request.(type) {
	case ReqListeners:
		fds := r.GetFds()
		if len(req.Listeners) != len(fds) {
			return nil, fmt.Errorf("len of listeners fds %d does not match metadata %d", len(fds), len(req.Listeners))
		}
		for i, fd := range fds {
			rls = append(rls, receivedListener{
				Network: req.Listeners[i].Network,
				Address: req.Listeners[i].Address,
				Fd:      fd,
			})
		}
		if req.Continue {
			crls, err := receiveListeners(r)
			return append(rls, crls...), err
		}
		return rls, nil
	default:
		return nil, fmt.Errorf("expected %T, but got %T", ReqListeners{}, request)
	}
}

// receivingConnections receives connections from parent process and deliver them to proper listeners.
func receivingConnections(r *Reader, lconns map[string]map[string]chan net.Conn) {
	for {
		var request protocol
		if err := r.Decode(&request); err != nil {
			if !errors.Is(err, io.EOF) {
				stdErrf("stop receiving connections for unexpected error: %s", err.Error())
			}
			return
		}
		switch req := request.(type) {
		case ReqConn:
			fds := r.GetFds()
			if len(fds) != 1 {
				stdErrf("conn should be received one by one, but got %d", len(fds))
				return
			}

			file := os.NewFile(uintptr(fds[0]), "")
			conn, err := net.FileConn(file)
			if err := file.Close(); err != nil {
				stdErrf("failed to close temporary file: %s", err.Error())
				return
			}
			if err != nil {
				stdErrf("failed to create net conn: %s", err.Error())
				return
			}

			if addrConns, ok := lconns[req.Network]; ok {
				if conns, ok := addrConns[req.Address]; ok {
					conns <- NewConn(conn, newConnOnClosed(req.Network, req.Address))
					continue
				}
			}

			// receive a connection which belongs none of inherit listeners, just close it.
			if err := conn.Close(); err != nil {
				stdErrf("failed to close orphan conn: %s", err.Error())
			}
		default:
			stdErrf("expected %T, but got %T", req, request)
			return
		}
	}
}

func sendListenersWaitAck(w *Writer, r *Reader) error {
	tcpReq, tcpFds, err := getListenerFDs(listeners)
	if err != nil {
		return fmt.Errorf("get tcp req and fds: %w", err)
	}
	udpReq, udpFds, err := getListenerFDs(packetConns)
	if err != nil {
		return fmt.Errorf("get udp req and fds: %w", err)
	}
	req := append(tcpReq, udpReq...)
	fds := append(tcpFds, udpFds...)

	if err := sendListeners(w, req, fds); err != nil {
		return fmt.Errorf("failed to send listeners: %w", err)
	}

	var rsp protocol
	if err := r.Decode(&rsp); err != nil {
		return fmt.Errorf("failed to recv rsp: %w", err)
	}
	ack, ok := rsp.(AckListeners)
	if !ok {
		return fmt.Errorf("expected %T, but got %T", ack, rsp)
	}
	if ack.Cnt != len(req) {
		return fmt.Errorf("child recv %d conns which does not match parent send %d", ack.Cnt, len(req))
	}

	return nil
}

func getListenerFDs[T any](listeners *Safe[map[string]map[string]T]) ([]ReqListener, []int, error) {
	var rls []ReqListener
	var fds []int
	listeners.Lock()
	defer listeners.Unlock()
	for network, ls := range listeners.T {
		for address, l := range ls {
			rls = append(rls, ReqListener{
				Network: network,
				Address: address,
			})
			fd, err := sysConnFd(l)
			if err != nil {
				return nil, nil, fmt.Errorf("get sys conn fd from %v:%v: %w", network, address, err)
			}
			fds = append(fds, fd)
		}
	}
	return rls, fds, nil
}

func sendListeners(w *Writer, ls []ReqListener, fds []int) error {
	for {
		end := maxSCMDataLen
		if len(ls) < end {
			end = len(ls)
		}

		var req protocol = ReqListeners{
			Listeners: ls[:end],
			Continue:  end != len(ls),
		}
		if err := w.Encode(&req); err != nil {
			return fmt.Errorf("failed to encode req: %w", err)
		}
		if err := w.Flush(fds[:end]); err != nil {
			return fmt.Errorf("failed to flush: %w", err)
		}

		ls, fds = ls[end:], fds[end:]
		if len(ls) == 0 {
			break
		}
	}
	return nil
}

func newConnOnClosed(network, address string) func(net.Conn) {
	return func(c net.Conn) {
		sw := writerToChildProcess.Load()
		if sw == nil {
			return
		}

		fd, err := sysConnFd[net.Conn](c)
		if err != nil {
			stdErrf("failed to retrieve underlying fd: %s", err.Error())
			return
		}

		var req protocol = ReqConn{
			Network: network,
			Address: address,
		}
		sw.Lock()
		defer sw.Unlock()
		if err := sw.T.Encode(&req); err != nil {
			stdErrf("failed to encode ReqConn %s: %s: %s", network, address, err.Error())
			return
		}
		if err := sw.T.Flush([]int{fd}); err != nil {
			stdErrf("failed to flush: %s", err.Error())
			return
		}
	}
}

type receivedListener struct {
	Network string
	Address string
	Fd      int
}

// forkExec is defined for unit test.
var forkExec = syscall.ForkExec
