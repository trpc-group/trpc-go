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
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	// ErrTraverseNotFound is the error which indicates the field is
	// not found after traversing the proto message.
	ErrTraverseNotFound = errors.New("field not found")
)

// PopulateMessage populates a proto message.
func PopulateMessage(msg proto.Message, fieldPath []string, values []string) error {
	// empty check
	if len(fieldPath) == 0 || len(values) == 0 {
		return fmt.Errorf("fieldPath: %v or values: %v is empty", fieldPath, values)
	}

	// proto reflection
	message := msg.ProtoReflect()

	// traverse for leaf field by field path
	message, fd, err := traverse(message, fieldPath)
	if err != nil {
		return fmt.Errorf("failed to traverse for leaf field by fieldPath %v: %w", fieldPath, err)
	}

	// populate the field
	switch {
	case fd.IsList(): // repeated field
		return populateRepeatedField(fd, message.Mutable(fd).List(), values)
	case fd.IsMap(): // map field
		return populateMapField(fd, message.Mutable(fd).Map(), values)
	default: // normal field
		return populateField(fd, message, values)
	}
}

// fdByName returns field descriptor by field name.
func fdByName(message protoreflect.Message, name string) (protoreflect.FieldDescriptor, error) {
	if message == nil {
		return nil, errors.New("get field descriptor from nil message")
	}

	field := message.Descriptor().Fields().ByJSONName(name)
	if field == nil {
		field = message.Descriptor().Fields().ByName(protoreflect.Name(name))
	}
	if field == nil {
		return nil, fmt.Errorf("%w: %v", ErrTraverseNotFound, name)
	}
	return field, nil
}

// traverse traverses the nested proto message by names and returns the descriptor of the leaf field.
func traverse(
	message protoreflect.Message,
	fieldPath []string,
) (protoreflect.Message, protoreflect.FieldDescriptor, error) {
	field, err := fdByName(message, fieldPath[0])
	if err != nil {
		return nil, nil, err
	}

	// leaf field
	if len(fieldPath) == 1 {
		return message, field, nil
	}

	// haven't reached the leaf field, need to continue traversing,
	// and type of current field must be proto message
	if field.Message() == nil || field.Cardinality() == protoreflect.Repeated {
		return nil, nil, fmt.Errorf("type of field %s is not proto message", fieldPath[0])
	}

	// recursion
	return traverse(message.Mutable(field).Message(), fieldPath[1:])
}

// populateField populates normal fields.
func populateField(fd protoreflect.FieldDescriptor, msg protoreflect.Message, values []string) error {
	// len of values should be 1
	if len(values) != 1 {
		return fmt.Errorf("tried to populate field %s with values %v", fd.FullName().Name(), values)
	}

	// parse value into protoreflect.Value
	v, err := parseField(fd, values[0])
	if err != nil {
		return fmt.Errorf("failed to parse field %s: %w", fd.FullName().Name(), err)
	}

	// do the population
	msg.Set(fd, v)
	return nil
}

// populateRepeatedField populates repeated fields.
func populateRepeatedField(fd protoreflect.FieldDescriptor, list protoreflect.List, values []string) error {
	for _, value := range values {
		// parse value into protoreflect.Value
		v, err := parseField(fd, value)
		if err != nil {
			return fmt.Errorf("failed to parse repeated field %s: %w", fd.FullName().Name(), err)
		}
		// do the population
		list.Append(v)
	}
	return nil
}

// populateMapField populates map fields.
func populateMapField(fd protoreflect.FieldDescriptor, m protoreflect.Map, values []string) error {
	// len of values should be 2
	if len(values) != 2 {
		return fmt.Errorf("tried to populate map field %s with values %v", fd.FullName().Name(), values)
	}

	// parse map key into protoreflect.Value
	key, err := parseField(fd.MapKey(), values[0])
	if err != nil {
		return fmt.Errorf("failed to parse key of map field %s: %w", fd.FullName().Name(), err)
	}

	// parse map value into protoreflect.Value
	value, err := parseField(fd.MapValue(), values[1])
	if err != nil {
		return fmt.Errorf("failed to parse value of map field %s: %w", fd.FullName().Name(), err)
	}

	// do the population
	m.Set(key.MapKey(), value)
	return nil
}

// parseField parses string value into protoreflect.Value by protoreflect.FieldDescriptor.
func parseField(fd protoreflect.FieldDescriptor, value string) (protoreflect.Value, error) {
	switch kind := fd.Kind(); kind {
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(v), nil
	case protoreflect.EnumKind:
		return parseEnumField(fd, value)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(value), nil
	case protoreflect.BytesKind:
		v, err := base64.URLEncoding.DecodeString(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBytes(v), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return parseMessage(fd.Message(), value)
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported field kind: %v", kind)
	}
}

// parseEnumField parses enum fields.
func parseEnumField(fd protoreflect.FieldDescriptor, value string) (protoreflect.Value, error) {
	enum, err := protoregistry.GlobalTypes.FindEnumByName(fd.Enum().FullName())
	switch {
	case errors.Is(err, protoregistry.NotFound):
		return protoreflect.Value{}, fmt.Errorf("enum %s is not registered", fd.Enum().FullName())
	case err != nil:
		return protoreflect.Value{}, fmt.Errorf("failed to look up enum: %w", err)
	}
	v := enum.Descriptor().Values().ByName(protoreflect.Name(value))
	if v == nil {
		i, err := strconv.Atoi(value)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("%s is not a valid value", value)
		}
		v = enum.Descriptor().Values().ByNumber(protoreflect.EnumNumber(i))
		if v == nil {
			return protoreflect.Value{}, fmt.Errorf("%s is not a valid value", value)
		}
	}
	return protoreflect.ValueOfEnum(v.Number()), nil
}

