// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package trpc_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var ctx = context.Background()

func init() {
	trpc.LoadGlobalConfig("testdata/trpc_go.yaml")
	trpc.Setup(trpc.GlobalConfig())
}

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestCodec(t *testing.T) {

	msg := trpc.Message(ctx)
	assert.NotNil(t, msg)

	ctx := trpc.BackgroundContext()
	assert.NotNil(t, ctx)

	val := trpc.GetMetaData(ctx, "no-exist")
	assert.Nil(t, val)

	trpc.SetMetaData(ctx, "exist1", []byte("value1"))
	val = trpc.GetMetaData(ctx, "exist1")
	assert.NotNil(t, val)
	assert.Equal(t, []byte("value1"), val)

	trpc.SetMetaData(ctx, "exist2", []byte("value2"))
	val = trpc.GetMetaData(ctx, "exist2")
	assert.NotNil(t, val)
	assert.Equal(t, []byte("value2"), val)

	serverCodec := codec.GetServer("trpc")
	clientCodec := codec.GetClient("trpc")
	frameBuilder := transport.GetFramerBuilder("trpc")

	assert.Equal(t, trpc.DefaultServerCodec, serverCodec)
	assert.Equal(t, trpc.DefaultClientCodec, clientCodec)

	request := trpc.Request(ctx)
	response := trpc.Response(ctx)
	assert.NotNil(t, request)
	assert.NotNil(t, response)

	msg = trpc.Message(ctx)
	msg.WithServerReqHead(request)
	msg.WithServerRspHead(response)
	request = trpc.Request(ctx)
	response = trpc.Response(ctx)
	assert.NotNil(t, request)
	assert.NotNil(t, response)

	request.Func = []byte("test")
	data, err := proto.Marshal(request)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x3a, 0x4, 0x74, 0x65, 0x73, 0x74}, data)

	response.ErrorMsg = []byte("ok")
	data, err = proto.Marshal(response)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x32, 0x2, 0x6f, 0x6b}, data)

	// frame head: 2 bytes magic stx(0x930) + 1 byte stream type(1) + 1 byte stream frame type(2)
	// + 4 bytes total len(23) + 2 bytes pb header len(6) + 4 bytes stream id(0)
	// + 2 bytes reserved(0) + head + body
	in := []byte{0x9, 0x30, 0, 2, 0, 0, 0, 23, 0, 6, 0, 0, 0, 0, 0, 0, 0x3a, 0x4, 0x74, 0x65, 0x73, 0x74, 1}
	reader := bytes.NewReader(in)

	reader = bytes.NewReader(in)
	frame := frameBuilder.New(reader)
	data, err = frame.ReadFrame()
	assert.Nil(t, err)
	assert.Equal(t, in, data)

	// invalid magic num
	in1 := []byte{0x30, 0x9, 1, 2, 0, 0, 0, 23, 0, 6, 0, 0, 0, 0, 0, 0, 0x3a, 0x4, 0x74, 0x65, 0x73, 0x74, 1}
	reader = bytes.NewReader(in1)

	msg = codec.Message(ctx)
	reqBody, err := serverCodec.Decode(msg, in)
	assert.Nil(t, err)
	assert.Equal(t, []byte{1}, reqBody)

	// head len invalid
	in2 := []byte{0x30, 0x9, 0, 2, 0, 0, 0, 23, 0, 7, 0, 0, 0, 0, 0, 0, 0x3a, 0x4, 0x74, 0x65, 0x73, 0x74, 1}
	reqBody2, err := serverCodec.Decode(msg, in2)
	assert.NotNil(t, err)
	assert.Nil(t, reqBody2)

	rspBuf, err := serverCodec.Encode(msg, reqBody)
	assert.Nil(t, err)
	assert.NotNil(t, rspBuf)

	reqBuf, err := clientCodec.Encode(msg, reqBody)
	assert.Nil(t, err)
	assert.NotNil(t, reqBuf)

	in3 := []byte{0x9, 0x30, 0, 2, 0, 0, 0, 21, 0, 4, 0, 0, 0, 0, 0, 0, 0x32, 0x2, 0x6f, 0x6b, 1}
	msg.ClientReqHead().(*trpcpb.RequestProtocol).RequestId = 0
	rspBody, err := clientCodec.Decode(msg, in3)
	assert.Nil(t, err)
	assert.Equal(t, []byte{1}, rspBody)
}

func TestVersion(t *testing.T) {
	version := trpc.Version()

	assert.NotNil(t, version)
}

