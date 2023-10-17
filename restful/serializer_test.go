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

package restful_test

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/testdata/restful/bookstore"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var j = &restful.JSONPBSerializer{}
var f = &restful.FormSerializer{}
var p = &restful.ProtoSerializer{}

type anonymousSerializer struct {
	restful.Serializer
}

func (anonymousSerializer) Name() string { return "" }

type mockSerializer struct {
	restful.Serializer
}

func (mockSerializer) Name() string        { return "mock" }
func (mockSerializer) ContentType() string { return "mock" }

func TestRegisterSerializer(t *testing.T) {
	for _, test := range []struct {
		serializer  restful.Serializer
		expectPanic bool
		desc        string
	}{
		{
			serializer:  nil,
			expectPanic: true,
			desc:        "register nil serializer test",
		},
		{
			serializer:  anonymousSerializer{},
			expectPanic: true,
			desc:        "register anonymous serializer test",
		},
		{
			serializer:  mockSerializer{},
			expectPanic: false,
			desc:        "register mock serializer test",
		},
	} {
		register := func() { restful.RegisterSerializer(test.serializer) }
		if test.expectPanic {
			require.Panics(t, register, test.desc)
		} else {
			require.NotPanics(t, register, test.desc)
		}
		var s restful.Serializer
		if !test.expectPanic {
			s = restful.GetSerializer(test.serializer.Name())
			require.True(t, reflect.DeepEqual(s, test.serializer), test.desc)
		}
	}
}

func TestSetDefaultSerializer(t *testing.T) {
	for _, test := range []struct {
		serializer  restful.Serializer
		expectPanic bool
		desc        string
	}{
		{
			serializer:  nil,
			expectPanic: true,
			desc:        "set nil serializer test",
		},
		{
			serializer:  anonymousSerializer{},
			expectPanic: true,
			desc:        "set anonymous serializer test",
		},
		{
			serializer:  mockSerializer{},
			expectPanic: false,
			desc:        "set mock serializer test",
		},
	} {
		register := func() { restful.SetDefaultSerializer(test.serializer) }
		if test.expectPanic {
			require.Panics(t, register, test.desc)
		} else {
			require.NotPanics(t, register, test.desc)
		}
	}
}

func TestContentType(t *testing.T) {
	require.Equal(t, "application/json", j.ContentType())
	require.Equal(t, "application/json", f.ContentType())
	require.Equal(t, "application/octet-stream", p.ContentType())
}

func TestProtoSerializer(t *testing.T) {
	input := &helloworld.HelloRequest{
		Name: "nobody",
		SingleNested: &helloworld.NestedOuter{
			Name: "anybody",
		},
		PrimitiveDoubleValue: float64(1.23),
		Time: &timestamppb.Timestamp{
			Seconds: int64(111111111),
		},
		EnumValue: helloworld.NumericEnum_ONE,
		OneofValue: &helloworld.HelloRequest_OneofString{
			OneofString: "oneof",
		},
		MappedStringValue: map[string]string{
			"foo": "bar",
		},
	}

	// marshal
	wrongMarshalObj := "foobar"
	_, err := p.Marshal(wrongMarshalObj)
	require.NotNil(t, err)

	buf, err := p.Marshal(input)
	require.Nil(t, err)

	// unmarshal
	wrongUnmarshalObj := 1
	err = p.Unmarshal(buf, &wrongUnmarshalObj)
	require.NotNil(t, err)

	output := &helloworld.HelloRequest{}
	err = p.Unmarshal(buf, output)
	require.Nil(t, err)
	require.True(t, proto.Equal(input, output))
}

