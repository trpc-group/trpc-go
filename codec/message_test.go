// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package codec_test

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestPutBackMessage(t *testing.T) {
	ctx := context.Background()
	_, msg := codec.WithCloneMessage(ctx)
	type foo struct {
		I int
	}

	msg.WithRemoteAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.2")})
	msg.WithLocalAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	msg.WithNamespace("2")
	msg.WithEnvName("3")
	msg.WithSetName("4")
	msg.WithEnvTransfer("5")
	msg.WithRequestTimeout(time.Second)
	msg.WithSerializationType(1)
	msg.WithCompressType(2)
	msg.WithServerRPCName("6")
	msg.WithClientRPCName("7")
	msg.WithCallerServiceName("8")
	msg.WithCalleeServiceName("9")
	msg.WithCallerApp("10")
	msg.WithCallerServer("11")
	msg.WithCallerService("12")
	msg.WithCallerMethod("13")
	msg.WithCalleeApp("14")
	msg.WithCalleeServer("15")
	msg.WithCalleeService("16")
	msg.WithCalleeMethod("17")
	msg.WithCalleeContainerName("18")
	msg.WithServerMetaData(codec.MetaData{"a": []byte("1")})
	msg.WithFrameHead(foo{I: 1})
	msg.WithServerReqHead(foo{I: 2})
	msg.WithServerRspHead(foo{I: 3})
	msg.WithDyeing(true)
	msg.WithDyeingKey("19")
	msg.WithServerRspErr(errors.New("err1"))
	msg.WithClientMetaData(codec.MetaData{"b": []byte("2")})
	msg.WithClientReqHead(foo{I: 4})
	msg.WithClientRspErr(errors.New("err2"))
	msg.WithClientRspHead(foo{I: 5})
	msg.WithLogger(foo{I: 6})
	msg.WithRequestID(3)
	msg.WithStreamID(4)
	msg.WithStreamFrame(foo{I: 6})
	msg.WithCalleeSetName("20")
	msg.WithCommonMeta(codec.CommonMeta{21: []byte("hello")})
	msg.WithCallType(codec.SendOnly)

	codec.PutBackMessage(msg)

	ctx2 := context.Background()
	_, msg2 := codec.WithNewMessage(ctx2)

	assert.Nil(t, msg2.FrameHead())
	assert.Equal(t, time.Duration(0), msg2.RequestTimeout())
	assert.Equal(t, 0, msg2.SerializationType())
	assert.Equal(t, 0, msg2.CompressType())
	assert.Equal(t, false, msg2.Dyeing())
	assert.Equal(t, "", msg2.DyeingKey())
	assert.Equal(t, "", msg2.ServerRPCName())
	assert.Equal(t, "", msg2.ClientRPCName())
	assert.Nil(t, msg2.ServerMetaData())
	assert.Nil(t, msg2.ClientMetaData())
	assert.Equal(t, "", msg2.CallerServiceName())
	assert.Equal(t, "", msg2.CalleeServiceName())
	assert.Equal(t, "", msg2.CalleeContainerName())
	assert.Nil(t, msg2.ServerRspErr())
	assert.Nil(t, msg2.ClientRspErr())
	assert.Nil(t, msg2.ServerReqHead())
	assert.Nil(t, msg2.ServerRspHead())
	assert.Nil(t, msg2.ClientReqHead())
	assert.Nil(t, msg2.ClientRspHead())
	assert.Nil(t, msg2.LocalAddr())
	assert.Nil(t, msg2.RemoteAddr())
	assert.Nil(t, msg2.Logger())
	assert.Equal(t, "", msg2.CallerApp())
	assert.Equal(t, "", msg2.CallerServer())
	assert.Equal(t, "", msg2.CallerService())
	assert.Equal(t, "", msg2.CallerMethod())
	assert.Equal(t, "", msg2.CalleeApp())
	assert.Equal(t, "", msg2.CalleeServer())
	assert.Equal(t, "", msg2.CalleeService())
	assert.Equal(t, "", msg2.CalleeMethod())
	assert.Equal(t, "", msg2.Namespace())
	assert.Equal(t, "", msg2.SetName())
	assert.Equal(t, "", msg2.EnvName())
	assert.Equal(t, "", msg2.EnvTransfer())
	assert.Equal(t, uint32(0), msg2.RequestID())
	assert.Nil(t, msg2.StreamFrame())
	assert.Equal(t, uint32(0), msg2.StreamID())
	assert.Equal(t, "", msg2.CalleeSetName())
	assert.Nil(t, msg2.CommonMeta())
	assert.Equal(t, codec.SendAndRecv, msg2.CallType())

}

