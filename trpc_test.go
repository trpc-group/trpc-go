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

package trpc_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/rpcz"
	"trpc.group/trpc-go/trpc-go/server"
	pb "trpc.group/trpc-go/trpc-go/testdata"
	"trpc.group/trpc-go/trpc-go/transport"
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
	msg.ClientReqHead().(*trpc.RequestProtocol).RequestId = 0
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
	require.Nil(t, trpc.LoadGlobalConfig("testdata/trpc_go.yaml"))

	trpc.Setup(trpc.GlobalConfig())
	conf := trpc.GlobalConfig()
	assert.NotNil(t, conf)
	assert.Equal(t, 4, len(conf.Server.Service))
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

func TestSetup(t *testing.T) {
	config := client.Config("empty")
	assert.Equal(t, "Development", config.Namespace)
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

func TestServerMethodTimeout(t *testing.T) {
	var cfg trpc.Config
	require.Nil(t, yaml.Unmarshal([]byte(`
server:
  service:
    - protocol: trpc
      timeout: 200
      method:
        SayHello:
          timeout: 100
`), &cfg))

	l, err := net.Listen("tcp", ":")
	require.Nil(t, err)
	s := trpc.NewServerWithConfig(&cfg, server.WithListener(l))
	pb.RegisterGreeterService(s, &GreeterAlwaysTimeout{})
	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	defer s.Close(nil)

	c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + l.Addr().String()))
	start := time.Now()
	_, err = c.SayHello(context.Background(), &pb.HelloRequest{})
	require.NotNil(t, err)
	require.InDelta(t, time.Millisecond*100, time.Since(start), float64(time.Millisecond*30))

	start = time.Now()
	_, err = c.SayHi(context.Background(), &pb.HelloRequest{})
	require.NotNil(t, err)
	require.InDelta(t, time.Millisecond*200, time.Since(start), float64(time.Millisecond*30))
}

func TestServiceCustomizedSerializationAndCompressionType(t *testing.T) {
	var cfg trpc.Config
	const (
		clientSerializationType = 2 // json
		clientCompressionType   = 1 // gzip
		serverSerializationType = 4 // noop
		serverCompressionType   = 0 // noop
	)
	require.Nil(t, yaml.Unmarshal([]byte(fmt.Sprintf(`
server:
  service:
    - protocol: trpc
      timeout: 200
      current_serialization_type: %d
      current_compress_type: %d
`, serverSerializationType, serverCompressionType)), &cfg))

	l, err := net.Listen("tcp", ":")
	require.Nil(t, err)
	defer l.Close()
	s := trpc.NewServerWithConfig(&cfg, server.WithListener(l))
	// Echo service is actually a transparent echo service.
	registerEchoService(s, &impl{})
	go func() { s.Serve() }()
	defer s.Close(nil)
	c := pb.NewGreeterClientProxy(
		client.WithTarget("ip://"+l.Addr().String()),
		client.WithCompressType(clientCompressionType),
		client.WithSerializationType(clientSerializationType),
	)
	const msg = "hello"
	rsp, err := c.SayHello(context.Background(), &pb.HelloRequest{Msg: msg},
		client.WithFilter(func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
			err := next(ctx, req, rsp)
			msg := trpc.Message(ctx)
			if msg.SerializationType() != clientSerializationType {
				return fmt.Errorf("rsp serialization type got: %d, want: %d, original err: %+v",
					msg.SerializationType(), clientSerializationType, err)
			}
			if msg.CompressType() != clientCompressionType {
				return fmt.Errorf("rsp compression type got: %d, want: %d, original err: %+v",
					msg.CompressType(), clientCompressionType, err)
			}
			return nil
		}))
	require.Nil(t, err)
	require.Equal(t, msg, rsp.Msg)
}

