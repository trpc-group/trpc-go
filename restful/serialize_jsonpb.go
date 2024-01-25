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

package restful

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func init() {
	RegisterSerializer(&JSONPBSerializer{})
}

// JSONPBSerializer is used for content-Type: application/json.
// It's based on google.golang.org/protobuf/encoding/protojson.
//
// This serializer will firstly try jsonpb's serialization. If object does not
// conform to protobuf proto.Message interface, the serialization will switch to
// json-iterator.
type JSONPBSerializer struct {
	AllowUnmarshalNil bool // allow unmarshalling nil body
}

// JSONAPI is a copy of jsoniter.ConfigCompatibleWithStandardLibrary.
// github.com/json-iterator/go is faster than Go's standard json library.
//
// Deprecated: This global variable is exportable due to backward comparability issue but
// should not be modified. If users want to change the default behavior of
// internal JSON serialization, please use register your customized serializer
// function like:
//
//	restful.RegisterSerializer(yourOwnJSONSerializer)
var JSONAPI = jsoniter.ConfigCompatibleWithStandardLibrary

// Marshaller is a configurable protojson marshaler.
var Marshaller = protojson.MarshalOptions{EmitUnpopulated: true}

// Unmarshaller is a configurable protojson unmarshaler.
var Unmarshaller = protojson.UnmarshalOptions{DiscardUnknown: true}

// Marshal implements Serializer.
// Unlike Serializers in trpc-go/codec, Serializers in trpc-go/restful
// could be used to marshal a field of a tRPC message.
func (*JSONPBSerializer) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok { // marshal a field of a tRPC message
		return marshal(v)
	}
	// marshal tRPC message
	return Marshaller.Marshal(msg)
}

// marshal is a helper function that is used to marshal a field of a tRPC message.
func marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok { // marshal none proto field
		return marshalNonProtoField(v)
	}
	// marshal proto field
	return Marshaller.Marshal(msg)
}

// wrappedEnum is used to get the name of enum.
type wrappedEnum interface {
	protoreflect.Enum
	String() string
}

// typeOfProtoMessage is used to avoid multiple reflection and check if the object
// implements proto.Message interface.
var typeOfProtoMessage = reflect.TypeOf((*proto.Message)(nil)).Elem()

// marshalNonProtoField marshals none proto fields.
// Go's standard json lib or github.com/json-iterator/go doesn't support marshaling
// of some types of protobuf message, therefore reflection is needed to support it.
// TODO: performance optimization.
func marshalNonProtoField(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	// reflection
	rv := reflect.ValueOf(v)

	// get value to which the pointer points
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []byte("null"), nil
		}
		rv = rv.Elem()
	}

	// marshal name but value of enum
	if enum, ok := rv.Interface().(wrappedEnum); ok && !Marshaller.UseEnumNumbers {
		return JSONAPI.Marshal(enum.String())
	}
	// marshal map proto message
	if rv.Kind() == reflect.Map {
		// make map for marshalling
		m := make(map[string]*jsoniter.RawMessage)
		for _, key := range rv.MapKeys() { // range all keys
			// marshal value
			out, err := marshal(rv.MapIndex(key).Interface())
			if err != nil {
				return out, err
			}
			// assignment
			m[fmt.Sprintf("%v", key.Interface())] = (*jsoniter.RawMessage)(&out)
			if Marshaller.Indent != "" { // 指定 indent
				return JSONAPI.MarshalIndent(v, "", Marshaller.Indent)
			}
			return JSONAPI.Marshal(v)
		}
	}
	// marshal slice proto message
	if rv.Kind() == reflect.Slice {
		if rv.IsNil() { // nil slice
			if Marshaller.EmitUnpopulated {
				return []byte("[]"), nil
			}
			return []byte("null"), nil
		}

		if rv.Type().Elem().Implements(typeOfProtoMessage) { // type is proto
			var buf bytes.Buffer
			buf.WriteByte('[')
			for i := 0; i < rv.Len(); i++ { // marshal one by one
				out, err := marshal(rv.Index(i).Interface().(proto.Message))
				if err != nil {
					return nil, err
				}
				buf.Write(out)
				if i != rv.Len()-1 {
					buf.WriteByte(',')
				}
			}
			buf.WriteByte(']')
			return buf.Bytes(), nil
		}
	}

	return JSONAPI.Marshal(v)
}

