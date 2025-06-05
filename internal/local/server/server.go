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

// Package server provides implementations of local servers.
package server

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	icodec "trpc.group/trpc-go/trpc-go/internal/codec"
	ireflect "trpc.group/trpc-go/trpc-go/internal/reflect"
	"trpc.group/trpc-go/trpc-go/internal/report"
)

var defaultServer = NewLocalServer()

// Server is the local server.
type Server struct {
	mu              sync.Mutex
	protocolService map[string]services
}

type services map[string]*Service

// NewLocalServer creates a new local server.
func NewLocalServer() *Server {
	return &Server{
		protocolService: make(map[string]services),
	}
}

// Register registers a service with certain rpc name and handler to the server.
func (s *Server) Register(
	serviceName, rpcName string,
	handler Handler,
	opts Options,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.protocolService[opts.Protocol]
	if !ok {
		ps = make(services)
		s.protocolService[opts.Protocol] = ps
	}
	service, ok := ps[serviceName]
	if !ok {
		service = &Service{
			handlers: make(map[string]Handler),
			opts:     opts,
		}
		ps[serviceName] = service
	}
	service.handlers[rpcName] = handler
}

// GetService gets the service from the default server.
func (s *Server) GetService(protocol, serviceName string) (*Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.protocolService[protocol]
	if !ok {
		return nil, fmt.Errorf("service with protocol %s not found", protocol)
	}
	service, ok := ps[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	return service, nil
}

// Register registers the handler to the default local server.
func Register(
	serviceName, rpcName string,
	handler Handler,
	opts Options,
) {
	defaultServer.Register(serviceName, rpcName, handler, opts)
}

// GetService gets the service from the default server.
func GetService(protocol, serviceName string) (*Service, error) {
	return defaultServer.GetService(protocol, serviceName)
}

// FilterFunc is the alias of the filter function used by the stub code.
type FilterFunc = func(reqBody interface{}) (filter.ServerChain, error)

// Handler is the server-side handle function for the stub code.
type Handler func(ctx context.Context, f FilterFunc) (rspBody interface{}, err error)

// Service is a single service.
type Service struct {
	handlers map[string]Handler
	opts     Options
}

// PartialDecode decodes the partial reqBuf and set necessary information into context message.
// This partial decode is needed to convert the client context message to server context message
// and avoid the marshalling/unmarshalling of the request body.
// This step is separated from the handle stage to avoid passing both reqBuf and reqBody to the
// handle function.
func (s *Service) PartialDecode(msg codec.Msg, reqBuf []byte) error {
	if c := s.opts.ServerCodecGetter(); c != nil {
		_, err := c.Decode(msg, reqBuf)
		return err
	}
	return errors.New("server codec is nil for partial decode")
}

// Handle handles a single RPC request.
func (s *Service) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	msg := codec.Message(ctx)
	if fh, ok := msg.FrameHead().(icodec.FrameHead); ok && fh.IsStream() {
		return nil, errors.New("stream RPC is not supported")
	}
	handler, ok := s.handlers[msg.ServerRPCName()]
	if !ok {
		handler, ok = s.handlers["*"]
		if !ok {
			report.ServiceHandleRPCNameInvalid.Incr()
			return nil, errs.NewFrameError(errs.RetServerNoFunc, msg.ServerRPCName()+" not found")
		}
	}
	newFilterFunc := func(reqBody interface{}) (filter.ServerChain, error) {
		if err := ireflect.Assign(reqBody, req); err != nil {
			return nil, err
		}
		return s.opts.Filters, nil
	}
	rspBody, err := handler(ctx, newFilterFunc)
	if err != nil {
		return nil, err
	}
	if msg.CallType() == codec.SendOnly {
		return nil, errs.ErrServerNoResponse
	}
	return rspBody, nil
}
