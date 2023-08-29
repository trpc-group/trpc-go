// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package server

import (
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

// DefaultServerCloseSIG are signals that trigger server shutdown.
var DefaultServerCloseSIG = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV}

// DefaultServerGracefulSIG is signal that triggers server graceful restart.
var DefaultServerGracefulSIG = syscall.SIGUSR2

// Serve implements Service, starting all services that belong to the server.
func (s *Server) Serve() error {
	defer log.Sync()
	if len(s.services) == 0 {
		panic("service empty")
	}
	s.signalCh = make(chan os.Signal)
	s.closeCh = make(chan struct{})

	var (
		mu  sync.Mutex
		err error
	)
	for name, service := range s.services {
		go func(n string, srv Service) {
			if e := srv.Serve(); e != nil {
				mu.Lock()
				err = multierror.Append(err, e).ErrorOrNil()
				mu.Unlock()
				s.failedServices.Store(n, srv)
				time.Sleep(time.Millisecond * 300)
				s.signalCh <- syscall.SIGTERM
			}
		}(name, service)
	}
	signal.Notify(s.signalCh, append(DefaultServerCloseSIG, DefaultServerGracefulSIG)...)

	var sig os.Signal
	select {
	case <-s.closeCh:
	case sig = <-s.signalCh:
	}

	// graceful restart.
	if sig == DefaultServerGracefulSIG {
		if _, err := s.StartNewProcess(); err != nil {
			panic(err)
		}
	}
	// try to close server.
	s.tryClose()

	if err != nil {
		log.Errorf(`service serve errors: %+v
Note: it is normal to have "use of closed network connection" error during hot restart.
DO NOT panic.`, err)
	}
	return err
}

// StartNewProcess starts a new process.
func (s *Server) StartNewProcess(args ...string) (uintptr, error) {
	pid := os.Getpid()
	log.Infof("process: %d, received graceful restart signal, so restart the process", pid)

	// pass tcp listeners' Fds and udp conn's Fds
	listenersFds := transport.GetListenersFds()

	files := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}

	os.Setenv(transport.EnvGraceRestart, "1")
	os.Setenv(transport.EnvGraceFirstFd, strconv.Itoa(len(files)))
	os.Setenv(transport.EnvGraceRestartFdNum, strconv.Itoa(len(listenersFds)))
	os.Setenv(transport.EnvGraceRestartPPID, strconv.Itoa(os.Getpid()))

	files = append(files, prepareListenFds(listenersFds)...)

	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: files,
	}

	os.Args = append(os.Args, args...)
	childPID, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.Errorf("process: %d, failed to forkexec with err: %s", pid, err.Error())
		return 0, err
	}

	for _, f := range listenersFds {
		f.OriginalListenCloser.Close()
	}
	return uintptr(childPID), nil
}

func prepareListenFds(fds []*transport.ListenFd) []uintptr {
	files := make([]uintptr, 0, len(fds))
	for _, fd := range fds {
		files = append(files, fd.Fd)
	}
	return files
}