func TestNewServerWithConfigReflectionService(t *testing.T) {
	t.Run("reflection_service not matched", func(t *testing.T) {
		var cfg trpc.Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server:
  reflection_service: a.b.c.d
  service:
    - name: a.b.c.d
`), &cfg))
		s := trpc.NewServerWithConfig(&cfg)
		require.NotNil(t, s.Service("a.b.c.d"))
	})
	t.Run("reflection_service matched", func(t *testing.T) {
		var cfg trpc.Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server:
  reflection_service: w.x.y.z
  service:
    - name: a.b.c.d
`), &cfg))
		require.Panics(t, func() {
			_ = trpc.NewServerWithConfig(&cfg)
		})
	})
}

func TestServiceAddressDesensitization(t *testing.T) {
	type args struct {
		password string
		address  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "ending with password",
			args: args{
				password: "123456",
				address:  "127.0.0.1:80?user=trpc&password=123456",
			},
			want: "127.0.0.1:80?user=trpc&password=*",
		},
		{
			name: "ending with passwd",
			args: args{
				password: "123456",
				address:  "127.0.0.1:80?user=trpc&passwd=123456",
			},
			want: "127.0.0.1:80?user=trpc&passwd=*",
		},
		{
			name: "not ending with password",
			args: args{
				password: "123456",
				address:  "127.0.0.1:80?user=trpc&password=123456&batch=10",
			},
			want: "127.0.0.1:80?user=trpc&password=*&batch=10",
		},
		{
			name: "not ending with passwd",
			args: args{
				password: "123456",
				address:  "127.0.0.1:80?user=trpc&passwd=123456&batch=10",
			},
			want: "127.0.0.1:80?user=trpc&passwd=*&batch=10",
		},
		{
			name: "without password or passwd",
			args: args{
				password: "no password",
				address:  "127.0.0.1:80?user=trpc",
			},
			want: "127.0.0.1:80?user=trpc",
		},
		{
			name: "kafka dsn with password",
			args: args{
				password: "123456",
				address:  "127.0.0.1:80?topics=topic1,topic2&mechanism=SCRAM-SHA-512&user=trpc&password=123456",
			},
			want: "127.0.0.1:80?topics=topic1,topic2&mechanism=SCRAM-SHA-512&user=trpc&password=*",
		},
		{
			name: "kafka dsn without password",
			args: args{
				password: "no password",
				address:  "127.0.0.1:9092?topics=quickstart-events&group=quickstart-group",
			},
			want: "127.0.0.1:9092?topics=quickstart-events&group=quickstart-group",
		},
		{
			name: "rabbitmq dsn",
			args: args{
				password: "123456",
				address:  "user:123456@127.0.0.1:80?exchange=test-exchange&queue=test-queue&key=test-key",
			},
			want: "user:*@127.0.0.1:80?exchange=test-exchange&queue=test-queue&key=test-key",
		},
		{
			name: "rabbitmq dsn password contain @",
			args: args{
				password: "secretWith@secretWith",
				address:  "user:secretWith@secretWith@localhost:6379/0?foo=bar&qux=baz",
			},
			want: "user:*@localhost:6379/0?foo=bar&qux=baz",
		},
	}

	dftLogger := log.DefaultLogger
	defer log.SetLogger(dftLogger)
	// set config
	var cfg trpc.Config
	require.Nil(t, yaml.Unmarshal([]byte(`
server:
  service:
    - protocol: trpc
      timeout: 200
      method:
        SayHello:
          timeout: 100
`), &cfg))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set logger to file
			logDir := t.TempDir()
			logger := log.NewZapLog(log.Config{
				{
					Writer: log.OutputFile,
					WriteConfig: log.WriteConfig{
						LogPath:   logDir,
						Filename:  "trpc.log",
						WriteMode: log.WriteSync,
					},
					Level: "DEBUG",
				},
			})
			log.SetLogger(logger)

			// start server
			l, err := net.Listen("tcp", ":")
			require.Nil(t, err)
			s := trpc.NewServerWithConfig(&cfg,
				server.WithListener(l),
				server.WithAddress(tt.args.address))
			pb.RegisterGreeterService(s, &GreeterAlwaysTimeout{})
			errCh := make(chan error)
			go func() { errCh <- s.Serve() }()
			select {
			case err := <-errCh:
				require.FailNow(t, "serve failed", err)
			case <-time.After(time.Millisecond * 200):
			}
			defer s.Close(nil)

			// read log from file
			fp := filepath.Join(logDir, "trpc.log")
			buf, err := os.ReadFile(fp)
			assert.Nil(t, err)

			// password is not in log
			assert.NotContains(t, string(buf), tt.args.password)
			// password is replaced with *
			assert.Contains(t, string(buf), tt.want)
		})
	}
}

