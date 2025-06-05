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
// //
// //
// // Tencent is pleased to support the open source community by making tRPC available.
// //
// // Copyright (C) 2023 THL A29 Limited, a Tencent company.
// // All rights reserved.
// //
// // If you have downloaded a copy of the tRPC source code from Tencent,
// // please note that tRPC source code is licensed under the  Apache 2.0 License,
// // A copy of the Apache 2.0 License is included in this file.
// //
// //

package reflection_test

// import (
// 	"context"
// 	"fmt"
// 	"net"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/require"
// 	"github.com/stretchr/testify/suite"
// 	"google.golang.org/protobuf/proto"
// 	"google.golang.org/protobuf/reflect/protodesc"
// 	"google.golang.org/protobuf/reflect/protoregistry"
// 	"google.golang.org/protobuf/types/descriptorpb"
// 	reflectionpb "trpc.group/trpc/trpc-protocol/pb/trpc/trpc/reflection"

// 	"trpc.group/trpc-go/trpc-go"
// 	"trpc.group/trpc-go/trpc-go/client"
// 	"trpc.group/trpc-go/trpc-go/errs"
// 	"trpc.group/trpc-go/trpc-go/reflection"
// 	"trpc.group/trpc-go/trpc-go/server"
// 	testpb "trpc.group/trpc-go/trpc-go/testdata/reflection"
// )

// func TestRunSuite(t *testing.T) {
// 	suite.Run(t, new(TestSuite))
// }

// type TestSuite struct {
// 	suite.Suite
// 	server *server.Server
// 	client reflectionpb.ServerReflectionClientProxy

// 	fdSearch     []byte
// 	fdReflection []byte
// 	fdSort       []byte
// }

// type service struct{}

// func (s *service) Search(ctx context.Context, in *testpb.SearchRequest) (*testpb.SearchResponse, error) {
// 	return &testpb.SearchResponse{}, nil
// }

// func (s *service) StreamingSearch(stream testpb.Search_StreamingSearchServer) error {
// 	return nil
// }

// func (ts *TestSuite) SetupSuite() {
// 	l1, err := net.Listen("tcp", "127.0.0.1:0")
// 	require.NoError(ts.T(), err)
// 	svr := &server.Server{}
// 	svr.AddService("trpc.test.reflection.Search", server.New(
// 		server.WithServiceName("trpc.test.reflection.Search"),
// 		server.WithProtocol("trpc"),
// 		server.WithListener(l1),
// 	))
// 	testpb.RegisterSearchService(svr.Service("trpc.test.reflection.Search"), new(service))

// 	l2, err := net.Listen("tcp", "127.0.0.1:0")
// 	require.NoError(ts.T(), err)
// 	svr.AddService("trpc.reflection.v1.ServerReflection", server.New(
// 		server.WithServiceName("trpc.reflection.v1.ServerReflection"),
// 		server.WithProtocol("trpc"),
// 		server.WithListener(l2),
// 	))
// 	reflection.Register(svr.Service("trpc.reflection.v1.ServerReflection"), svr)

// 	ts.server = svr
// 	go func() {
// 		if err := ts.server.Serve(); err != nil {
// 			ts.T().Logf("server serving: %v", err)
// 		}
// 	}()

// 	ts.client = reflectionpb.NewServerReflectionClientProxy(
// 		client.WithTimeout(time.Second),
// 		client.WithTarget(fmt.Sprintf("ip://%s", l2.Addr())),
// 	)

// 	// from testdata/reflection/search.proto
// 	_, ts.fdSearch = loadFileDesc(ts.T(), "search.proto")
// 	// from testdata/reflection/sort.proto
// 	_, ts.fdSort = loadFileDesc(ts.T(), "sort.proto")
// 	// from "git.woa.com/trpc/trpc-protocol/reflection/reflection.proto"
// 	_, ts.fdReflection = loadFileDesc(ts.T(), "reflection.proto")
// }

// func (ts *TestSuite) TearDownSuite() {
// 	if err := ts.server.Close(nil); err != nil {
// 		ts.T().Logf("server closing: %v", err)
// 	}
// }

// func (ts *TestSuite) TestFileByFilenameTransitiveClosure() {
// 	r, err := ts.client.ServiceReflectionInfo(trpc.BackgroundContext(), &reflectionpb.ServerReflectionRequest{
// 		MessageRequest: &reflectionpb.ServerReflectionRequest_FileByFilename{FileByFilename: "sort.proto"},
// 	})
// 	ts.Nil(err)
// 	ts.IsType(r.MessageResponse, &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{})
// 	ts.Len(r.GetFileDescriptorResponse().GetFileDescriptorProto(), 2)
// 	ts.EqualValues(ts.fdSort, r.GetFileDescriptorResponse().FileDescriptorProto[0])
// 	ts.EqualValues(ts.fdSearch, r.GetFileDescriptorResponse().FileDescriptorProto[1])
// }