func TestJSONPBSerializer(t *testing.T) {
	input := &helloworld.HelloRequest{
		Name: "nobody",
		SingleNested: &helloworld.NestedOuter{
			Name: "anybody",
		},
		PrimitiveDoubleValue: float64(1.23),
		Time: &timestamppb.Timestamp{
			Seconds: int64(111111111),
		},
		EnumValue: helloworld.NumericEnum_ONE,
		OneofValue: &helloworld.HelloRequest_OneofString{
			OneofString: "oneof",
		},
		MappedStringValue: map[string]string{
			"foo": "bar",
		},
	}
	// marshal
	buf, err := j.Marshal(input)
	require.Nil(t, err)
	// unmarshal
	output := &helloworld.HelloRequest{}
	err = j.Unmarshal(buf, output)
	require.Nil(t, err)
	require.True(t, proto.Equal(input, output))
}

func TestJSONPBSerializerMarshalField(t *testing.T) {
	for _, test := range []struct {
		input interface{}
		want  []byte
		desc  string
	}{
		{input: int32(1), want: []byte("1"), desc: "jsonpb marshal field test 01"},
		{input: proto.Int32(1), want: []byte("1"), desc: "jsonpb marshal field test 02"},
		{input: int64(1), want: []byte("1"), desc: "jsonpb marshal field test 03"},
		{input: proto.Int64(1), want: []byte("1"), desc: "jsonpb marshal field test 04"},
		{input: uint32(1), want: []byte("1"), desc: "jsonpb marshal field test 05"},
		{input: proto.Uint32(1), want: []byte("1"), desc: "jsonpb marshal field test 06"},
		{input: uint64(1), want: []byte("1"), desc: "jsonpb marshal field test 07"},
		{input: proto.Uint64(1), want: []byte("1"), desc: "jsonpb marshal field test 08"},
		{input: "abc", want: []byte(`"abc"`), desc: "jsonpb marshal field test 09"},
		{input: proto.String("abc"), want: []byte(`"abc"`), desc: "jsonpb marshal field test 10"},
		{input: float32(1.5), want: []byte(`1.5`), desc: "jsonpb marshal field test 11"},
		{input: proto.Float32(1.5), want: []byte(`1.5`), desc: "jsonpb marshal field test 12"},
		{input: float64(1.5), want: []byte(`1.5`), desc: "jsonpb marshal field test 13"},
		{input: proto.Float64(1.5), want: []byte(`1.5`), desc: "jsonpb marshal field test 14"},
		{input: true, want: []byte("true"), desc: "jsonpb marshal field test 15"},
		{input: false, want: []byte("false"), desc: "jsonpb marshal field test 16"},
		{input: (*string)(nil), want: []byte("null"), desc: "jsonpb marshal field test 17"},
		{
			input: helloworld.NumericEnum_ONE,
			want:  []byte(`"ONE"`),
			desc:  "jsonpb marshal field test 18",
		},
		{
			input: (*helloworld.NumericEnum)(proto.Int32(int32(helloworld.NumericEnum_ONE))),
			want:  []byte(`"ONE"`),
			desc:  "jsonpb marshal field test 19",
		},
		{
			input: map[string]int32{
				"foo": 1,
			},
			want: []byte(`{"foo":1}`),
			desc: "jsonpb marshal field test 20",
		},
		{
			input: map[string]*bookstore.Book{
				"foo": {Id: 123},
			},
			want: []byte(`{"foo":{"id":123}}`),
			desc: "jsonpb marshal field test 21",
		},
		{
			input: map[int32]*bookstore.Book{
				1: {Id: 123},
			},
			want: []byte(`{"1":{"id":123}}`),
			desc: "jsonpb marshal field test 22",
		},
		{
			input: map[bool]*bookstore.Book{
				true: {Id: 123},
			},
			want: []byte(`{"true":{"id":123}}`),
			desc: "jsonpb marshal field test 23",
		},
		{
			input: &durationpb.Duration{
				Seconds: 123,
				Nanos:   456000000,
			},
			want: []byte(`"123.456s"`),
			desc: "jsonpb marshal field test 24",
		},
		{
			input: &timestamppb.Timestamp{
				Seconds: 1462875553,
				Nanos:   123000000,
			},
			want: []byte(`"2016-05-10T10:19:13.123Z"`),
			desc: "jsonpb marshal field test 25",
		},
		{
			input: new(emptypb.Empty),
			want:  []byte("{}"),
			desc:  "jsonpb marshal field test 26",
		},
		{
			input: &structpb.Value{
				Kind: new(structpb.Value_NullValue),
			},
			want: []byte("null"),
			desc: "jsonpb marshal field test 27",
		},
		{
			input: &structpb.Value{
				Kind: &structpb.Value_NumberValue{
					NumberValue: 123.4,
				},
			},
			want: []byte("123.4"),
			desc: "jsonpb marshal field test 28",
		},
		{
			input: &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: "abc",
				},
			},
			want: []byte(`"abc"`),
			desc: "jsonpb marshal field test 29",
		},
		{
			input: &structpb.Value{
				Kind: &structpb.Value_BoolValue{
					BoolValue: true,
				},
			},
			want: []byte("true"),
			desc: "jsonpb marshal field test 30",
		},
		{
			input: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"foo_bar": {
						Kind: &structpb.Value_BoolValue{
							BoolValue: true,
						},
					},
				},
			},
			want: []byte(`{"foo_bar":true}`),
			desc: "jsonpb marshal field test 31",
		},

		{
			input: &wrapperspb.BoolValue{Value: true},
			want:  []byte("true"),
			desc:  "jsonpb marshal field test 32",
		},
		{
			input: &wrapperspb.DoubleValue{Value: 123.456},
			want:  []byte("123.456"),
			desc:  "jsonpb marshal field test 33",
		},
		{
			input: &wrapperspb.FloatValue{Value: 123.456},
			want:  []byte("123.456"),
			desc:  "jsonpb marshal field test 34",
		},
		{
			input: &wrapperspb.Int32Value{Value: -123},
			want:  []byte("-123"),
			desc:  "jsonpb marshal field test 35",
		},
		{
			input: &wrapperspb.Int64Value{Value: -123},
			want:  []byte(`"-123"`),
			desc:  "jsonpb marshal field test 36",
		},
		{
			input: &wrapperspb.UInt32Value{Value: 123},
			want:  []byte("123"),
			desc:  "jsonpb marshal field test 37",
		},
		{
			input: &wrapperspb.UInt64Value{Value: 123},
			want:  []byte(`"123"`),
			desc:  "jsonpb marshal field test 38",
		},
		{
			input: []**bookstore.Book{},
			want:  []byte("[]"),
			desc:  "jsonpb marshal field test 39",
		},
		{
			input: []*bookstore.Book{{Id: 1}},
			want:  []byte(`[{"id":"1","author":"","title":"","content":null}]`),
			desc:  "jsonpb marshal field test 40",
		},
		{
			input: nil,
			want:  []byte("null"),
			desc:  "jsonpb marshal field test 41",
		},
	} {
		got, err := j.Marshal(test.input)
		require.Nil(t, err, test.desc)
		s := strings.Replace(string(got), " ", "", -1)
		s = strings.Replace(s, "\n", "", -1)
		s = strings.Replace(s, "\t", "", -1)
		require.Equal(t, test.want, []byte(s), test.desc)
	}
}

