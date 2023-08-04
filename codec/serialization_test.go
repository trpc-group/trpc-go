package codec_test

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/jce"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	fb "trpc.group/trpc-go/trpc-go/testdata/fbstest"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestSerialization(t *testing.T) {
	noop := &codec.NoopSerialization{}
	body := &codec.Body{Data: []byte("Serializer Body")}

	str := body.String()
	assert.Equal(t, "[83 101 114 105 97 108 105 122 101 114 32 66 111 100 121]", str)

	invalidBodyType := []byte{}
	_, err := noop.Marshal(invalidBodyType)
	assert.NotNil(t, err)

	err = noop.Unmarshal(body.Data, invalidBodyType)
	assert.NotNil(t, err)

	codec.RegisterSerializer(codec.SerializationTypeNoop, noop)

	s := codec.GetSerializer(-1)
	assert.Nil(t, s)

	s = codec.GetSerializer(codec.SerializationTypeNoop)
	assert.Equal(t, noop, s)

	data, err := codec.Marshal(codec.SerializationTypeNoop, body)
	assert.Nil(t, err)
	assert.Equal(t, body.Data, data)

	err = codec.Unmarshal(codec.SerializationTypeNoop, []byte("Serializer Unmarshal Body"), body)
	assert.Nil(t, err)
	assert.Equal(t, []byte("Serializer Unmarshal Body"), body.Data)

	data, err = codec.Marshal(codec.SerializationTypePB, body)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	err = codec.Unmarshal(codec.SerializationTypePB, []byte("Serializer Unmarshal Body"), body)
	assert.NotNil(t, err)

	data, err = codec.Marshal(codec.SerializationTypeFlatBuffer, body)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	err = codec.Unmarshal(codec.SerializationTypeFlatBuffer, []byte("Serializer Unmarshal Body"), body)
	assert.NotNil(t, err)

	data, err = codec.Marshal(codec.SerializationTypeUnsupported, body)
	assert.Nil(t, err)
	assert.Nil(t, data)

	err = codec.Unmarshal(codec.SerializationTypeUnsupported, []byte("Serializer Unmarshal Body"), body)
	assert.Nil(t, err)

	empty := []byte{}
	emptyBody := (*codec.Body)(nil)
	err = codec.Unmarshal(codec.SerializationTypeNoop, empty, body)
	assert.Nil(t, err)

	err = codec.Unmarshal(codec.SerializationTypeNoop, empty, emptyBody)
	assert.Nil(t, err)

	err = codec.Unmarshal(codec.SerializationTypeNoop, []byte("Serializer Unmarshal Body"), emptyBody)
	assert.NotNil(t, err)

	data, err = codec.Marshal(codec.SerializationTypeNoop, emptyBody)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	data, err = codec.Marshal(codec.SerializationTypePB, nil)
	assert.Nil(t, err)
	assert.Nil(t, data)

	data, err = codec.Marshal(codec.SerializationTypeUnsupported, body)
	assert.Nil(t, err)
	assert.Nil(t, data)

	err = codec.Unmarshal(codec.SerializationTypeUnsupported, nil, body)
	assert.Nil(t, err)

	err = codec.Unmarshal(codec.SerializationTypeUnsupported, nil, nil)
	assert.Nil(t, err)

	err = codec.Unmarshal(codec.SerializationTypeUnsupported, []byte{1, 2}, body)
	assert.Nil(t, err)

	data, err = codec.Marshal(100009, body)
	assert.NotNil(t, err)

	err = codec.Unmarshal(100009, []byte{1, 2}, body)
	assert.NotNil(t, err)
}

func TestJson(t *testing.T) {
	type Data struct {
		A int
		B string
	}
	s := &codec.JSONSerialization{}
	body := []byte("{\"A\":1,\"B\":\"bb\"}")
	data := &Data{}

	err := s.Unmarshal(body, data)
	assert.Nil(t, err)
	assert.Equal(t, 1, data.A)
	assert.Equal(t, "bb", data.B)

	bytes, err := s.Marshal(data)
	assert.Nil(t, err)
	assert.Equal(t, body, bytes)

	// json-iterator issue https://github.com/golang/go/issues/48238#issuecomment-917321536
	m := make(map[string]string)
	m["a"] = "hello"
	bytes, err = s.Marshal(m)
	body = []byte("{\"a\":\"hello\"}")
	assert.Nil(t, err)
	assert.Equal(t, body, bytes)
}