// func (ts *TestSuite) TestFileByFilename() {
// 	for _, test := range []struct {
// 		filename string
// 		want     []byte
// 	}{
// 		{"search.proto", ts.fdSearch},
// 		{"reflection.proto", ts.fdReflection},
// 	} {
// 		r, err := ts.client.ServiceReflectionInfo(trpc.BackgroundContext(), &reflectionpb.ServerReflectionRequest{
// 			MessageRequest: &reflectionpb.ServerReflectionRequest_FileByFilename{FileByFilename: test.filename},
// 		})
// 		ts.Nil(err)
// 		ts.IsType(r.MessageResponse, &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{})
// 		ts.EqualValues(test.want, r.GetFileDescriptorResponse().FileDescriptorProto[0])
// 	}
// }

// func (ts *TestSuite) TestListServices() {
// 	r, err := ts.client.ServiceReflectionInfo(trpc.BackgroundContext(), &reflectionpb.ServerReflectionRequest{
// 		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{},
// 	})
// 	ts.Require().NoError(err)
// 	ts.IsType(r.MessageResponse, &reflectionpb.ServerReflectionResponse_ListServicesResponse{})

// 	want := map[string]string{
// 		"trpc.reflection.v1.ServerReflection": "trpc.reflection.v1.ServerReflection",
// 		"trpc.test.reflection.Search":         "trpc.testdata.reflection.Search",
// 	}
// 	got := make(map[string]string)
// 	for _, s := range r.GetListServicesResponse().GetService() {
// 		fmt.Println(s)
// 		got[s.RoutingServiceName] = s.InterfaceServiceName
// 	}
// 	ts.EqualValues(want, got)
// }

// func (ts *TestSuite) TestFileContainingSymbol() {
// 	for _, test := range []struct {
// 		symbol string
// 		want   []byte
// 	}{
// 		{"trpc.testdata.reflection.Search", ts.fdSearch},
// 		{"trpc.testdata.reflection.SearchRequest", ts.fdSearch},
// 		{"trpc.testdata.reflection.SearchResponse", ts.fdSearch},

// 		{"trpc.reflection.v1.ServerReflection", ts.fdReflection},
// 		{"trpc.reflection.v1.ServerReflection.ServiceReflectionInfo", ts.fdReflection},
// 		{"trpc.reflection.v1.ServerReflectionRequest", ts.fdReflection},
// 		{"trpc.reflection.v1.ServerReflectionResponse", ts.fdReflection},
// 		{"trpc.reflection.v1.ListServiceResponse", ts.fdReflection},
// 		{"trpc.reflection.v1.ErrorResponse", ts.fdReflection},
// 		{"trpc.reflection.v1.FileDescriptorResponse", ts.fdReflection},
// 	} {
// 		r, err := ts.client.ServiceReflectionInfo(trpc.BackgroundContext(), &reflectionpb.ServerReflectionRequest{
// 			MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: test.symbol},
// 		})
// 		ts.Nil(err)
// 		ts.IsType(r.MessageResponse, &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{})
// 		ts.EqualValues(test.want, r.GetFileDescriptorResponse().FileDescriptorProto[0])
// 	}
// }

// func (ts *TestSuite) TestFileContainingSymbolError() {
// 	for _, test := range []struct {
// 		symbol string
// 		want   []byte
// 	}{
// 		{"trpc.testdata.reflection.SearchX", ts.fdSearch},
// 		{"trpc.testdata.reflection.SearchRequestX", ts.fdSearch},
// 		{"trpc.testdataX.reflection.SearchResponse", ts.fdSearch},
// 	} {
// 		r, err := ts.client.ServiceReflectionInfo(trpc.BackgroundContext(), &reflectionpb.ServerReflectionRequest{
// 			MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: test.symbol},
// 		})
// 		ts.Equal(errs.RetNotFound, errs.Code(err))
// 		ts.Nil(r)
// 	}
// }

// func loadFileDesc(t *testing.T, filename string) (*descriptorpb.FileDescriptorProto, []byte) {
// 	t.Helper()
// 	fd, err := protoregistry.GlobalFiles.FindFileByPath(filename)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	fdProto := protodesc.ToFileDescriptorProto(fd)
// 	b, err := proto.Marshal(fdProto)
// 	if err != nil {
// 		t.Fatalf("failed to marshal fd: %v", err)
// 	}
// 	return fdProto, b
// }
