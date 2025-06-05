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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

func TestFramer_ReadFrame(t *testing.T) {
	// test magic num mismatch
	{
		var err error
		totalLen := 0
		buf := new(bytes.Buffer)
		// MagicNum 0x930, 2bytes
		err = binary.Write(buf, binary.BigEndian, uint16(trpc.TrpcMagic_TRPC_MAGIC_VALUE+1))
		// frame type, 1byte
		err = binary.Write(buf, binary.BigEndian, uint8(0))
		// stream frame type, 1byte
		err = binary.Write(buf, binary.BigEndian, uint8(0))
		// total len
		err = binary.Write(buf, binary.BigEndian, uint32(totalLen))
		// pb header len
		err = binary.Write(buf, binary.BigEndian, uint16(0))
		// stream ID
		err = binary.Write(buf, binary.BigEndian, uint16(0))
		// reserved
		err = binary.Write(buf, binary.BigEndian, uint32(0))
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
		err = binary.Write(buf, binary.BigEndian, uint16(trpc.TrpcMagic_TRPC_MAGIC_VALUE))
		// frame type, 1byte
		err = binary.Write(buf, binary.BigEndian, uint8(0))
		// stream frame type, 1byte
		err = binary.Write(buf, binary.BigEndian, uint8(0))
		// total len
		err = binary.Write(buf, binary.BigEndian, uint32(totalLen))
		err = binary.Write(buf, binary.BigEndian, uint16(0))
		// stream ID
		err = binary.Write(buf, binary.BigEndian, uint16(0))
		// reserved
		err = binary.Write(buf, binary.BigEndian, uint32(0))
		assert.Nil(t, err)

		fb := &trpc.FramerBuilder{}
		fr := fb.New(bytes.NewReader(buf.Bytes()))
		assert.NotNil(t, fr)
		_, err = fr.ReadFrame()
		assert.NotNil(t, err)
	}
}

func TestReadFrameMagicMisMatch(t *testing.T) {
	buf := new(bytes.Buffer)
	_, err := buf.Write([]byte(`HTTP/1.1 200 OK
Date: Wed, 21 Oct 2023 07:28:00 GMT
Server: Apache/2.4.1 (Unix)
Last-Modified: Sat, 17 Oct 2023 19:15:00 GMT
Content-Length: 88
Content-Type: text/html; charset=UTF-8
Connection: close
	
<html>
<head>
  <title>An Example Page</title>
</head>
<body>
  Hello World, this is a very simple HTML document.
</body>
</html>`))
	require.NoError(t, err)

	fb := &trpc.FramerBuilder{}
	fr := fb.New(bytes.NewReader(buf.Bytes()))
	require.NotNil(t, fr)
	_, err = fr.ReadFrame()
	// The error is like:
	// trpc framer: read framer head magic 18516 != 2352, not match for the first two bytes of the TRPC packet,
	// the expected trpc protocol is not detected; received bytes are 18516 (hex: 0x4854, ASCII: 'HT'),
	// possible causes include: an HTTP response from the gateway, an incorrect protocol packet,
	// or corrupted response bytes that do not conform to any valid protocol
	t.Logf("read frame magic mismatch error: %v", err)
	require.Error(t, err)
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
	head := &trpc.RequestProtocol{}
	err = proto.Unmarshal(reqBuf[16:], head)
	assert.Nil(t, err)
	assert.Equal(t, head.TransInfo[trpc.EnvTransfer], []byte{})

	// msg.EnvTransfer() not empty
	_, msg = codec.WithNewMessage(context.Background())
	msg.WithEnvTransfer("env transfer")
	reqBuf, err = cliCodec.Encode(msg, nil)
	assert.Nil(t, err)
	head = &trpc.RequestProtocol{}
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
	head := &trpc.RequestProtocol{}
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
	t.Run("ok, message doesn't contain ResponseProtocol", func(t *testing.T) {
		bts := mustEncode(t, []byte("hello-world"))
		fb := trpc.FramerBuilder{}
		frame := fb.New(bytes.NewReader(bts))

		responseFrame, err := frame.(codec.Decoder).Decode()
		require.Nil(t, err)

		require.Zero(t, responseFrame.GetRequestID())
		require.Equal(t, []byte("hello-world"), responseFrame.GetResponseBuf())

		require.Nil(t, frame.(codec.Decoder).UpdateMsg(responseFrame, trpc.Message(context.Background())))
	})
	t.Run("ok, message contains ResponseProtocol", func(t *testing.T) {
		bts := mustEncode(t, []byte("hello-world"))
		fb := trpc.FramerBuilder{}
		frame := fb.New(bytes.NewReader(bts))

		responseFrame, err := frame.(codec.Decoder).Decode()
		require.Nil(t, err)

		require.Zero(t, responseFrame.GetRequestID())

		msg := trpc.Message(context.Background())
		msg.WithClientRspHead(&trpc.ResponseProtocol{RequestId: 1})
		require.Nil(t, frame.(codec.Decoder).UpdateMsg(responseFrame, msg))
		require.Zero(t, responseFrame.GetRequestID(), msg.ClientRspHead().(*trpc.ResponseProtocol).RequestId)
	})
	t.Run("garbage data", func(t *testing.T) {
		bts := []byte("hello-world xxxxxxxxxxxx")
		fb := trpc.FramerBuilder{}
		frame := fb.New(bytes.NewReader(bts))

		_, err := frame.(codec.Decoder).Decode()
		require.Regexp(t, regexp.MustCompile(`magic .+ not match`), err.Error())
	})
	t.Run("invalid rsp type", func(t *testing.T) {
		fb := trpc.FramerBuilder{}
		frame := fb.New(nil)
		require.Contains(t, frame.(codec.Decoder).UpdateMsg("xxx", trpc.Message(context.Background())).Error(),
			"invalid rsp type")

	})
}

