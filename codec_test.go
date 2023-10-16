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
	"encoding/binary"
	"errors"
	"log"
	"net"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/pool/multiplexed"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func TestFramer_ReadFrame(t *testing.T) {
	// test magic num mismatch
	{
		var err error
		totalLen := 0
		buf := new(bytes.Buffer)
		// MagicNum 0x930, 2bytes
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint16(trpcpb.TrpcMagic_TRPC_MAGIC_VALUE+1)))
		// frame type, 1byte
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint8(0)))
		// stream frame type, 1byte
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint8(0)))
		// total len
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint32(totalLen)))
		// pb header len
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint16(0)))
		// stream ID
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint16(0)))
		// reserved
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint32(0)))
		assert.Nil(t, err)

		fb := &trpc.FramerBuilder{}
		fr := fb.New(bytes.NewReader(buf.Bytes()))
		assert.NotNil(t, fr)
		_, err = fr.ReadFrame()
		assert.NotNil(t, err)
	}

	// test total len exceed max error
	{
		var err error
		totalLen := trpc.DefaultMaxFrameSize + 1
		buf := new(bytes.Buffer)
		// MagicNum 0x930, 2bytes
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint16(trpcpb.TrpcMagic_TRPC_MAGIC_VALUE)))
		// frame type, 1byte
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint8(0)))
		// stream frame type, 1byte
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint8(0)))
		// total len
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint32(totalLen)))
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint16(0)))
		// stream ID
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint16(0)))
		// reserved
		assert.Nil(t, binary.Write(buf, binary.BigEndian, uint32(0)))
		assert.Nil(t, err)

		fb := &trpc.FramerBuilder{}
		fr := fb.New(bytes.NewReader(buf.Bytes()))
		assert.NotNil(t, fr)
		_, err = fr.ReadFrame()
		assert.NotNil(t, err)
	}
}

func TestClientCodecEnvTransfer(t *testing.T) {
	envTransfer := []byte("env transfer")
	cliCodec := &trpc.ClientCodec{}

	// if msg.EnvTransfer() empty, transmitted env info in req.TransInfo should be cleared
	_, msg := codec.WithNewMessage(context.Background())
	msg.WithClientMetaData(map[string][]byte{trpc.EnvTransfer: envTransfer})
	msg.WithEnvTransfer("")
	reqBuf, err := cliCodec.Encode(msg, nil)
	assert.Nil(t, err)
	head := &trpcpb.RequestProtocol{}
	err = proto.Unmarshal(reqBuf[16:], head)
	assert.Nil(t, err)
	assert.Equal(t, head.TransInfo[trpc.EnvTransfer], []byte{})

	// msg.EnvTransfer() not empty
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithEnvTransfer("env transfer")
	reqBuf, err = cliCodec.Encode(msg, nil)
	assert.Nil(t, err)
	head = &trpcpb.RequestProtocol{}
	err = proto.Unmarshal(reqBuf[16:], head)
	assert.Nil(t, err)
	assert.Equal(t, head.TransInfo[trpc.EnvTransfer], envTransfer)
}

func TestClientCodecDyeing(t *testing.T) {
	dyeingKey := "123456789"
	cliCodec := &trpc.ClientCodec{}
	_, msg := codec.WithNewMessage(context.Background())
	msg.WithDyeingKey(dyeingKey)
	reqBuf, err := cliCodec.Encode(msg, nil)
	assert.Nil(t, err)
	head := &trpcpb.RequestProtocol{}
	err = proto.Unmarshal(reqBuf[16:], head)
	assert.Nil(t, err)
	assert.Equal(t, head.TransInfo[trpc.DyeingKey], []byte(dyeingKey))
}

func TestFramerBuilder(t *testing.T) {
	t.Run("frame build is a SafeFramer", func(t *testing.T) {
		fb := trpc.FramerBuilder{}
		frame := fb.New(bytes.NewReader(nil))
		require.True(t, frame.(codec.SafeFramer).IsSafe())
	})
	t.Run("ok, read valid response", func(t *testing.T) {
		bts := mustEncode(t, []byte("hello-world"))
		vid, buf, err := (&trpc.FramerBuilder{}).Parse(bytes.NewReader(bts))
		require.Nil(t, err)
		require.Zero(t, vid)
		require.Equal(t, bts, buf)
	})
	t.Run("garbage data", func(t *testing.T) {
		_, _, err := (&trpc.FramerBuilder{}).Parse(bytes.NewReader([]byte("hello-world xxxxxxxxxxxx")))
		require.Regexp(t, regexp.MustCompile(`magic .+ not match`), err.Error())
	})
}

