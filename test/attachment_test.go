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

package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// tRPC-Go implementation:
// 需要进行协议兼容性测试
// 不支持attachment的版本记为v0，支持attachment的版本记为v1。
//
// | 主调框架协议版本  | 被调框架协议版本  | 是否兼容  |
// | -------------  | -------------  | --------------  |
// | v0 | v1 | 兼容（v0版本的包头没有attachment_size字段，v1版本被调按attachment_size为0，即没有attachment进行解析）  |
// | v1(不带attachment) | v0 | 兼容（v1版本的RPC包在attachment_size为0的时候与v0版本相同，不影响v0版本被调的解析）  |
// | v1(带attachment) | v0 | 不兼容（v0版本被调不知道attachment的存在，会将Unary包体和attachment都当成Unary包体，导致解析失败）  |
func (s *TestSuite) TestAttachment() {
	s.T().Run("client doesn't send request attachment to server, "+
		"but receives response attachment from server", func(t *testing.T) {
		attachment := []byte("attachment")
		s.startServer(&TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			msg := trpc.Message(ctx)
			a := server.GetAttachment(msg)
			bts, err := io.ReadAll(a.Request())
			if err != nil || len(bts) != 0 {
				t.Errorf("err, attachment = want: (%v, %s),  got: (%v, %s)", nil, "", err, string(bts))
			}
			a.SetResponse(bytes.NewReader(attachment))
			return &testpb.Empty{}, nil
		}})
		t.Cleanup(func() { s.closeServer(nil) })

		c := s.newTRPCClient()
		ctx, msg := codec.EnsureMessage(context.Background())
		// using var a client.Attachment or a := client.NewAttachment(nil) will panic.
		a := client.NewAttachment(bytes.NewReader([]byte("")))
		_, err := c.EmptyCall(ctx, &testpb.Empty{}, client.WithAttachment(a))
		require.Nil(s.T(), err)

		bts, err := io.ReadAll(a.Response())
		require.Same(t, msg, trpc.Message(ctx))
		require.Nil(t, err)
		require.Equal(t, attachment, bts)
	})
	s.T().Run("client send request attachment to server, "+
		"and receives response attachment from server", func(t *testing.T) {
		attachment := []byte("attachment")
		s.startServer(&TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			msg := trpc.Message(ctx)
			a := server.GetAttachment(msg)
			bts, err := io.ReadAll(a.Request())

			if err != nil || !bytes.Equal(bts, attachment) {
				t.Errorf("err, attachment = want: (%v, %s),  got: (%v, %s)",
					nil, string(attachment), err, string(bts))
			}
			a.SetResponse(bytes.NewReader(attachment))
			return &testpb.Empty{}, nil
		}})
		t.Cleanup(func() { s.closeServer(nil) })

		c := s.newTRPCClient()
		ctx, msg := codec.EnsureMessage(context.Background())
		a := client.NewAttachment(bytes.NewReader(attachment))
		_, err := c.EmptyCall(ctx, &testpb.Empty{}, client.WithAttachment(a))
		require.Nil(s.T(), err)

		got, err := io.ReadAll(a.Response())
		require.Same(t, msg, trpc.Message(ctx))
		require.Nil(t, err)
		require.Equal(t, attachment, got)
	})
	s.T().Run("server receives request attachment from client, "+
		"but doesn't send response attachment to client", func(t *testing.T) {
		attachment := []byte("attachment")
		s.startServer(&TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			msg := trpc.Message(ctx)
			a := server.GetAttachment(msg)
			bts, err := io.ReadAll(a.Request())
			if err != nil || !bytes.Equal(bts, attachment) {
				t.Errorf("err, attachment = want: (%v, %s),  got: (%v, %s)",
					nil, string(attachment), err, string(bts))
			}
			return &testpb.Empty{}, nil
		}})
		t.Cleanup(func() { s.closeServer(nil) })

		c := s.newTRPCClient()
		ctx, msg := codec.EnsureMessage(context.Background())
		a := client.NewAttachment(bytes.NewReader(attachment))
		_, err := c.EmptyCall(ctx, &testpb.Empty{}, client.WithAttachment(a))
		require.Nil(s.T(), err)

		got, err := io.ReadAll(a.Response())
		require.Same(t, msg, trpc.Message(ctx))
		require.Nil(t, err)
		require.Empty(t, got)
	})
	s.T().Run("client doesn't send request attachment to server,"+
		"and server doesn't send response attachment to client", func(t *testing.T) {
		s.startServer(&TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			msg := trpc.Message(ctx)
			a := server.GetAttachment(msg)
			bts, err := io.ReadAll(a.Request())
			if err != nil || len(bts) != 0 {
				t.Errorf("err, attachment = want: (%v, %s),  got: (%v, %s)", nil, "", err, string(bts))
			}
			// adding a.SetResponse(nil) will panic.
			return &testpb.Empty{}, nil
		}})
		t.Cleanup(func() { s.closeServer(nil) })

		c := s.newTRPCClient()
		ctx, msg := codec.EnsureMessage(context.Background())
		attachment := client.NewAttachment(nil)

		_, err := c.EmptyCall(ctx, &testpb.Empty{})
		require.Nil(s.T(), err)

		bts, err := io.ReadAll(attachment.Response())
		require.Same(t, msg, trpc.Message(ctx))
		require.Nil(t, err)
		require.Empty(t, bts)
	})
}