func mustEncode(t *testing.T, body []byte) (buffer []byte) {
	t.Helper()

	msgHead := &trpc.RequestProtocol{
		Version: uint32(trpc.TrpcProtoVersion_TRPC_PROTO_V1),
		Callee:  []byte("trpc.test.helloworld.Greeter"),
		Func:    []byte("/trpc.test.helloworld.Greeter/SayHello"),
	}
	head, err := proto.Marshal(msgHead)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	// MagicNum 0x930, 2bytes
	if err := binary.Write(buf, binary.BigEndian, uint16(trpc.TrpcMagic_TRPC_MAGIC_VALUE)); err != nil {
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
	head := &trpc.RequestProtocol{}
	err = proto.Unmarshal(reqBuf[16:], head)
	assert.Nil(t, err)
	assert.Equal(t, head.GetCallType(), uint32(codec.SendOnly))
}

func TestClientCodec_DecodeEmptyHeader(t *testing.T) {
	_, msg := codec.EnsureMessage(context.Background())
	bs, err := trpc.DefaultServerCodec.Encode(msg, nil)
	require.Nil(t, err)
	t.Logf("%x", bs)
	b, err := trpc.DefaultClientCodec.Decode(msg, bs)
	// Empty header is valid and no error is returned.
	require.Nil(t, err)
	t.Logf("%x", b)

	bs, err = trpc.DefaultClientCodec.Encode(msg, nil)
	require.Nil(t, err)
	t.Logf("%x", bs)
	b, err = trpc.DefaultServerCodec.Decode(msg, bs)
	// Empty header is valid and no error is returned.
	require.Nil(t, err)
	t.Logf("%x", b)

	fr := trpc.DefaultFramerBuilder.New(bytes.NewReader(bs))
	_, err = fr.(codec.Decoder).Decode()
	require.Nil(t, err)
}

func TestServerCodec_CallTypeDecode(t *testing.T) {
	cc := trpc.ClientCodec{}
	sc := trpc.ServerCodec{}
	msg := codec.Message(trpc.BackgroundContext())
	msg.WithCallType(codec.SendOnly)
	reqBuf, err := cc.Encode(msg, nil)
	assert.Nil(t, err)
	_, err = sc.Decode(msg, reqBuf)
	assert.Equal(t, msg.CallType(), codec.SendOnly)
}

func TestClientDecodeError(t *testing.T) {
	buf := make([]byte, 16)
	h := &trpc.FrameHead{
		FrameType: uint8(trpc.TrpcDataFrameType_TRPC_UNARY_FRAME),
		TotalLen:  uint32(len(buf)),
		HeaderLen: 0,
	}
	buf[2] = h.FrameType
	binary.BigEndian.PutUint32(buf[4:8], h.TotalLen)
	binary.BigEndian.PutUint16(buf[8:10], h.HeaderLen)
	_, msg := codec.EnsureMessage(context.Background())
	h.HeaderLen = 10
	binary.BigEndian.PutUint16(buf[8:10], h.HeaderLen+uint16(len(buf)))
	_, err := trpc.DefaultClientCodec.Decode(msg, buf)
	t.Logf("got decode err: %+v", err)
	require.NotNil(t, err)
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
		assert.Regexp(t, `.*frameSize\(\d+\) = headerSize\(\d+\) \+ bodySize\(\d+\) \+ attachmentSize\(\d+\)`+
			` is larger than MaxFrameSize\(\d+\).*`, err.Error())
	})
	t.Run("encoding attachment failed", func(t *testing.T) {
		cc := trpc.ClientCodec{}
		msg := codec.Message(trpc.BackgroundContext())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &attachment.Attachment{Request: &errorReader{}, Response: attachment.NoopAttachment{}}})
		_, err := cc.Encode(msg, nil)
		assert.ErrorContains(t, err, "reading errorReader always returns error")
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

		head := &trpc.ResponseProtocol{}
		err = proto.Unmarshal(rspBuf[16:], head)
		assert.Nil(t, err)
		assert.Equal(t, int32(errs.RetServerEncodeFail), head.GetRet())
	})
	t.Run("frame len is too large", func(t *testing.T) {
		msg := codec.Message(trpc.BackgroundContext())
		sc := trpc.ServerCodec{}
		rspBuf, err := sc.Encode(msg, make([]byte, trpc.DefaultMaxFrameSize))
		assert.Nil(t, err)

		head := &trpc.ResponseProtocol{}
		err = proto.Unmarshal(rspBuf[16:], head)
		err = proto.Unmarshal(rspBuf[16:], head)
		assert.Nil(t, err)
		assert.Equal(t, int32(errs.RetServerEncodeFail), head.GetRet())
	})
	t.Run("encoding attachment failed", func(t *testing.T) {
		msg := codec.Message(trpc.BackgroundContext())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ServerAttachmentKey{}: &attachment.Attachment{Request: attachment.NoopAttachment{}, Response: &errorReader{}}})
		sc := trpc.ServerCodec{}
		_, err := sc.Encode(msg, nil)
		assert.ErrorContains(t, err, "reading errorReader always returns error")
	})
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
