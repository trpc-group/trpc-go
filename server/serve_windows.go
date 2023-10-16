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

//go:build windows
// +build windows

package server

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"trpc.group/trpc-go/trpc-go/log"
)

// Serve implements Service, starting all services that belong to this server
func (s *Server) Serve() error {
	defer log.Sync()
	if len(s.services) == 0 {
		panic("service empty")
	}
	s.signalCh = make(chan os.Signal)
	s.closeCh = make(chan struct{})

	var (
		mu     sync.Mutex
		svrErr error
	)
	for name, service := range s.services {
		go func(name string, service Service) {
			if err := service.Serve(); err != nil {
				mu.Lock()
				svrErr = multierror.Append(svrErr, err).ErrorOrNil()
				mu.Unlock()
				s.failedServices.Store(name, service)
				time.Sleep(time.Millisecond * 300)
				s.signalCh <- syscall.SIGTERM
			}
		}(name, service)
	}

	signal.Notify(s.signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)
	select {
	case <-s.closeCh:
	case <-s.signalCh:
	}

	s.tryClose()
	if svrErr != nil {
		log.Errorf("service serve errors: %+v", svrErr)
	}
	return svrErr
}