func TestJSONPBSerializerUnmarshalField(t *testing.T) {
	for _, test := range []struct {
		input []byte
		want  interface{}
		desc  string
	}{
		{input: []byte("1"), want: int32(1), desc: "jsonpb unmarshal field test 01"},
		{input: []byte("1"), want: proto.Int32(1), desc: "jsonpb unmarshal field test 02"},
		{input: []byte("1"), want: int64(1), desc: "jsonpb unmarshal field test 03"},
		{input: []byte("1"), want: proto.Int64(1), desc: "jsonpb unmarshal field test 04"},
		{input: []byte("1"), want: uint32(1), desc: "jsonpb unmarshal field test 05"},
		{input: []byte("1"), want: proto.Uint32(1), desc: "jsonpb unmarshal field test 06"},
		{input: []byte("1"), want: uint64(1), desc: "jsonpb unmarshal field test 07"},
		{input: []byte("1"), want: proto.Uint64(1), desc: "jsonpb unmarshal field test 08"},
		{input: []byte(`"abc"`), want: "abc", desc: "jsonpb unmarshal field test 09"},
		{input: []byte(`"abc"`), want: proto.String("abc"), desc: "jsonpb unmarshal field test 10"},
		{input: []byte(`1.5`), want: float32(1.5), desc: "jsonpb unmarshal field test 11"},
		{input: []byte(`1.5`), want: proto.Float32(1.5), desc: "jsonpb unmarshal field test 12"},
		{input: []byte(`1.5`), want: float64(1.5), desc: "jsonpb unmarshal field test 13"},
		{input: []byte(`1.5`), want: proto.Float64(1.5), desc: "jsonpb unmarshal field test 14"},
		{input: []byte("true"), want: true, desc: "jsonpb unmarshal field test 15"},
		{input: []byte("false"), want: false, desc: "jsonpb unmarshal field test 16"},
		{input: []byte("null"), want: (*string)(nil), desc: "jsonpb unmarshal field test 17"},
		{
			input: []byte("1"),
			want:  helloworld.NumericEnum_ONE,
			desc:  "jsonpb unmarshal field test 18",
		},
		{
			input: []byte("1"),
			want:  (*helloworld.NumericEnum)(proto.Int32(int32(helloworld.NumericEnum_ONE))),
			desc:  "jsonpb unmarshal field test 19",
		},
		{
			input: []byte(`{"foo":1}`),
			want: map[string]int32{
				"foo": 1,
			},
			desc: "jsonpb unmarshal field test 20",
		},
		{
			input: []byte(`{"foo":{"id":123}}`),
			want: map[string]*bookstore.Book{
				"foo": {Id: 123},
			},
			desc: "jsonpb unmarshal field test 21",
		},
		{
			input: []byte(`{"1":{"id":123}}`),
			want: map[int32]*bookstore.Book{
				1: {Id: 123},
			},
			desc: "jsonpb unmarshal field test 22",
		},
		{
			input: []byte(`{"true":{"id":123}}`),
			want: map[bool]*bookstore.Book{
				true: {Id: 123},
			},
			desc: "jsonpb unmarshal field test 23",
		},
		{
			input: []byte(`"123.456s"`),
			want: &durationpb.Duration{
				Seconds: 123,
				Nanos:   456000000,
			},
			desc: "jsonpb unmarshal field test 24",
		},
		{
			input: []byte(`"2016-05-10T10:19:13.123Z"`),
			want: &timestamppb.Timestamp{
				Seconds: 1462875553,
				Nanos:   123000000,
			},
			desc: "jsonpb unmarshal field test 25",
		},
		{
			input: []byte("{}"),
			want:  new(emptypb.Empty),
			desc:  "jsonpb unmarshal field test 26",
		},
		{
			input: []byte("null"),
			want: &structpb.Value{
				Kind: new(structpb.Value_NullValue),
			},
			desc: "jsonpb unmarshal field test 27",
		},
		{
			input: []byte("123.4"),
			want: &structpb.Value{
				Kind: &structpb.Value_NumberValue{
					NumberValue: 123.4,
				},
			},
			desc: "jsonpb unmarshal field test 28",
		},
		{
			input: []byte(`"abc"`),
			want: &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: "abc",
				},
			},
			desc: "jsonpb unmarshal field test 29",
		},
		{
			input: []byte("true"),
			want: &structpb.Value{
				Kind: &structpb.Value_BoolValue{
					BoolValue: true,
				},
			},
			desc: "jsonpb unmarshal field test 30",
		},
		{
			input: []byte(`{"foo_bar":true}`),
			want: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"foo_bar": {
						Kind: &structpb.Value_BoolValue{
							BoolValue: true,
						},
					},
				},
			},
			desc: "jsonpb unmarshal field test 31",
		},
		{
			input: []byte("true"),
			want:  &wrapperspb.BoolValue{Value: true},
			desc:  "jsonpb unmarshal field test 32",
		},
		{
			input: []byte("123.456"),
			want:  &wrapperspb.DoubleValue{Value: 123.456},
			desc:  "jsonpb unmarshal field test 33",
		},
		{
			input: []byte("123.456"),
			want:  &wrapperspb.FloatValue{Value: 123.456},
			desc:  "jsonpb unmarshal field test 34",
		},
		{
			input: []byte("-123"),
			want:  &wrapperspb.Int32Value{Value: -123},
			desc:  "jsonpb unmarshal field test 35",
		},
		{
			input: []byte(`"-123"`),
			want:  &wrapperspb.Int64Value{Value: -123},
			desc:  "jsonpb unmarshal field test 36",
		},
		{
			input: []byte("123"),
			want:  &wrapperspb.UInt32Value{Value: 123},
			desc:  "jsonpb unmarshal field test 37",
		},
		{
			input: []byte(`"123"`),
			want:  &wrapperspb.UInt64Value{Value: 123},
			desc:  "jsonpb unmarshal field test 38",
		},
		{
			input: []byte(`{"1":"foo"}`),
			want: map[uint32]string{
				1: "foo",
			},
			desc: "jsonpb unmarshal field test 39",
		},
		{
			input: []byte(`{"1":"foo"}`),
			want: map[int64]string{
				1: "foo",
			},
			desc: "jsonpb unmarshal field test 40",
		},
		{
			input: []byte(`{"1":"foo"}`),
			want: map[uint64]string{
				1: "foo",
			},
			desc: "jsonpb unmarshal field test 41",
		},
		{
			input: []byte(`{"1":"foo"}`),
			want: map[float32]string{
				1: "foo",
			},
			desc: "jsonpb unmarshal field test 42",
		},
		{
			input: []byte(`{"1":"foo"}`),
			want: map[float64]string{
				1: "foo",
			},
			desc: "jsonpb unmarshal field test 43",
		},
		{
			input: []byte(`{"1":null}`),
			want: map[float64]*string{
				1: nil,
			},
			desc: "jsonpb unmarshal field test 44",
		},
		{
			input: []byte(`[{"id":"1"},{"id":"2"}]`),
			want: []*bookstore.Book{
				{Id: 1}, {Id: 2},
			},
			desc: "jsonpb unmarshal field test 45",
		},
		{
			input: []byte(`[{"a":true},{"b":true}]`),
			want: []helloworld.NestedInner{
				{A: true}, {B: true},
			},
			desc: "jsonpb unmarshal field test 46",
		},
	} {
		rflValue := reflect.New(reflect.TypeOf(test.want))
		err := j.Unmarshal(test.input, rflValue.Interface())
		require.Nil(t, err, test.desc)
		require.Equal(t, "", cmp.Diff(rflValue.Elem().Interface(), test.want, protocmp.Transform()), test.desc)
	}
}