// Unmarshal implements Serializer.
func (j *JSONPBSerializer) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 && j.AllowUnmarshalNil {
		return nil
	}
	msg, ok := v.(proto.Message)
	if !ok { // unmarshal a field of a tRPC message
		return unmarshal(data, v)
	}
	// unmarshal tRPC message
	return Unmarshaller.Unmarshal(data, msg)
}

// unmarshal unmarshal a field of a tRPC message.
func unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok { // unmarshal none proto fields
		return unmarshalNonProtoField(data, v)
	}
	// unmarshal proto fields
	return Unmarshaller.Unmarshal(data, msg)
}

// unmarshalNonProtoField unmarshals none proto fields.
// TODO: performance optimization.
func unmarshalNonProtoField(data []byte, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr { // Must be pointer type.
		return fmt.Errorf("%T is not a pointer", v)
	}
	// get the value to which the pointer points
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() { // New an object if nil
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		// if the object's type is proto, just unmarshal
		if msg, ok := rv.Interface().(proto.Message); ok {
			return Unmarshaller.Unmarshal(data, msg)
		}
		rv = rv.Elem()
	}
	// can only unmarshal numeric enum
	if _, ok := rv.Interface().(wrappedEnum); ok {
		var x interface{}
		if err := jsoniter.Unmarshal(data, &x); err != nil {
			return err
		}
		switch t := x.(type) {
		case float64:
			rv.Set(reflect.ValueOf(int32(t)).Convert(rv.Type()))
			return nil
		default:
			return fmt.Errorf("unmarshalling of %T into %T is not supported", t, rv.Interface())
		}
	}
	// unmarshal to slice
	if rv.Kind() == reflect.Slice {
		// unmarshal to jsoniter.RawMessage first
		var rms []jsoniter.RawMessage
		if err := JSONAPI.Unmarshal(data, &rms); err != nil {
			return err
		}
		if rms != nil { // rv MakeSlice
			rv.Set(reflect.MakeSlice(rv.Type(), 0, 0))
		}
		// unmarshal one by one
		for _, rm := range rms {
			rn := reflect.New(rv.Type().Elem())
			if err := unmarshal(rm, rn.Interface()); err != nil {
				return err
			}
			rv.Set(reflect.Append(rv, rn.Elem()))
		}
		return nil
	}
	// unmarshal to map
	if rv.Kind() == reflect.Map {
		if rv.IsNil() { // rv MakeMap
			rv.Set(reflect.MakeMap(rv.Type()))
		}
		// unmarshal to map[string]*jsoniter.RawMessage first
		m := make(map[string]*jsoniter.RawMessage)
		if err := JSONAPI.Unmarshal(data, &m); err != nil {
			return err
		}
		kind := rv.Type().Key().Kind()
		for key, value := range m { // unmarshal (k, v) one by one
			convertedKey, err := convert(key, kind) // convert key
			if err != nil {
				return err
			}
			// unmarshal value
			if value == nil {
				rm := jsoniter.RawMessage("null")
				value = &rm
			}
			rn := reflect.New(rv.Type().Elem())
			if err := unmarshal([]byte(*value), rn.Interface()); err != nil {
				return err
			}
			rv.SetMapIndex(reflect.ValueOf(convertedKey), rn.Elem())
		}
	}
	return JSONAPI.Unmarshal(data, v)
}

// convert converts map key by reflect.Kind.
func convert(key string, kind reflect.Kind) (interface{}, error) {
	switch kind {
	case reflect.String:
		return key, nil
	case reflect.Bool:
		return strconv.ParseBool(key)
	case reflect.Int32:
		v, err := strconv.ParseInt(key, 0, 32)
		if err != nil {
			return nil, err
		}
		return int32(v), nil
	case reflect.Uint32:
		v, err := strconv.ParseUint(key, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint32(v), nil
	case reflect.Int64:
		return strconv.ParseInt(key, 0, 64)
	case reflect.Uint64:
		return strconv.ParseUint(key, 0, 64)
	case reflect.Float32:
		v, err := strconv.ParseFloat(key, 32)
		if err != nil {
			return nil, err
		}
		return float32(v), nil
	case reflect.Float64:
		return strconv.ParseFloat(key, 64)
	default:
		return nil, fmt.Errorf("unsupported kind: %v", kind)
	}
}

// Name implements Serializer.
func (*JSONPBSerializer) Name() string {
	return "application/json"
}

// ContentType implements Serializer.
func (*JSONPBSerializer) ContentType() string {
	return "application/json"
}
