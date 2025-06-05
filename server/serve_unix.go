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

//go:build aix || darwin || dragonfly || freebsd || netbsd || openbsd || solaris || linux
// +build aix darwin dragonfly freebsd netbsd openbsd solaris linux

package server

import (
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/sys/unix"
	ierror "trpc.group/trpc-go/trpc-go/internal/error"
	igr "trpc.group/trpc-go/trpc-go/internal/graceful"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

// DefaultServerCloseSIG are signals that trigger server shutdown.
var DefaultServerCloseSIG = []os.Signal{unix.SIGINT, unix.SIGTERM, unix.SIGSEGV}

// DefaultServerGracefulSIG is signal that triggers server graceful restart.
var DefaultServerGracefulSIG = unix.SIGUSR2

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
				s.signalCh <- unix.SIGTERM
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
	if sig == DefaultServerGracefulSIG && !s.disableGracefulRestart {
		s.muxRestartHook.Lock()
		for _, f := range s.beforeGracefulRestartHooks {
			f()
		}
		s.muxRestartHook.Unlock()
		if err := igr.Restart(prepareGracefulRestart()); err != nil {
			panic(err)
		}
		s.tryClose(ierror.GracefulRestart)
	} else {
		s.tryClose(ierror.NormalShutdown)
	}

	if err != nil {
		log.Errorf(`service serve errors: %+v
Note: it is normal to have "use of closed network connection" error during hot restart.
DO NOT panic (Reference: internal issues/791).`, err)
	}
	return err
}

// StartNewProcess starts a new process.
// Deprecated: This function has been deprecated and will be removed in a future version.
func (s *Server) StartNewProcess(args ...string) (uintptr, error) {
	pid := os.Getpid()
	log.Infof("process: %d, received graceful restart signal, so restart the process", pid)

	fds := prepareGracefulRestart()
	childPID, err := syscall.ForkExec(os.Args[0], append(os.Args, args...), &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: fds,
	})
	if err != nil {
		log.Errorf("process: %d, failed to forkexec with err: %s", pid, err.Error())
		return 0, err
	}

	return uintptr(childPID), nil
}

// SetDisableGracefulRestart sets whether to disable graceful restart or not.
// SetDisableGracefulRestart(true) will not clear gracefulRestartHooks.
func (s *Server) SetDisableGracefulRestart(disable bool) {
	s.muxRestartHook.Lock()
	s.disableGracefulRestart = disable
	s.muxRestartHook.Unlock()
}

func prepareGracefulRestart() []uintptr {
	// Only tnet still uses this function to get listener fds.
	listenersFds := transport.GetListenersFds()
	files := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}

	os.Setenv(transport.EnvGraceRestart, "1")
	os.Setenv(transport.EnvGraceFirstFd, strconv.Itoa(len(files)))
	os.Setenv(transport.EnvGraceRestartFdNum, strconv.Itoa(len(listenersFds)))
	os.Setenv(transport.EnvGraceRestartPPID, strconv.Itoa(os.Getpid()))

	return append(files, prepareListenFds(listenersFds)...)
}

func prepareListenFds(fds []*transport.ListenFd) []uintptr {
	files := make([]uintptr, 0, len(fds))
	for _, fd := range fds {
		files = append(files, fd.Fd)
	}
	return files
}