func TestFormSerializerMarshal(t *testing.T) {
	for _, test := range []struct {
		input           interface{}
		emitUnpopupated bool
		expect          string
		desc            string
	}{
		{
			input: &helloworld.HelloRequest{
				Name: "nobody",
				SingleNested: &helloworld.NestedOuter{
					Name: "anybody",
				},
				PrimitiveDoubleValue: float64(1.23),
				Time: &timestamppb.Timestamp{
					Seconds: int64(111111111),
				},
				EnumValue: helloworld.NumericEnum_ONE,
				OneofValue: &helloworld.HelloRequest_OneofString{
					OneofString: "oneof",
				},
				MappedStringValue: map[string]string{
					"foo": "bar",
				},
			},
			emitUnpopupated: false,
			expect: `{
						"name":"nobody", 
						"singleNested":{"name":"anybody"}, 
						"primitiveDoubleValue":1.23, 
						"enumValue":"ONE", 
						"oneofString":"oneof", 
						"mappedStringValue":{"foo":"bar"}, 
						"time":"1973-07-10T00:11:51Z"
					}`,
			desc: "form serializer marshal proto message test ",
		},
		{
			input:           helloworld.NumericEnum_ONE,
			emitUnpopupated: true,
			expect:          `"ONE"`,
			desc:            "form serializer marshal non proto field test",
		},
	} {
		restful.Marshaller.EmitUnpopulated = test.emitUnpopupated
		buf, err := f.Marshal(test.input)
		require.Nil(t, err)
		want := strings.Replace(test.expect, " ", "", -1)
		want = strings.Replace(want, "\n", "", -1)
		want = strings.Replace(want, "\t", "", -1)
		got := strings.Replace(string(buf), " ", "", -1)
		got = strings.Replace(got, "\n", "", -1)
		got = strings.Replace(got, "\t", "", -1)
		require.Equal(t, want, got, test.desc)
	}
}