func TestRegisterMessage(t *testing.T) {
	ctx := context.Background()
	m0 := codec.Message(ctx)
	assert.NotNil(t, m0)
	assert.Equal(t, ctx, m0.Context())
	ctx, m0 = codec.WithCloneMessage(ctx)
	assert.NotNil(t, m0)
	assert.Equal(t, ctx, m0.Context())

	meta := codec.MetaData{}
	reqhead := &trpcpb.RequestProtocol{}
	rsphead := &trpcpb.ResponseProtocol{}

	meta["key"] = []byte("value")
	clone := meta.Clone()
	assert.Equal(t, []byte("value"), clone["key"])

	ctx, msg := codec.WithNewMessage(ctx)
	assert.NotNil(t, msg)
	assert.NotNil(t, ctx)
	assert.Equal(t, ctx, msg.Context())

	msg.WithRequestTimeout(time.Second)
	assert.Equal(t, time.Second, msg.RequestTimeout())
	msg.WithSerializationType(codec.SerializationTypePB)
	assert.Equal(t, codec.SerializationTypePB, msg.SerializationType())
	msg.WithServerRPCName("/package.service/method")
	msg.WithServerRPCName("/package.service/method") // dup set
	assert.Equal(t, "/package.service/method", msg.ServerRPCName())
	msg.WithServerMetaData(meta)
	assert.Equal(t, meta, msg.ServerMetaData())
	msg.WithServerMetaData(nil)
	assert.NotNil(t, msg.ServerMetaData())
	msg.WithServerReqHead(reqhead)
	assert.Equal(t, reqhead, msg.ServerReqHead().(*trpcpb.RequestProtocol))
	msg.WithServerRspHead(rsphead)
	assert.Equal(t, rsphead, msg.ServerRspHead().(*trpcpb.ResponseProtocol))
	msg.WithDyeing(true)
	assert.Equal(t, true, msg.Dyeing())
	msg.WithDyeingKey("hellotrpc")
	assert.Equal(t, "hellotrpc", msg.DyeingKey())

	var addr net.Addr
	msg.WithRemoteAddr(addr)
	assert.Equal(t, addr, msg.RemoteAddr())

	msg.WithLocalAddr(addr)
	assert.Equal(t, addr, msg.LocalAddr())

	h := trpc.FrameHead{}
	msg.WithFrameHead(h)
	assert.Equal(t, h, msg.FrameHead())

	msg.WithCompressType(1)
	assert.Equal(t, 1, msg.CompressType())

	msg.WithCallerApp("callerApp")
	assert.Equal(t, "callerApp", msg.CallerApp())
	msg.WithCallerServer("callerServer")
	assert.Equal(t, "callerServer", msg.CallerServer())
	msg.WithCallerService("callerService")
	assert.Equal(t, "callerService", msg.CallerService())
	msg.WithCallerMethod("callerMethod")
	assert.Equal(t, "callerMethod", msg.CallerMethod())
	msg.WithCalleeApp("calleeApp")
	assert.Equal(t, "calleeApp", msg.CalleeApp())
	msg.WithCalleeServer("calleeServer")
	assert.Equal(t, "calleeServer", msg.CalleeServer())
	msg.WithCalleeService("calleeService")
	assert.Equal(t, "calleeService", msg.CalleeService())
	msg.WithCalleeMethod("calleeMethod")
	assert.Equal(t, "calleeMethod", msg.CalleeMethod())
	msg.WithSetName("setName")
	assert.Equal(t, "setName", msg.SetName())
	msg.WithCalleeSetName("calleeSetName")
	assert.Equal(t, "calleeSetName", msg.CalleeSetName())
	msg.WithEnvName("test")
	assert.Equal(t, "test", msg.EnvName())
	msg.WithNamespace("Production")
	assert.Equal(t, "Production", msg.Namespace())
	msg.WithEnvTransfer("test-test")
	assert.Equal(t, "test-test", msg.EnvTransfer())
	msg.WithCalleeContainerName("container")
	assert.Equal(t, "container", msg.CalleeContainerName())

	msg.WithLogger(log.DefaultLogger)
	assert.NotNil(t, msg.Logger())

	msg.WithCallType(codec.SendOnly)
	assert.Equal(t, msg.CallType(), codec.SendOnly)
}

