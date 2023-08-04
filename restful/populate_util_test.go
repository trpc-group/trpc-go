package restful_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
)

func TestPopulateMessage(t *testing.T) {
	for _, test := range []struct {
		msg       proto.Message
		fieldPath []string
		values    []string
		want      proto.Message
		wantErr   bool
		desc      string
	}{
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: nil,
			values:    nil,
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test empty fieldpath or fieldvalues",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"foobar"},
			values:    []string{"anything"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test field name not found",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"repeatedEnumValue", "x"},
			values:    []string{"anything"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test wrong fieldpath",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"name"},
			values:    []string{"name1", "name2"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test populate field with multi values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"mapped_string_value"},
			values:    []string{"key", "value", "anything"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test populate map field with wrong len of values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"mapped_enum_value"},
			values:    []string{"key", "value"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test populate map field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"enum_value"},
			values:    []string{"11"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse enum field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"enum_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse enum field with non numeric values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_bool_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse bool type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_int32_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse int32 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_int64_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse int64 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_uint32_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse uint32 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_uint64_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse uint64 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_float_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse float type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_double_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse double type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"primitive_bytes_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse bytes type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"mask_value"},
			values:    []string{"a.b.c,d.e.f"},
			want: &helloworld.HelloRequest{
				MaskValue: &fieldmaskpb.FieldMask{Paths: []string{"a.b.c", "d.e.f"}},
			},
			wantErr: false,
			desc:    "populating message test parse fieldmask type field",
		},
	} {
		err := restful.PopulateMessage(test.msg, test.fieldPath, test.values)
		if test.wantErr {
			require.NotNil(t, err, test.desc)
		} else {
			require.Nil(t, err, test.desc)
			require.Equal(t, "", cmp.Diff(test.want, test.msg, protocmp.Transform()), test.desc)
		}
	}
}

func TestPopulateMessageWrappedField(t *testing.T) {
	for _, test := range []struct {
		msg       proto.Message
		fieldPath []string
		values    []string
		want      proto.Message
		wantErr   bool
		desc      string
	}{
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_bool_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped bool type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_float_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped float type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_double_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped double type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_int32_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped int32 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_int64_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped int64 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_uint32_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped uint32 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_uint64_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped uint64 type field with invalid values",
		},
		{
			msg:       &helloworld.HelloRequest{},
			fieldPath: []string{"wrapped_bytes_value"},
			values:    []string{"foo"},
			want:      &helloworld.HelloRequest{},
			wantErr:   true,
			desc:      "populating message test parse wrapped bytes type field with invalid values",
		},
	} {
		err := restful.PopulateMessage(test.msg, test.fieldPath, test.values)
		if test.wantErr {
			require.NotNil(t, err, test.desc)
		} else {
			require.Nil(t, err, test.desc)
			require.True(t, reflect.DeepEqual(test.want, test.msg), test.desc)
		}
	}
}

func TestPopulateMessageTraverseNotFound(t *testing.T) {
	msg := &helloworld.HelloRequest{}
	fieldPath := []string{"foobar"}
	values := []string{"anything"}
	err := restful.PopulateMessage(msg, fieldPath, values)
	require.True(t, errors.Is(err, restful.ErrTraverseNotFound))
}