// 这里通过测试用例来展示其他可行方法，并讨论各种方法的优点和缺点，包括以下方法：
// 1. trans_info 字段透传
// 2. client 指定空序列化方式
// 3. server 自定义桩代码透传数据
// 4. pb3 中 byte 定义字段加上相关减少拷贝的函数 https://learn.microsoft.com/en-us/aspnet/core/grpc/performance?view=aspnetcore-7.0#binary-payloads
// 5. streaming

// 1. trans_info 字段透传。
// 框架支持在 client 和 server 之间透传字段，并在整个调用链路自动透传下去。
// 因为 trans_info 声明为 pb 中的 map<string, bytes> 类型，所以二进制文件不可避免的需要被序列化/反序列化。
//
// 请求协议头
// message RequestProtocol {
// map<string, bytes> trans_info         = 9;
// }
// 响应协议头
// message ResponseProtocol {
// map<string, bytes> trans_info         = 8;
// }
//
// 因为当前 FrameHead 中 HeaderLen 的类型为 uint16，所以使用 client.WithMetaData(key, value) 设置的 value 大小不能超过 2^16。
// FrameHead is head of the trpc frame.
//
//	type FrameHead struct {
//		FrameType       uint8  // type of the frame
//		StreamFrameType uint8  // type of the stream frame
//		TotalLen        uint32 // total length
//		HeaderLen       uint16 // header's length
//		StreamID        uint32 // stream id for streaming rpc, request id for unary rpc
//		ProtocolVersion uint8  // version of protocol
//		FrameReserved   uint8  // reserved bits for further development
//	}
func (s *TestSuite) TestTransInfo() {
	// Given a server that provide trpc service with Echo interface
	s.startServer(&TRPCService{})

	// When a client invokes Echo with MetaData
	c := s.newTRPCClient()
	head := &trpcpb.ResponseProtocol{}
	_, err := c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithMetaData("invalid-key", []byte("value")),
		client.WithRspHead(head),
	)

	// Then the client should receive same metadata that is automatic post backed by server
	require.Nil(s.T(), err)
	require.Equal(s.T(),
		[]byte("value"),
		head.TransInfo["invalid-key"],
		"metadata set by client.WithMetaData option will automatic postback",
	)

	// When another client invokes Echo with very large MetaData
	_, err = c.UnaryCall(
		trpc.BackgroundContext(),
		s.defaultSimpleRequest,
		client.WithMetaData("invalid-key", make([]byte, 65536)),
		client.WithRspHead(head),
	)

	// Then the request shouldn't send to server, because encoding FrameHead is failed
	require.Equal(s.T(), errs.RetClientEncodeFail, errs.Code(err))
	require.Contains(s.T(), err.Error(), "head len overflows uint16")
}

// 2. client 指定空序列化方式。
// 向下游发起rpc请求时直接把二进制body打包发出去，没有经过序列化，回包后，也是直接把二进制body返回，没有经过反序列化。
// 只有 client 没有经过序列化/反序列化，而 server 需要经过序列化和反序列。
func (s *TestSuite) TestClientNoopSerialization() {
	// Given a server that provide trpc service named "trpc.testing.end2end.TestTRPC" with method "UnaryCall"
	s.startServer(&TRPCService{})

	// When the DefaultClient invokes Echo to send binary data directly
	ctx, msg := codec.WithCloneMessage(trpc.BackgroundContext())
	msg.WithClientRPCName("/trpc.testing.end2end.TestTRPC/UnaryCall")
	rsp := &codec.Body{}
	err := client.DefaultClient.Invoke(ctx, &codec.Body{Data: mustMarshalPb(s.T(), s.defaultSimpleRequest)}, rsp,
		[]client.Option{
			client.WithProtocol("trpc"),
			client.WithSerializationType(codec.SerializationTypePB),
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithTarget(s.serverAddress()),
		}...,
	)

	// Then the request is handled successfully
	require.Nil(s.T(), err)
	var sr testpb.SimpleResponse
	require.Nil(s.T(), proto.Unmarshal(rsp.Data, &sr))
	require.Len(s.T(), sr.Payload.Body, int(s.defaultSimpleRequest.ResponseSize))
}