// parseMessage parses string value into protoreflect.Value by protoreflect.MessageDescriptor.
// It's used to parse google.protobuf.xxx.
func parseMessage(md protoreflect.MessageDescriptor, value string) (protoreflect.Value, error) {
	switch md.FullName() {
	case "google.protobuf.Timestamp":
		return parseTimestampMessage(value)
	case "google.protobuf.Duration":
		return parseDurationMessage(value)
	case "google.protobuf.DoubleValue":
		return parseDoubleValueMessage(value)
	case "google.protobuf.FloatValue":
		return parseFloatValueMessage(value)
	case "google.protobuf.Int64Value":
		return parseInt64ValueMessage(value)
	case "google.protobuf.Int32Value":
		return parseInt32ValueMessage(value)
	case "google.protobuf.UInt64Value":
		return parseUInt64ValueMessage(value)
	case "google.protobuf.UInt32Value":
		return parseUInt32ValueMessage(value)
	case "google.protobuf.BoolValue":
		return parseBoolValueMessage(value)
	case "google.protobuf.StringValue":
		sv := &wrapperspb.StringValue{Value: value}
		return protoreflect.ValueOfMessage(sv.ProtoReflect()), nil
	case "google.protobuf.BytesValue":
		return parseBytesValueMessage(value)
	case "google.protobuf.FieldMask":
		fm := &fieldmaskpb.FieldMask{}
		fm.Paths = append(fm.Paths, strings.Split(value, ",")...)
		return protoreflect.ValueOfMessage(fm.ProtoReflect()), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported message type: %s", string(md.FullName()))
	}
}

// parseTimestampMessage parses google.protobuf.Timestamp.
func parseTimestampMessage(value string) (protoreflect.Value, error) {
	var msg proto.Message
	if value != "null" {
		t, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = timestamppb.New(t)
	}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseDurationMessage parses google.protobuf.Duration.
func parseDurationMessage(value string) (protoreflect.Value, error) {
	var msg proto.Message
	if value != "null" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = durationpb.New(d)
	}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseDoubleValueMessage parses google.protobuf.DoubleValue.
func parseDoubleValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.DoubleValue{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseFloatValueMessage parses google.protobuf.FloatValue.
func parseFloatValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.FloatValue{Value: float32(v)}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseInt64ValueMessage parses google.protobuf.Int64Value.
func parseInt64ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.Int64Value{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseInt32ValueMessage parses google.protobuf.Int32Value.
func parseInt32ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.Int32Value{Value: int32(v)}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseUInt64ValueMessage parses google.protobuf.UInt64Value.
func parseUInt64ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.UInt64Value{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseUInt32ValueMessage parses google.protobuf.UInt32Value.
func parseUInt32ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.UInt32Value{Value: uint32(v)}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseBoolValueMessage parses google.protobuf.BoolValue.
func parseBoolValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.BoolValue{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseBytesValueMessage parses google.protobuf.BytesValue.
func parseBytesValueMessage(value string) (protoreflect.Value, error) {
	v, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.BytesValue{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// setFieldMask sets field mask for the field.
func setFieldMask(message protoreflect.Message, fieldPath string) error {
	maskFd := theMaskField(message)
	if maskFd == nil {
		return nil
	}

	partiallyUpdated, err := fdByName(message, fieldPath)
	if err != nil {
		return fmt.Errorf("failed to find partially updated field %s, err: %w", fieldPath, err)
	}
	if !isPlainMessage(partiallyUpdated) {
		return fmt.Errorf("with FieldMask enabled, partially updated field must be a plain message")
	}
	message.Set(maskFd, protoreflect.ValueOfMessage((&fieldmaskpb.FieldMask{
		Paths: getPopulatedFieldPaths(message.Get(partiallyUpdated).Message()),
	}).ProtoReflect()))
	return nil
}

// theMaskField returns the only field whose type is googleProtobufFieldMaskFullName, otherwise, returns nil.
func theMaskField(message protoreflect.Message) protoreflect.FieldDescriptor {
	var count int
	var theFd protoreflect.FieldDescriptor
	message.Descriptor().Fields()
	for i, fds := 0, message.Descriptor().Fields(); i < fds.Len(); i++ {
		fd := fds.Get(i)
		if isPlainMessage(fd) && fd.Message().FullName() == googleProtobufFieldMaskFullName {
			count++
			theFd = fd
		}
	}

	if count == 1 {
		return theFd
	}
	return nil
}

var googleProtobufFieldMaskFullName = (*fieldmaskpb.FieldMask)(nil).ProtoReflect().Descriptor().FullName()

func isPlainMessage(fd protoreflect.FieldDescriptor) bool {
	return fd.Message() != nil && !fd.IsList() && !fd.IsMap()
}

// getPopulatedFieldPaths returns all populated field paths.
func getPopulatedFieldPaths(message protoreflect.Message) []string {
	var res []string
	dfs(message, []string{}, &res)
	return res
}

// dfs performs the Depth-first search algorithm.
func dfs(message protoreflect.Message, paths []string, res *[]string) {
	message.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		name := string(fd.FullName().Name())
		if isPlainMessage(fd) {
			dfs(v.Message(), append(paths, name), res)
		} else {
			*res = append(*res, strings.Join(append(paths, name), "."))
		}
		return true
	})
}
