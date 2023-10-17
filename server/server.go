// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package server provides a framework for managing multiple services within a single process.
// A server process may listen on multiple ports, providing different services on different ports.
package server

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"
)

// Server is a tRPC server.
// One process, one server. A server may offer one or more services.
type Server struct {
	MaxCloseWaitTime time.Duration // max waiting time when closing server

	services map[string]Service // k=serviceName,v=Service

	mux sync.Mutex // guards onShutdownHooks
	// onShutdownHooks are hook functions that would be executed when server is
	// shutting down (before closing all services of the server).
	onShutdownHooks []func()

	failedServices sync.Map
	signalCh       chan os.Signal
	closeCh        chan struct{}
	closeOnce      sync.Once
}

// AddService adds a service for the server.
// The param serviceName refers to the name used for Naming Services and
// configured by config file (typically trpc_go.yaml).
// When trpc.NewServer() is called, it will traverse service configuration from config file,
// and call AddService to add a service implementation to the server's map[string]Service (serviceName as key).
func (s *Server) AddService(serviceName string, service Service) {
	if s.services == nil {
		s.services = make(map[string]Service)
	}
	s.services[serviceName] = service
}

// Service returns a service by service name.
func (s *Server) Service(serviceName string) Service {
	if s.services == nil {
		return nil
	}
	return s.services[serviceName]
}

// Register implements Service interface, registering a proto service.
// Normally, a server contains only one service, so the registration is straightforward.
// When it comes to server with multiple services, remember to use Service("servicename") to specify
// which service this proto service is registered for.
// Otherwise, this proto service will be registered for all services of the server.
func (s *Server) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	desc, ok := serviceDesc.(*ServiceDesc)
	if !ok {
		return errors.New("service desc type invalid")
	}

	for _, srv := range s.services {
		if err := srv.Register(desc, serviceImpl); err != nil {
			return err
		}
	}
	return nil
}

// Close implements Service interface, notifying all services of server shutdown.
// Would wait no more than 10s.
func (s *Server) Close(ch chan struct{}) error {
	if s.closeCh != nil {
		close(s.closeCh)
	}

	s.tryClose()

	if ch != nil {
		ch <- struct{}{}
	}
	return nil
}

func (s *Server) tryClose() {
	fn := func() {
		// execute shutdown hook functions before closing services.
		s.mux.Lock()
		for _, f := range s.onShutdownHooks {
			f()
		}
		s.mux.Unlock()

		// close all Services
		closeWaitTime := s.MaxCloseWaitTime
		if closeWaitTime < MaxCloseWaitTime {
			closeWaitTime = MaxCloseWaitTime
		}
		ctx, cancel := context.WithTimeout(context.Background(), closeWaitTime)
		defer cancel()

		var wg sync.WaitGroup
		for name, service := range s.services {
			if _, ok := s.failedServices.Load(name); ok {
				continue
			}

			wg.Add(1)
			go func(srv Service) {
				defer wg.Done()

				c := make(chan struct{}, 1)
				go srv.Close(c)

				select {
				case <-c:
				case <-ctx.Done():
				}
			}(service)
		}
		wg.Wait()
	}
	s.closeOnce.Do(fn)
}

// RegisterOnShutdown registers a hook function that would be executed when server is shutting down.
func (s *Server) RegisterOnShutdown(fn func()) {
	if fn == nil {
		return
	}
	s.mux.Lock()
	s.onShutdownHooks = append(s.onShutdownHooks, fn)
	s.mux.Unlock()
}