func mustMarshalPb(t *testing.T, any protoreflect.ProtoMessage) []byte {
	t.Helper()
	bts, err := proto.Marshal(any)
	if err != nil {
		t.Fatal(err)
	}
	return bts
}

// 3. server 自定义桩代码透传数据
// server 收包时直接把二进制 body 取出来交给 handle 处理函数，没有经过反序列化。
// 回包时，也是直接把二进制body打包给上游，没有经过序列化。
// 因为没有序列化与反序列化过程，也就是没有 pb 协议文件，所以需要用户自己编写服务桩代码和处理函数。
// 关键点是使用 codec.Body 来透传二进制，并自行执行 filter 拦截器
func (s *TestSuite) TestServerNoopSerialization() {
	// Given a server that provide trpc service named "trpc.testing.end2end.TestTRPC" with method "Echo"
	l := startServer(s.T(), &echoService{})

	// When the DefaultClient invokes Echo to send binary data directly
	ctx, msg := codec.WithCloneMessage(trpc.BackgroundContext())
	msg.WithClientRPCName("trpc.testing.end2end.TestTRPC/Echo")
	req := &codec.Body{Data: mustMarshalPb(s.T(), s.defaultSimpleRequest)}
	rsp := &codec.Body{}
	err := client.DefaultClient.Invoke(ctx, req, rsp,
		[]client.Option{
			client.WithProtocol("trpc"),
			client.WithSerializationType(codec.SerializationTypePB),
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithTarget(fmt.Sprintf("%s://%v", "ip", l.Addr())),
			client.WithTimeout(time.Second),
		}...,
	)

	// Then the request is handled successfully
	require.Nil(s.T(), err)
	require.Equal(s.T(), req, rsp)
}

func startServer(t *testing.T, ts trpcService) net.Listener {
	const serviceName = "trpc.testing.end2end.testTRPC"
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	svr := &server.Server{}
	svr.AddService(serviceName, server.New(
		server.WithServiceName(serviceName),
		server.WithCurrentSerializationType(codec.SerializationTypeNoop),
		server.WithListener(l),
		server.WithProtocol("trpc"),
		server.WithNetwork("tcp"),
	))
	if err := svr.Service(serviceName).Register(&server.ServiceDesc{
		ServiceName: trpcServiceName,
		HandlerType: (*trpcService)(nil),
		Methods: []server.Method{
			{
				Name: "trpc.testing.end2end.TestTRPC/Echo",
				Func: testTRPCServiceUnaryCallHandler,
			},
		}}, ts); err != nil {
		t.Fatal(err)
	}

	go func() {
		t.Log(svr.Serve())
	}()

	return l
}

type trpcService interface {
	Echo(ctx context.Context, req *codec.Body) (*codec.Body, error)
}

type echoService struct{}

func (s *echoService) Echo(_ context.Context, req *codec.Body) (*codec.Body, error) {
	return req, nil
}

func testTRPCServiceUnaryCallHandler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &codec.Body{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, req interface{}) (interface{}, error) {
		return svr.(trpcService).Echo(ctx, req.(*codec.Body))
	}
	return filters.Filter(ctx, req, handleFunc)
}

// 4. pb3 中 byte 定义字段加上相关减少拷贝的函数
// 减少拷贝的其他技术：
// Arena:
// proposal: arena: new package providing memory arenas https://github.com/golang/go/issues/51317
// How to reuse []byte field when unmarshalling? https://github.com/golang/protobuf/issues/1495
// C++ Arena Allocation Guide https://protobuf.dev/reference/cpp/arenas/
// Zero Copy:
// Opensource C++ zero-copy API:  https://github.com/protocolbuffers/protobuf/issues/1896
// https://protobuf.dev/reference/cpp/api-docs/google.protobuf.io.zero_copy_stream/
func TestPBByte(t *testing.T) {
	t.Skip("unsupported feature")
}
