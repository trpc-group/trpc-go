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

// Package server contains the server-side components, including network communication,
// name services, monitoring and statistics, link tracing, and other fundamental interfaces.
// The specific implementations are registered by third-party middleware.
package server

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// NewServer creates a new Server.
func NewServer(opts ...Option) *Server {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return &Server{
		MaxCloseWaitTime:       o.MaxCloseWaitTime,
		disableGracefulRestart: o.DisableGracefulRestart,
	}
}

// Server is a tRPC server.
// One process, one server. A server may offer one or more services.
type Server struct {
	// MaxCloseWaitTime determines the max waiting time when closing server.
	MaxCloseWaitTime time.Duration

	// services is a map that k=serviceName, v=Service.
	services map[string]Service

	// muxRestartHook guards beforeGracefulRestartHooks.
	muxRestartHook sync.Mutex
	// beforeGracefulRestartHooks are hook functions that would be executed
	// when server is starting (before gracefully restarting).
	beforeGracefulRestartHooks []func()

	// disableGracefulRestart indicates whether the server is disabled for graceful restart.
	// disableGracefulRestart is invalid for windows (because graceful restart is not supported on windows).
	disableGracefulRestart bool

	// muxShutdownHook guards onShutdownHooks.
	muxShutdownHook sync.Mutex
	// onShutdownHooks are hook functions that would be executed when server is
	// shutting down (before closing all services of the server).
	onShutdownHooks []func()

	failedServices sync.Map
	signalCh       chan os.Signal
	closeCh        chan struct{}
	closeOnce      sync.Once
}

// ServiceInfo contains unary RPC method info, streaming RPC method info for a service.
// ServiceInfo is obtained from the ServiceDesc in the pb file, and is consistent with the description in the pb file.
// We define a simple struct ServiceInfo instead of using ServiceDesc that contains too much detailed information.
type ServiceInfo struct {
	Name    string
	Methods []MethodInfo
}

// MethodInfo contains the information of an RPC including its method name and type.
type MethodInfo struct {
	// Name is the method name only, without the service name or package name.
	Name string
	// IsClientStream indicates whether the RPC is a client streaming RPC.
	IsClientStream bool
	// IsServerStream indicates whether the RPC is a server streaming RPC.
	IsServerStream bool
}

// GetServiceInfo returns a map from service names to ServiceInfo.
// key: server.service.name field in yaml.
// value: ServiceInfo registered in xxx.trpc.go
func (s *Server) GetServiceInfo() map[string]ServiceInfo {
	serviceInfo := make(map[string]ServiceInfo)
	for serviceName, srv := range s.services {
		service, ok := srv.(*service)
		if !ok {
			continue
		}
		methods := make([]MethodInfo, 0, len(service.handlers)+len(service.streamInfo))
		for methodName := range service.handlers {
			methods = append(methods, MethodInfo{
				Name: methodName,
			})
		}
		for _, info := range service.streamInfo {
			methods = append(methods, MethodInfo{
				Name:           info.FullMethod,
				IsClientStream: info.IsClientStream,
				IsServerStream: info.IsServerStream,
			})
		}
		serviceInfo[serviceName] = ServiceInfo{
			Name:    service.name,
			Methods: methods,
		}
	}
	return serviceInfo
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

	s.tryClose(nil)

	if ch != nil {
		ch <- struct{}{}
	}
	return nil
}

func (s *Server) tryClose(e error) {
	fn := func() {
		// execute shutdown hook functions before closing services.
		s.muxShutdownHook.Lock()
		for _, f := range s.onShutdownHooks {
			f()
		}
		s.muxShutdownHook.Unlock()

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
				go func() {
					if causeCloser, ok := srv.(causeCloser); ok {
						_ = causeCloser.CloseCause(e)
						close(c)
					} else {
						_ = srv.Close(c)
					}
				}()

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

// RegisterBeforeGracefulRestart registers a hook function that would be executed
// before server is gracefully restarting.
func (s *Server) RegisterBeforeGracefulRestart(fn func()) {
	if fn == nil {
		return
	}
	s.muxRestartHook.Lock()
	s.beforeGracefulRestartHooks = append(s.beforeGracefulRestartHooks, fn)
	s.muxRestartHook.Unlock()
}

// RegisterOnShutdown registers a hook function that would be executed when server is shutting down.
func (s *Server) RegisterOnShutdown(fn func()) {
	if fn == nil {
		return
	}
	s.muxShutdownHook.Lock()
	s.onShutdownHooks = append(s.onShutdownHooks, fn)
	s.muxShutdownHook.Unlock()
}

// MustService returns a service by service name, if the service doesn't exist,
// it return a NoopService. Use it when you want to skip empty services during
// service registration. For example, when you're unsure whether the service
// exists or not, you need to check:
//
//	if service := s.Service("my_service"); service != nil {
//	    stub.RegisterService(service, impl)
//	}
//
// Using MustService, you don't need to check if the service is nil:
//
// stub.RegisterService(s.MustService("my_service"), impl)
func (s *Server) MustService(name string) Service {
	if svc := s.Service(name); svc != nil {
		return svc
	}
	return &NoopService{Name: name}
}

// NoopService is an empty implementation of Service.
type NoopService struct {
	Name string
}

// Register simply skips.
func (s *NoopService) Register(desc, impl interface{}) error {
	log.Infof("noop service %s registration is auto skipped", s.Name)
	return nil
}

// Serve does nothing.
func (s *NoopService) Serve() error {
	log.Infof("noop service %s serving does nothing", s.Name)
	return nil
}

// Close directly sends a value to the parameter chan.
func (s *NoopService) Close(ch chan struct{}) error {
	ch <- struct{}{}
	log.Infof("noop service %s closing just send a new value to parameter chan", s.Name)
	return nil
}

type causeCloser interface {
	CloseCause(error) error
}