func TestMoreRegisterMessage(t *testing.T) {
	ctx := context.Background()
	meta := codec.MetaData{}
	commonMeta := codec.CommonMeta{32: []byte("aaa")}
	ctx, msg := codec.WithNewMessage(ctx)
	reqhead := &trpcpb.RequestProtocol{}
	rsphead := &trpcpb.ResponseProtocol{}
	// client codec marshal
	msg.WithClientRPCName("/package.service/method")
	msg.WithClientRPCName("/package.service/method") // dup set
	assert.Equal(t, "/package.service/method", msg.ClientRPCName())
	msg.WithClientMetaData(meta)
	assert.Equal(t, meta, msg.ClientMetaData())
	msg.WithClientMetaData(nil)
	assert.NotNil(t, msg.ClientMetaData())
	msg.WithCommonMeta(commonMeta)
	assert.Equal(t, commonMeta, msg.CommonMeta())
	msg.WithCallerServiceName("package.service")
	msg.WithCallerServiceName("package.service") // dup set
	assert.Equal(t, "package.service", msg.CallerServiceName())
	msg.WithCalleeServiceName("package.service")
	msg.WithCalleeServiceName("package.service") // dup set
	assert.Equal(t, "package.service", msg.CalleeServiceName())
	msg.WithClientReqHead(reqhead)
	assert.Equal(t, reqhead, msg.ClientReqHead().(*trpcpb.RequestProtocol))
	msg.WithClientRspHead(rsphead)
	assert.Equal(t, rsphead, msg.ClientRspHead().(*trpcpb.ResponseProtocol))
	msg.WithCompressType(1)
	assert.Equal(t, msg.CompressType(), 1)

	// client codec unmarshal
	msg.WithClientRspErr(errs.ErrServerNoResponse)
	assert.Equal(t, errs.ErrServerNoResponse, msg.ClientRspErr())

	// trpc inner logic
	assert.Nil(t, msg.ServerRspErr())
	msg.WithServerRspErr(errs.ErrServerNoResponse)
	assert.Equal(t, errs.ErrServerNoResponse, msg.ServerRspErr())
	msg.WithServerRspErr(errors.New("no trpc errs"))
	assert.EqualValues(t, int32(999), msg.ServerRspErr().Code)

	m1 := codec.Message(ctx)
	assert.Equal(t, msg, m1)

	ctx, m2 := codec.WithCloneMessage(ctx)
	assert.Equal(t, m2.ServerReqHead(), m1.ServerReqHead())
	assert.Equal(t, m2.ServerRspHead(), m1.ServerRspHead())
	assert.Equal(t, m2.CallerServiceName(), m1.CallerServiceName())
	assert.Equal(t, m2.RequestTimeout(), m1.RequestTimeout())
	assert.Equal(t, m2.ServerRPCName(), m1.ServerRPCName())
	assert.Equal(t, m2.SerializationType(), m1.SerializationType())
	assert.Equal(t, true, reflect.DeepEqual(m2.ServerMetaData(), m1.ServerMetaData()))
	assert.Equal(t, m2.Dyeing(), m1.Dyeing())
	assert.Equal(t, m2.DyeingKey(), m1.DyeingKey())
	assert.Equal(t, m2.CommonMeta(), m1.CommonMeta())
	assert.NotEqual(t, m2.CompressType(), m1.CompressType())

	codec.PutBackMessage(msg)
	ctx, m3 := codec.WithNewMessage(ctx)
	assert.Equal(t, m3, msg)
	assert.Equal(t, m3.CalleeApp(), "")
	_, m4 := codec.WithNewMessage(ctx)
	assert.NotEqual(t, m4, m1)

	var fakemsg codec.Msg = nil
	codec.PutBackMessage(fakemsg)
}

