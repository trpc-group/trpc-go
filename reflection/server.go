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

// Package reflection implements server reflection service.
//
// The service implemented is defined in:
// https://git.woa.com/trpc/trpc-protocol/trpc/reflection.proto.
package reflection

import (
	"context"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	reflectionpb "trpc.group/trpc/trpc-protocol/pb/go/trpc/reflection"

	"trpc.group/trpc-go/trpc-go/errs"
	ireflection "trpc.group/trpc-go/trpc-go/internal/reflection"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

func init() {
	ireflection.Register = register
}

// Register Registers the reflection service and to the server.Service.
// reflection service get ServiceInfo by calling *server.Server.GetServiceInfo.
func Register(service server.Service, serviceInfo ServiceInfoProvider) {
	log.Warnf("The server reflection feature is being enabled. " +
		"Please note that this feature is typically only available in the testing environment, " +
		"and using it in the production environment may cause security issues.")
	reflectionpb.RegisterServerReflectionService(service, newServer(serverOptions{ServiceInfo: serviceInfo}))
}

func register(service server.Service, svr *server.Server) {
	Register(service, svr)
}

// newServer returns a reflection server implementation using the given options.
// This can be used to customize behavior of the reflection service.
func newServer(opts serverOptions) *service {
	if opts.ServiceInfo == nil {
		opts.ServiceInfo = emptyServer{}
	}
	return &service{
		serviceInfo:  opts.ServiceInfo,
		descResolver: protoregistry.GlobalFiles,
	}
}

// service is reflection service
type service struct {
	descResolver protodesc.Resolver
	serviceInfo  ServiceInfoProvider
}

// serverOptions represents the options used to construct a reflection server.
type serverOptions struct {
	ServiceInfo ServiceInfoProvider
}

// ServiceInfoProvider is an interface used to retrieve metadata about the
// services to expose.
//
// The reflection service is only interested in the service names, but the
// signature is this way so that *trpc.Server implements it. So it is okay
// for a custom implementation to return zero values for the
// trpc.ServiceDesc values in the map.
type ServiceInfoProvider interface {
	// GetServiceInfo returns service info
	// key: the service name of the routing
	GetServiceInfo() map[string]server.ServiceInfo
}

type emptyServer struct{}

func (s emptyServer) GetServiceInfo() map[string]server.ServiceInfo {
	return map[string]server.ServiceInfo{}
}

// ServiceReflectionInfo returns Reflection Info
func (s *service) ServiceReflectionInfo(
	_ context.Context, req *reflectionpb.ServerReflectionRequest) (*reflectionpb.ServerReflectionResponse, error) {
	rsp := &reflectionpb.ServerReflectionResponse{
		ValidHost:       req.Host,
		OriginalRequest: req,
	}
	switch req := req.MessageRequest.(type) {
	case *reflectionpb.ServerReflectionRequest_ListServices:
		rsp.MessageResponse = &reflectionpb.ServerReflectionResponse_ListServicesResponse{
			ListServicesResponse: &reflectionpb.ListServiceResponse{
				Service: s.listServices(),
			},
		}
	case *reflectionpb.ServerReflectionRequest_FileContainingSymbol:
		b, err := s.fileDescEncodingContainingSymbol(req.FileContainingSymbol)

		if err != nil {
			rsp.MessageResponse = &reflectionpb.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &reflectionpb.ErrorResponse{
					ErrorCode:    int32(errs.RetNotFound),
					ErrorMessage: err.Error(),
				},
			}
			return rsp, errs.NewFrameError(errs.RetNotFound, err.Error())
		}
		rsp.MessageResponse = &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{FileDescriptorProto: b},
		}

	case *reflectionpb.ServerReflectionRequest_FileByFilename:
		var b [][]byte
		fd, err := s.descResolver.FindFileByPath(req.FileByFilename)
		if err == nil {
			b, err = s.fileDescWithDependencies(fd)
		}
		if err != nil {
			rsp.MessageResponse = &reflectionpb.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &reflectionpb.ErrorResponse{
					ErrorCode:    int32(errs.RetNotFound),
					ErrorMessage: err.Error(),
				},
			}
			return rsp, errs.NewFrameError(errs.RetNotFound, err.Error())
		}
		rsp.MessageResponse = &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{FileDescriptorProto: b}}
	default:
		return nil, errs.Newf(errs.RetInvalidArgument, "invalid MessageRequest: %v", req)
	}
	return rsp, nil
}

// listServices returns the names of services this server exposes.
func (s *service) listServices() []*reflectionpb.ServiceResponse {
	serviceInfo := s.serviceInfo.GetServiceInfo()
	resp := make([]*reflectionpb.ServiceResponse, 0, len(serviceInfo))
	for routingServiceName, svc := range serviceInfo {
		log.Debug(svc.Name)
		resp = append(resp, &reflectionpb.ServiceResponse{
			RoutingServiceName:   routingServiceName,
			InterfaceServiceName: svc.Name,
		})
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].RoutingServiceName < resp[j].RoutingServiceName
	})
	return resp
}

// fileDescWithDependencies returns a slice of serialized fileDescriptors in
// wire format ([]byte). The fileDescriptors will include fd and all the
// transitive dependencies of fd with names not in sentFileDescriptors.
func (s *service) fileDescWithDependencies(fd protoreflect.FileDescriptor) ([][]byte, error) {
	var r [][]byte
	sentFileDescriptors := make(map[string]bool)
	queue := []protoreflect.FileDescriptor{fd}
	for len(queue) > 0 {
		currentFD := queue[0]
		queue = queue[1:]
		if sent := sentFileDescriptors[currentFD.Path()]; len(r) == 0 || !sent {
			sentFileDescriptors[currentFD.Path()] = true
			fdProto := protodesc.ToFileDescriptorProto(currentFD)
			currentFDEncoded, err := proto.Marshal(fdProto)
			if err != nil {
				return nil, err
			}
			r = append(r, currentFDEncoded)
		}
		for i := 0; i < currentFD.Imports().Len(); i++ {
			queue = append(queue, currentFD.Imports().Get(i))
		}
	}
	return r, nil
}

// fileDescEncodingContainingSymbol finds the file descriptor containing the
// given symbol, finds all of its previously unsent transitive dependencies,
// does marshalling on them, and returns the marshalled result. The given symbol
// can be a type, a service or a method.
func (s *service) fileDescEncodingContainingSymbol(name string) (
	[][]byte, error) {
	d, err := s.descResolver.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	return s.fileDescWithDependencies(d.ParentFile())
}