func TestConfig(t *testing.T) {

	trpc.ServerConfigPath = "./testdata/trpc_go.yaml"

	conf := trpc.GlobalConfig()
	assert.NotNil(t, conf)
	assert.Equal(t, 3, len(conf.Server.Service))
	assert.Equal(t, "trpc.test.helloworld.Greeter1", conf.Server.Service[0].Name)
	assert.Equal(t, true, *conf.Server.Service[0].ServerAsync)
	assert.Equal(t, 1000, conf.Server.Service[1].MaxRoutines)
	assert.Equal(t, false, *conf.Server.Service[0].Writev)

	cfg := &trpc.Config{}
	cfg.Server.Network = "tcp"
	cfg.Server.Protocol = "trpc"
	cfg.Client.Network = "tcp"
	cfg.Client.Protocol = "trpc"
	trpc.SetGlobalConfig(cfg)
}

func TestNewServer(t *testing.T) {

	trpc.ServerConfigPath = "./testdata/trpc_go.yaml"

	logger := log.NewZapLog(log.Config{
		{
			Writer: log.OutputFile,
			WriteConfig: log.WriteConfig{
				LogPath:   os.TempDir(),
				Filename:  "trpc.log",
				WriteMode: 1,
			},
			Level: "DEBUG",
		},
	})
	dftLogger := log.DefaultLogger
	log.SetLogger(logger)
	defer log.SetLogger(dftLogger)

	fp := filepath.Join(os.TempDir(), "trpc.log")
	defer os.Remove(fp)

	s := trpc.NewServer()
	assert.NotNil(t, s)
	assert.NotNil(t, s.Service("trpc.test.helloworld.Greeter1"))
	assert.NotNil(t, s.Service("trpc.test.helloworld.Greeter2"))
	assert.NotNil(t, s.Service("trpc.test.helloworld.Greeter3"))
	assert.Equal(t, codec.DefaultReaderSize, codec.GetReaderSize())

	buf, err := os.ReadFile(fp)
	assert.Nil(t, err)

	// test namingservice not exist
	// registry set for service1、service2
	assert.Contains(t, string(buf), "trpc.test.helloworld.Greeter1 registry not exist")
	assert.Contains(t, string(buf), "trpc.test.helloworld.Greeter2 registry not exist")
	// registry not set for service3
	assert.NotContains(t, string(buf), "trpc.test.helloworld.Greeter3 registry not exist")
}

func TestProtocol(t *testing.T) {
	request := trpc.Request(ctx)
	response := trpc.Response(ctx)

	assert.NotNil(t, request.String())
	assert.NotNil(t, response.String())
	assert.Equal(t, uint32(0), request.GetContentType())
	assert.Equal(t, uint32(0), request.GetRequestId())
	assert.Equal(t, uint32(0), request.GetCallType())
	assert.Equal(t, uint32(0), request.GetVersion())
	assert.Equal(t, uint32(0), request.GetMessageType())
	assert.Nil(t, response.GetErrorMsg())
	assert.Nil(t, request.GetCallee())
	assert.Nil(t, request.GetCaller())
}

func TestGetAdminService(t *testing.T) {
	cfg := t.TempDir() + "trpc_go.yaml"
	require.Nil(t, os.WriteFile(cfg, []byte{}, 0644))
	oldPath := trpc.ServerConfigPath
	trpc.ServerConfigPath = cfg
	defer func() { trpc.ServerConfigPath = oldPath }()

	s := trpc.NewServer()
	admin, err := trpc.GetAdminService(trpc.NewServer())
	require.Nil(t, err)
	require.NotNil(t, admin)

	require.Nil(t, os.WriteFile(cfg, []byte(`
server:
  admin:
    port: 9528
`), 0644))

	s = trpc.NewServer()
	adminService, err := trpc.GetAdminService(s)
	require.Nil(t, err)
	require.NotNil(t, adminService)
}

func TestNewServerWithClosablePlugin(t *testing.T) {
	closed := make(chan struct{})
	plugin.Register("default", &closablePlugin{onClose: func() error {
		close(closed)
		return nil
	}})

	cfg := t.TempDir() + "trpc_go.yaml"
	require.Nil(t, os.WriteFile(cfg, []byte(`
plugins:
  closable_plugin:
    default:
`), 0644))
	oldPath := trpc.ServerConfigPath
	trpc.ServerConfigPath = cfg
	defer func() { trpc.ServerConfigPath = oldPath }()

	s := trpc.NewServer()
	require.Nil(t, s.Close(nil))
	select {
	case <-closed:
	default:
		require.FailNow(t, "plugin is not closed when server close")
	}
}

type closablePlugin struct {
	onClose func() error
}

func (*closablePlugin) Type() string {
	return "closable_plugin"
}

func (*closablePlugin) Setup(string, plugin.Decoder) error {
	return nil
}

func (p *closablePlugin) Close() error {
	return p.onClose()
}