// TestWithCallerServiceName WithCallerServiceName 单测
func TestWithCallerServiceName(t *testing.T) {
	ctx := trpc.BackgroundContext()
	msg := codec.Message(ctx)

	msg.WithCallerServiceName("trpc")
	assert.Equal(t, "trpc", msg.CallerApp())
	assert.Equal(t, "", msg.CallerServer())
	assert.Equal(t, "", msg.CallerService())

	msg.WithCallerServiceName("app.server")
	assert.Equal(t, "app", msg.CallerApp())
	assert.Equal(t, "server", msg.CallerServer())
	assert.Equal(t, "", msg.CallerService())

	msg.WithCallerServiceName("app.server.service")
	assert.Equal(t, "app", msg.CallerApp())
	assert.Equal(t, "server", msg.CallerServer())
	assert.Equal(t, "service", msg.CallerService())

	msg.WithCallerServiceName("trpc.app.server.service")
	assert.Equal(t, "app", msg.CallerApp())
	assert.Equal(t, "server", msg.CallerServer())
	assert.Equal(t, "service", msg.CallerService())

	msg.WithCallerServiceName("trpc.app.server.service.new")
	assert.Equal(t, "app", msg.CallerApp())
	assert.Equal(t, "server", msg.CallerServer())
	assert.Equal(t, "service.new", msg.CallerService())

	msg.WithCallerServiceName("*")
	assert.Equal(t, "*", msg.CallerServiceName())
	assert.Equal(t, "app", msg.CallerApp())
	assert.Equal(t, "server", msg.CallerServer())

	msg.WithCalleeServiceName("trpc")
	assert.Equal(t, "trpc", msg.CalleeApp())
	assert.Equal(t, "", msg.CalleeServer())
	assert.Equal(t, "", msg.CalleeService())

	msg.WithCalleeServiceName("app.server.service")
	assert.Equal(t, "app", msg.CalleeApp())
	assert.Equal(t, "server", msg.CalleeServer())
	assert.Equal(t, "service", msg.CalleeService())

	msg.WithCalleeServiceName("trpc.app.server.service")
	assert.Equal(t, "app", msg.CalleeApp())
	assert.Equal(t, "server", msg.CalleeServer())
	assert.Equal(t, "service", msg.CalleeService())

	msg.WithCalleeServiceName("trpc.app.server.service.new")
	assert.Equal(t, "app", msg.CalleeApp())
	assert.Equal(t, "server", msg.CalleeServer())
	assert.Equal(t, "service.new", msg.CalleeService())

	msg.WithCalleeServiceName("*")
	assert.Equal(t, "*", msg.CalleeServiceName())
	assert.Equal(t, "app", msg.CalleeApp())
	assert.Equal(t, "server", msg.CalleeServer())

}