func mustEncode(t *testing.T, body []byte) (buffer []byte) {
	t.Helper()

	msgHead := &trpcpb.RequestProtocol{
		Version: uint32(trpcpb.TrpcProtoVersion_TRPC_PROTO_V1),
		Callee:  []byte("trpc.test.helloworld.Greetor"),
		Func:    []byte("/trpc.test.helloworld.Greetor/SayHello"),
	}
	head, err := proto.Marshal(msgHead)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	// MagicNum 0x930, 2bytes
	if err := binary.Write(buf, binary.BigEndian, uint16(trpcpb.TrpcMagic_TRPC_MAGIC_VALUE)); err != nil {
		t.Fatal(err)
	}
	// frame type, 1byte
	if err := binary.Write(buf, binary.BigEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	// stream frame type, 1byte
	if err := binary.Write(buf, binary.BigEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	// total len
	totalLen := 16 + len(head) + len(body)
	if err := binary.Write(buf, binary.BigEndian, uint32(totalLen)); err != nil {
		t.Fatal(err)
	}
	// pb header len
	if err := binary.Write(buf, binary.BigEndian, uint16(len(head))); err != nil {
		t.Fatal(err)
	}
	// stream ID
	if err := binary.Write(buf, binary.BigEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	// reserved
	if err := binary.Write(buf, binary.BigEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	// header
	if err := binary.Write(buf, binary.BigEndian, head); err != nil {
		t.Fatal(err)
	}
	// body
	if err := binary.Write(buf, binary.BigEndian, body); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestClientCodec_DecodeHeadOverflowsUint16(t *testing.T) {
	cc := trpc.ClientCodec{}
	msg := codec.Message(trpc.BackgroundContext())

	msg.WithClientMetaData(codec.MetaData{"smallBuffer": make([]byte, 16)})
	rspBuf, err := cc.Encode(msg, nil)
	require.Nil(t, err)
	require.Contains(t, string(rspBuf), "smallBuffer")

	msg.WithClientMetaData(map[string][]byte{"largeBuffer": make([]byte, 64*1024)})
	_, err = cc.Encode(msg, nil)
	require.NotNil(t, err)
}

func TestServerCodec_DecodeHeadOverflowsUint16(t *testing.T) {
	cc := trpc.ServerCodec{}
	msg := codec.Message(trpc.BackgroundContext())

	msg.WithServerMetaData(map[string][]byte{"smallBuffer": make([]byte, 16)})
	rspBuf, err := cc.Encode(msg, nil)
	require.Nil(t, err)
	require.Contains(t, string(rspBuf), "smallBuffer")

	msg.WithServerMetaData(
		map[string][]byte{
			"smallBuffer": make([]byte, 16),
			"largeBuffer": make([]byte, 64*1024),
		})
	rspBuf, err = cc.Encode(msg, nil)
	require.Nil(t, err)
	require.Less(t, len(rspBuf), 64*1024)
	require.NotContains(t, string(rspBuf), "smallBuffer")
	require.NotContains(t, string(rspBuf), "largeBuffer")
}

func TestClientCodec_CallTypeEncode(t *testing.T) {
	sc := trpc.ClientCodec{}
	msg := codec.Message(trpc.BackgroundContext())
	msg.WithCallType(codec.SendOnly)
	reqBuf, err := sc.Encode(msg, nil)
	assert.Nil(t, err)
	head := &trpcpb.RequestProtocol{}
	err = proto.Unmarshal(reqBuf[16:], head)
	assert.Nil(t, err)
	assert.Equal(t, head.GetCallType(), uint32(codec.SendOnly))
}

func TestServerCodec_CallTypeDecode(t *testing.T) {
	cc := trpc.ClientCodec{}
	sc := trpc.ServerCodec{}
	msg := codec.Message(trpc.BackgroundContext())
	msg.WithCallType(codec.SendOnly)
	reqBuf, err := cc.Encode(msg, nil)
	assert.Nil(t, err)
	_, err = sc.Decode(msg, reqBuf)
	assert.Nil(t, err)
	assert.Equal(t, msg.CallType(), codec.SendOnly)
}

func TestClientCodec_EncodeErr(t *testing.T) {
	t.Run("head len overflows uint16", func(t *testing.T) {
		cc := trpc.ClientCodec{}
		msg := codec.Message(trpc.BackgroundContext())
		msg.WithClientMetaData(codec.MetaData{"overHeadLengthU16": make([]byte, 64*1024)})
		_, err := cc.Encode(msg, nil)
		assert.EqualError(t, err, "head len overflows uint16")
	})
	t.Run("frame len is too large", func(t *testing.T) {
		cc := trpc.ClientCodec{}
		msg := codec.Message(trpc.BackgroundContext())
		_, err := cc.Encode(msg, make([]byte, trpc.DefaultMaxFrameSize))
		assert.EqualError(t, err, "frame len is larger than MaxFrameSize(10485760)")
	})
	t.Run("encoding attachment failed", func(t *testing.T) {
		cc := trpc.ClientCodec{}
		msg := codec.Message(trpc.BackgroundContext())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &attachment.Attachment{Request: &errorReader{}, Response: attachment.NoopAttachment{}}})
		_, err := cc.Encode(msg, nil)
		assert.EqualError(t, err, "encoding attachment: reading errorReader always returns error")
	})

}

type errorReader struct{}

func (*errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("reading errorReader always returns error")
}

func TestServerCodec_EncodeErr(t *testing.T) {
	t.Run("head len overflows uint16", func(t *testing.T) {
		msg := codec.Message(trpc.BackgroundContext())
		sc := trpc.ServerCodec{}
		msg.WithServerMetaData(codec.MetaData{"overHeadLengthU16": make([]byte, 64*1024)})
		rspBuf, err := sc.Encode(msg, nil)
		assert.Nil(t, err)

		head := &trpcpb.ResponseProtocol{}
		err = proto.Unmarshal(rspBuf[16:], head)
		assert.Nil(t, err)
		assert.Equal(t, int32(errs.RetServerEncodeFail), head.GetRet())
	})
	t.Run("frame len is too large", func(t *testing.T) {
		msg := codec.Message(trpc.BackgroundContext())
		sc := trpc.ServerCodec{}
		rspBuf, err := sc.Encode(msg, make([]byte, trpc.DefaultMaxFrameSize))
		assert.Nil(t, err)

		head := &trpcpb.ResponseProtocol{}
		err = proto.Unmarshal(rspBuf[16:], head)
		assert.Nil(t, err)
		assert.Equal(t, int32(errs.RetServerEncodeFail), head.GetRet())
	})
	t.Run("encoding attachment failed", func(t *testing.T) {
		msg := codec.Message(trpc.BackgroundContext())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ServerAttachmentKey{}: &attachment.Attachment{Request: attachment.NoopAttachment{}, Response: &errorReader{}}})
		sc := trpc.ServerCodec{}
		_, err := sc.Encode(msg, nil)
		assert.EqualError(t, err, "encoding attachment: reading errorReader always returns error")
	})
}

func TestMultiplexFrame(t *testing.T) {
	buf := mustEncode(t, []byte("helloworld"))
	vid, frame, err := (&trpc.FramerBuilder{}).Parse(bytes.NewReader(buf))
	require.Nil(t, err)
	require.Equal(t, uint32(0), vid)
	require.Equal(t, buf, frame)
}

func TestClientCodecNoModifyOriginalFrameHead(t *testing.T) {
	_, msg := codec.WithNewMessage(context.Background())
	fh := &trpc.FrameHead{
		StreamID: 101,
	}
	msg.WithFrameHead(fh)
	clientCodec := &trpc.ClientCodec{}
	_, err := clientCodec.Encode(msg, []byte("helloworld"))
	require.Nil(t, err)
	require.Equal(t, uint32(101), fh.StreamID)
}

// GOMAXPROCS=1 go test -bench=ServerCodec_Decode -benchmem
// -benchtime=10s -memprofile mem.out -cpuprofile cpu.out codec_test.go
func BenchmarkServerCodec_Decode(b *testing.B) {
	sc := &trpc.ServerCodec{}
	cc := &trpc.ClientCodec{}
	_, msg := codec.WithNewMessage(context.Background())

	reqBody, err := proto.Marshal(&pb.HelloRequest{
		Msg: "helloworld",
	})
	assert.Nil(b, err)

	req, err := cc.Encode(msg, reqBody)
	assert.Nil(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		sc.Decode(msg, req)
	}
}

// GOMAXPROCS=1 go test -bench=ClientCodec_Encode -benchmem -benchtime=10s
// -memprofile mem.out -cpuprofile cpu.out codec_test.go
func BenchmarkClientCodec_Encode(b *testing.B) {
	cc := &trpc.ClientCodec{}

	_, msg := codec.WithNewMessage(context.Background())
	reqBody, err := proto.Marshal(&pb.HelloRequest{
		Msg: "helloworld",
	})
	assert.Nil(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cc.Encode(msg, reqBody)
	}
}

func TestUDPParseFail(t *testing.T) {
	s := &udpServer{}
	s.start(context.Background())
	t.Cleanup(s.stop)

	m := multiplexed.New(multiplexed.WithConnectNumber(1))
	test := func(id uint32, buf []byte, wantErr error) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		opts := multiplexed.NewGetOptions()
		opts.WithVID(id)
		opts.WithFrameParser(&trpc.FramerBuilder{})
		mc, err := m.GetMuxConn(ctx, s.conn.LocalAddr().Network(), s.conn.LocalAddr().String(), opts)
		assert.Nil(t, err)
		require.Nil(t, mc.Write(buf))
		_, err = mc.Read()
		assert.Equal(t, err, wantErr)
		cancel()
	}
	// fail when parse invalid buf
	var id uint32 = 1
	test(id, []byte("invalid buf"), context.DeadlineExceeded)

	// succeed when parse valid buf
	id = 2
	msg := codec.Message(context.Background())
	msg.WithFrameHead(&trpc.FrameHead{
		StreamID: id,
	})
	sc := &trpc.ServerCodec{}
	buf, _ := sc.Encode(msg, []byte("helloworld"))
	test(id, buf, nil)
}

type udpServer struct {
	cancel context.CancelFunc
	conn   net.PacketConn
}

func (s *udpServer) start(ctx context.Context) error {
	var err error
	s.conn, err = net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, addr, err := s.conn.ReadFrom(buf)
			if err != nil {
				log.Println("l.ReadFrom err: ", err)
				return
			}
			s.conn.WriteTo(buf[:n], addr)
		}
	}()
	return nil
}

func (s *udpServer) stop() {
	s.cancel()
	s.conn.Close()
}