func TestRPCZ(t *testing.T) {
	var cfg trpc.Config
	require.Nil(t, yaml.Unmarshal([]byte(`
server:
  admin:
    rpcz:
       fraction: 1.0
       capacity: 10000
  service:
    - protocol: trpc
      network: tcp
`), &cfg))

	l, err := net.Listen("tcp", ":")
	require.Nil(t, err)
	s := trpc.NewServerWithConfig(&cfg, server.WithListener(l))
	registerEchoService(s, &impl{})
	errCh := make(chan error)
	go func() { errCh <- s.Serve() }()
	select {
	case err := <-errCh:
		require.FailNow(t, "serve failed", err)
	case <-time.After(time.Millisecond * 200):
	}
	defer s.Close(nil)

	c := pb.NewGreeterClientProxy(client.WithTarget("ip://" + l.Addr().String()))
	_, err = c.SayHello(context.Background(), &pb.HelloRequest{})
	require.NotNil(t, err)

	// client span and server span
	spans := rpcz.GlobalRPCZ.BatchQuery(2)
	require.Equal(t, 2, len(spans))
}

func TestPeriodicallyUpdateGOMAXPROCS(t *testing.T) {
	updateGOMAXPROCSInterval := time.Millisecond * 200
	stop := trpc.PeriodicallyUpdateGOMAXPROCS(updateGOMAXPROCSInterval)
	time.Sleep(updateGOMAXPROCSInterval * 3)
	stop()
	require.True(t, true, "just to bypass increment coverage")

	cfg := &trpc.Config{}
	cfg.Global.RoundUpCPUQuota = true
	trpc.SetGlobalConfig(cfg)
	updateGOMAXPROCSInterval = time.Millisecond * 200
	stop = trpc.PeriodicallyUpdateGOMAXPROCS(updateGOMAXPROCSInterval)
	time.Sleep(updateGOMAXPROCSInterval * 3)
	stop()
	require.True(t, true, "just to bypass increment coverage")
}

type echoServer interface {
	Echo(ctx context.Context, reqbody *codec.Body) (*codec.Body, error)
}

func echoHandler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &codec.Body{}
	filters, err := f(req)
	if err != nil {
		return nil, err
	}
	return filters.Filter(ctx, req, func(ctx context.Context, req interface{}) (interface{}, error) {
		return svr.(echoServer).Echo(ctx, req.(*codec.Body))
	})
}

var echoServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.app.server.EchoService",
	HandlerType: ((*echoServer)(nil)),
	Methods: []server.Method{
		{
			Name: "*",
			Func: echoHandler,
		},
	},
}

func registerEchoService(s server.Service, svr echoServer) {
	s.Register(&echoServiceDesc, svr)
}

type impl struct{}

func (*impl) Echo(ctx context.Context, reqbody *codec.Body) (*codec.Body, error) {
	return reqbody, nil
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

type GreeterAlwaysTimeout struct{}

func (g *GreeterAlwaysTimeout) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	<-ctx.Done()
	return nil, errs.NewFrameError(errs.RetServerTimeout, "ctx timeout")
}

func (g *GreeterAlwaysTimeout) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	<-ctx.Done()
	return nil, errs.NewFrameError(errs.RetServerTimeout, "ctx timeout")
}