func TestMsg_CopyMsg_1_CIFunctionStatementsMustLessThan80Lines(t *testing.T) {
	ctx := context.Background()
	msg := codec.Message(ctx)

	msg.WithRemoteAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.2")})
	msg.WithLocalAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	msg.WithNamespace("2")
	msg.WithEnvName("3")
	msg.WithSetName("4")
	msg.WithEnvTransfer("5")
	msg.WithRequestTimeout(time.Second)
	msg.WithSerializationType(1)
	msg.WithCompressType(2)
	msg.WithServerRPCName("6")
	msg.WithClientRPCName("7")
	msg.WithCallerServiceName("8")
	msg.WithCalleeServiceName("9")
	msg.WithCallerApp("10")
	msg.WithCallerServer("11")
	msg.WithCallerService("12")
	msg.WithCallerMethod("13")

	_, newMsg := codec.WithNewMessage(ctx)
	codec.CopyMsg(newMsg, msg)

	require.False(t, reflect.DeepEqual(msg.Context(), newMsg.Context()))
	require.True(t, reflect.DeepEqual(msg.RemoteAddr(), newMsg.RemoteAddr()))
	require.True(t, reflect.DeepEqual(msg.LocalAddr(), newMsg.LocalAddr()))
	require.True(t, reflect.DeepEqual(msg.Namespace(), newMsg.Namespace()))
	require.True(t, reflect.DeepEqual(msg.EnvName(), newMsg.EnvName()))
	require.True(t, reflect.DeepEqual(msg.SetName(), newMsg.SetName()))
	require.True(t, reflect.DeepEqual(msg.EnvTransfer(), newMsg.EnvTransfer()))
	require.True(t, reflect.DeepEqual(msg.RequestTimeout(), newMsg.RequestTimeout()))
	require.True(t, reflect.DeepEqual(msg.SerializationType(), newMsg.SerializationType()))
	require.True(t, reflect.DeepEqual(msg.CompressType(), newMsg.CompressType()))
	require.True(t, reflect.DeepEqual(msg.ServerRPCName(), newMsg.ServerRPCName()))
	require.True(t, reflect.DeepEqual(msg.ClientRPCName(), newMsg.ClientRPCName()))
	require.True(t, reflect.DeepEqual(msg.CallerServiceName(), newMsg.CallerServiceName()))
	require.True(t, reflect.DeepEqual(msg.CalleeServiceName(), newMsg.CalleeServiceName()))
	require.True(t, reflect.DeepEqual(msg.CallerApp(), newMsg.CallerApp()))
	require.True(t, reflect.DeepEqual(msg.CallerServer(), newMsg.CallerServer()))
	require.True(t, reflect.DeepEqual(msg.CallerService(), newMsg.CallerService()))
	require.True(t, reflect.DeepEqual(msg.CallerMethod(), newMsg.CallerMethod()))
}

func TestMsg_CopyMsg_2_CIFunctionStatementsMustLessThan80Lines(t *testing.T) {
	ctx := context.Background()
	msg := codec.Message(ctx)
	type foo struct {
		I int
	}

	msg.WithCalleeApp("14")
	msg.WithCalleeServer("15")
	msg.WithCalleeService("16")
	msg.WithCalleeMethod("17")
	msg.WithCalleeContainerName("18")
	msg.WithServerMetaData(codec.MetaData{"a": []byte("1")})
	msg.WithFrameHead(foo{I: 1})
	msg.WithServerReqHead(foo{I: 2})
	msg.WithServerRspHead(foo{I: 3})
	msg.WithDyeing(true)
	msg.WithDyeingKey("19")
	msg.WithServerRspErr(errors.New("err1"))
	msg.WithClientMetaData(codec.MetaData{"b": []byte("2")})
	msg.WithClientReqHead(foo{I: 4})
	msg.WithClientRspErr(errors.New("err2"))
	msg.WithClientRspHead(foo{I: 5})
	msg.WithLogger(foo{I: 6})
	msg.WithRequestID(3)
	msg.WithStreamID(4)
	msg.WithStreamFrame(foo{I: 6})
	msg.WithCalleeSetName("20")
	msg.WithCommonMeta(codec.CommonMeta{21: []byte("hello")})
	msg.WithCallType(codec.SendOnly)

	_, newMsg := codec.WithNewMessage(ctx)
	codec.CopyMsg(newMsg, msg)

	require.False(t, reflect.DeepEqual(msg.Context(), newMsg.Context()))
	require.True(t, reflect.DeepEqual(msg.CalleeApp(), newMsg.CalleeApp()))
	require.True(t, reflect.DeepEqual(msg.CalleeServer(), newMsg.CalleeServer()))
	require.True(t, reflect.DeepEqual(msg.CalleeService(), newMsg.CalleeService()))
	require.True(t, reflect.DeepEqual(msg.CalleeMethod(), newMsg.CalleeMethod()))
	require.True(t, reflect.DeepEqual(msg.CalleeContainerName(), newMsg.CalleeContainerName()))
	require.True(t, reflect.DeepEqual(msg.ServerMetaData(), newMsg.ServerMetaData()))
	require.True(t, reflect.DeepEqual(msg.FrameHead(), newMsg.FrameHead()))
	require.True(t, reflect.DeepEqual(msg.ServerReqHead(), newMsg.ServerReqHead()))
	require.True(t, reflect.DeepEqual(msg.ServerRspHead(), newMsg.ServerRspHead()))
	require.True(t, reflect.DeepEqual(msg.Dyeing(), newMsg.Dyeing()))
	require.True(t, reflect.DeepEqual(msg.DyeingKey(), newMsg.DyeingKey()))
	require.True(t, reflect.DeepEqual(msg.ServerRspErr(), newMsg.ServerRspErr()))
	require.True(t, reflect.DeepEqual(msg.ClientMetaData(), newMsg.ClientMetaData()))
	require.True(t, reflect.DeepEqual(msg.ClientReqHead(), newMsg.ClientReqHead()))
	require.True(t, reflect.DeepEqual(msg.ClientRspErr(), newMsg.ClientRspErr()))
	require.True(t, reflect.DeepEqual(msg.ClientRspHead(), newMsg.ClientRspHead()))
	require.True(t, reflect.DeepEqual(msg.Logger(), newMsg.Logger()))
	require.True(t, reflect.DeepEqual(msg.RequestID(), newMsg.RequestID()))
	require.True(t, reflect.DeepEqual(msg.StreamID(), newMsg.StreamID()))
	require.True(t, reflect.DeepEqual(msg.StreamFrame(), newMsg.StreamFrame()))
	require.True(t, reflect.DeepEqual(msg.CalleeSetName(), newMsg.CalleeSetName()))
	require.True(t, reflect.DeepEqual(msg.CommonMeta(), newMsg.CommonMeta()))
	require.True(t, reflect.DeepEqual(msg.CallType(), newMsg.CallType()))

	// make sure map is deeply copied.
	newMsg.ServerMetaData()["aa"] = []byte("11")
	require.False(t, reflect.DeepEqual(msg.ServerMetaData(), newMsg.ServerMetaData()))
	newMsg.ClientMetaData()["bb"] = []byte("22")
	require.False(t, reflect.DeepEqual(msg.ClientMetaData(), newMsg.ClientMetaData()))

}