func TestFormSerializerUnmarshal(t *testing.T) {
	for _, test := range []struct {
		data         []byte
		unmarshalObj interface{}
		want         interface{}
		wantErr      bool
		desc         string
	}{
		{
			data: []byte(url.Values{
				"name":                   []string{"nobody"},
				"single_nested.name":     []string{"anybody"},
				"primitive_bool_value":   []string{"true"},
				"primitive_int32_value":  []string{"1"},
				"primitive_uint32_value": []string{"1"},
				"primitive_int64_value":  []string{"1"},
				"primitive_uint64_value": []string{"1"},
				"primitive_float_value":  []string{"1"},
				"primitive_double_value": []string{"1.23"},
				"primitive_bytes_value":  []string{""},
				"time":                   []string{"1970-01-01T00:00:01Z"},
				"duration":               []string{"0"},
				"wrapped_bool_value":     []string{"true"},
				"wrapped_int32_value":    []string{"1"},
				"wrapped_uint32_value":   []string{"1"},
				"wrapped_int64_value":    []string{"1"},
				"wrapped_uint64_value":   []string{"1"},
				"wrapped_float_value":    []string{"1"},
				"wrapped_double_value":   []string{"1"},
				"wrapped_str_value":      []string{"foo"},
				"wrapped_bytes_value":    []string{""},
				"enum_value":             []string{"1"},
				"oneof_string":           []string{"oneof"},
				"mapped_string_value":    []string{"foo", "bar"},
				"repeated_string_value":  []string{"foobar", "foo", "bar", "baz"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want: &helloworld.HelloRequest{
				Name: "nobody",
				SingleNested: &helloworld.NestedOuter{
					Name: "anybody",
				},
				PrimitiveBoolValue:   true,
				PrimitiveInt32Value:  1,
				PrimitiveUint32Value: 1,
				PrimitiveInt64Value:  1,
				PrimitiveUint64Value: 1,
				PrimitiveFloatValue:  1,
				PrimitiveDoubleValue: float64(1.23),
				PrimitiveBytesValue:  []byte(""),
				Time: &timestamppb.Timestamp{
					Seconds: int64(1),
				},
				Duration:           &durationpb.Duration{},
				WrappedBoolValue:   &wrapperspb.BoolValue{Value: true},
				WrappedFloatValue:  &wrapperspb.FloatValue{Value: 1},
				WrappedStrValue:    &wrapperspb.StringValue{Value: "foo"},
				WrappedBytesValue:  &wrapperspb.BytesValue{},
				WrappedInt32Value:  &wrapperspb.Int32Value{Value: 1},
				WrappedUint32Value: &wrapperspb.UInt32Value{Value: 1},
				WrappedDoubleValue: &wrapperspb.DoubleValue{Value: 1},
				WrappedInt64Value:  &wrapperspb.Int64Value{Value: 1},
				WrappedUint64Value: &wrapperspb.UInt64Value{Value: 1},
				EnumValue:          helloworld.NumericEnum_ONE,
				OneofValue: &helloworld.HelloRequest_OneofString{
					OneofString: "oneof",
				},
				RepeatedStringValue: []string{
					"foobar", "foo", "bar", "baz",
				},
				MappedStringValue: map[string]string{
					"foo": "bar",
				},
			},
			wantErr: false,
			desc:    "form serializer unmarshal to proto message test",
		},
		{
			data:         []byte("!@#$%^&"),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal invalid data test",
		},
		{
			data: []byte(url.Values{
				"primitive_string_value": []string{"foo", "bar"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal invalid len of values test",
		},
		{
			data: []byte(url.Values{
				"primitive_int32_value": []string{"foo"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal invalid field data test",
		},
		{
			data: []byte(url.Values{
				"mapped_string_value": []string{"foo", "bar", "baz"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal map invalid len of values test",
		},
		{
			data: []byte(url.Values{
				"mapped_enum_value": []string{"foo", "bar"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal map invalid value test",
		},
		{
			data: []byte(url.Values{
				"repeated_enum_value": []string{"foo", "bar"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal repeated test",
		},
		{
			data: []byte(url.Values{
				"wrapped_bool_value": []string{"foo"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal google.protobuf.bool field data test",
		},
		{
			data: []byte(url.Values{
				"wrapped_int32_value": []string{"foo"},
			}.Encode()),
			unmarshalObj: &helloworld.HelloRequest{},
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal google.protobuf.int32 field data test",
		},
		{
			data: []byte(url.Values{
				"foo": []string{"bar"},
			}.Encode()),
			unmarshalObj: 1,
			want:         nil,
			wantErr:      true,
			desc:         "form serializer unmarshal to wrong obj test",
		},
	} {
		err := f.Unmarshal(test.data, test.unmarshalObj)
		if test.wantErr {
			require.NotNil(t, err, test.desc)
		} else {
			require.Nil(t, err, test.desc)
			require.Equal(t, "", cmp.Diff(test.unmarshalObj, test.want, protocmp.Transform()), test.desc)
		}
	}
}

func TestJSONPBSerializerMarshalOptions(t *testing.T) {
	for _, test := range []struct {
		input   interface{}
		option  func(*protojson.MarshalOptions)
		want    []byte
		wantErr bool
		desc    string
	}{
		{
			input: ([]*helloworld.HelloRequest)(nil),
			option: func(o *protojson.MarshalOptions) {
				o.EmitUnpopulated = true
			},
			want:    []byte("[]"),
			wantErr: false,
			desc:    "jsonpb marshal options test 01",
		},
		{
			input: map[string]string{"foo": "bar"},
			option: func(o *protojson.MarshalOptions) {
				o.Indent = "  "
			},
			want: []byte(`{
  "foo": "bar"
}`),
			wantErr: false,
			desc:    "jsonpb marshal options test 02",
		},
	} {
		test.option(&restful.Marshaller)
		got, err := j.Marshal(test.input)
		if test.wantErr {
			require.NotNil(t, err, test.desc)
		} else {
			require.Nil(t, err, test.desc)
			require.Equal(t, got, test.want, test.desc)
		}
	}
}

func TestJSONPBAllowUnmarshalNil(t *testing.T) {
	var req helloworld.HelloRequest
	j.AllowUnmarshalNil = true
	err := j.Unmarshal([]byte{}, &req)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(&req, &helloworld.HelloRequest{}))
}