func TestJsonPB(t *testing.T) {
	s := &codec.JSONPBSerialization{}
	body := []byte("{\"msg\":\"utTest\"}")
	data := &pb.HelloReply{}

	err := s.Unmarshal(body, data)
	assert.Nil(t, err)
	assert.Equal(t, "utTest", data.Msg)

	bytes, err := s.Marshal(data)
	assert.Nil(t, err)
	assert.Equal(t, body, bytes)
}

func TestJsonPBNotImplProto(t *testing.T) {
	type Data struct {
		A int
		B string
	}
	s := &codec.JSONPBSerialization{}
	data := &Data{A: 1, B: "test"}

	bytes, err := s.Marshal(data)
	assert.Nil(t, err)

	var data1 Data
	err = s.Unmarshal(bytes, &data1)
	assert.Nil(t, err)
	assert.Equal(t, data.A, data1.A)
	assert.Equal(t, data.B, data1.B)
}

func TestProto(t *testing.T) {
	p := &trpcpb.RequestProtocol{
		Version: 1,
		Func:    []byte("/trpc.test.helloworld.Greeter/SayHello"),
	}

	s := &codec.PBSerialization{}
	data, err := s.Marshal(p)
	assert.Nil(t, err)
	p1 := &trpcpb.RequestProtocol{}

	err = s.Unmarshal(data, p1)
	assert.Nil(t, err)
	assert.Equal(t, p.Version, p1.Version)
}

func TestFlatbuffers(t *testing.T) {
	s := &codec.FBSerialization{}
	b := flatbuffers.NewBuilder(0)
	i := b.CreateString("this is a string")
	fb.HelloReqStart(b)
	fb.HelloReqAddMessage(b, i)
	b.Finish(fb.HelloReqEnd(b))

	data, err := s.Marshal(b)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	req := &fb.HelloReq{}
	err = s.Unmarshal(data, req)
	assert.Nil(t, err)
	assert.Equal(t, "this is a string", string(req.Message()))
}

// GetReq struct implement
// GetReq is code generate by
// [trpc4videopacket]
// source jce content:
//
// module Hello
//
//	{
//	   struct GetReq {
//	      0 optional int a;
//	      1 optional int b;
//	   };
//	}
type GetReq struct {
	A int32 `json:"a"`
	B int32 `json:"b"`
}

func (st *GetReq) ResetDefault() {
}

// ReadFrom reads  from _is and put into struct.
func (st *GetReq) ReadFrom(_is *jce.Reader) error {
	var err error
	var length int32
	var have bool
	var ty byte
	st.ResetDefault()

	err = _is.Read_int32(&st.A, 0, false)
	if err != nil {
		return err
	}

	err = _is.Read_int32(&st.B, 1, false)
	if err != nil {
		return err
	}

	_ = err
	_ = length
	_ = have
	_ = ty
	return nil
}

// WriteTo encode struct to buffer
func (st *GetReq) WriteTo(_os *jce.Buffer) error {
	var err error
	_ = err
	err = _os.Write_int32(st.A, 0)
	if err != nil {
		return err
	}

	err = _os.Write_int32(st.B, 1)
	if err != nil {
		return err
	}
	return nil
}

type GetReqNotJce struct {
	A int32 `json:"a"`
	B int32 `json:"b"`
}

func TestJCE(t *testing.T) {
	s := codec.GetSerializer(codec.SerializationTypeJCE)

	// 异常用例
	p1 := &GetReqNotJce{A: 100, B: 1000}
	data, err := s.Marshal(p1)
	assert.Nil(t, data)

	p2 := &GetReqNotJce{}
	err = s.Unmarshal(data, p2)
	assert.NotNil(t, err)

	// 正常用例
	p3 := &GetReq{A: 100, B: 1000}
	data, err = s.Marshal(p3)
	assert.Nil(t, err)

	p4 := &GetReq{}
	err = s.Unmarshal(data, p4)
	assert.Nil(t, err)
	assert.Equal(t, p3.A, p4.A)
	assert.Equal(t, p3.B, p4.B)
}

func TestXML(t *testing.T) {
	type Data struct {
		A int
		B string
	}
	var tests = []struct {
		In Data
	}{
		{In: Data{1, "1"}},
		{In: Data{2, "2"}},
	}

	for _, tt := range tests {
		buf, err := codec.Marshal(codec.SerializationTypeXML, tt.In)
		assert.Nil(t, err)

		got := &Data{}
		err = codec.Unmarshal(codec.SerializationTypeXML, buf, got)
		assert.Nil(t, err)

		assert.Equal(t, tt.In.A, got.A)
		assert.Equal(t, tt.In.B, got.B)
	}
}