func TestEnsureMessage(t *testing.T) {
	ctx := context.Background()
	newCtx, msg := codec.EnsureMessage(ctx)
	require.NotEqual(t, ctx, newCtx)
	require.Equal(t, msg, codec.Message(newCtx))

	ctx = trpc.BackgroundContext()
	msg = codec.Message(ctx)
	require.NotNil(t, msg)
	newCtx, newMsg := codec.EnsureMessage(ctx)
	require.Equal(t, ctx, newCtx)
	require.Equal(t, msg, newMsg)
}

func TestSetMethodNameUsingRPCName(t *testing.T) {
	msg := codec.Message(context.Background())
	testSetMethodNameUsingRPCName(t, msg, msg.WithServerRPCName)
	testSetMethodNameUsingRPCName(t, msg, msg.WithClientRPCName)
}

func testSetMethodNameUsingRPCName(t *testing.T, msg codec.Msg, msgWithRPCName func(string)) {
	var cases = []struct {
		name           string
		originalMethod string
		rpcName        string
		expectMethod   string
	}{
		{"normal trpc rpc name", "", "/trpc.app.server.service/method", "method"},
		{"normal http url path", "", "/v1/subject/info/get", "/v1/subject/info/get"},
		{"invalid trpc rpc name (method name is empty)", "", "trpc.app.server.service", "trpc.app.server.service"},
		{"invalid trpc rpc name (method name is not mepty)", "/v1/subject/info/get", "trpc.app.server.service", "/v1/subject/info/get"},
		{"valid trpc rpc name will override existing method name", "/v1/subject/info/get", "/trpc.app.server.service/method", "method"},
		{"invalid trpc rpc will not override existing method name", "/v1/subject/info/get", "/trpc.app.server.service", "/v1/subject/info/get"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resetMsgRPCNameAndMethodName(msg)
			msg.WithCalleeMethod(tt.originalMethod)
			msgWithRPCName(tt.rpcName)
			method := msg.CalleeMethod()
			if method != tt.expectMethod {
				t.Errorf("given original method %s and rpc name %s, expect new method name %s, got %s",
					tt.originalMethod, tt.rpcName, tt.expectMethod, method)
			}
		})
	}
}

func resetMsgRPCNameAndMethodName(msg codec.Msg) {
	msg.WithCalleeMethod("")
	msg.WithClientRPCName("")
	msg.WithServerRPCName("")
}
