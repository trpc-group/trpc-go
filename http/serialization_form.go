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

package http

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"

	"trpc.group/trpc-go/trpc-go/codec"

	"github.com/go-playground/form/v4"
	"github.com/mitchellh/mapstructure"
)

// Uses the same tag as json.
var tag = "json"

func init() {
	codec.RegisterSerializer(
		codec.SerializationTypeForm,
		NewFormSerialization(tag),
	)
}

// NewFormSerialization initializes the form serialized object.
func NewFormSerialization(tag string) codec.Serializer {
	encoder := form.NewEncoder()
	encoder.SetTagName(tag)
	decoder := form.NewDecoder()
	decoder.SetTagName(tag)
	return &FormSerialization{
		tagname: tag,
		MapType: false,
		encode:  encoder.Encode,
		decode:  wrapDecodeWithRecovery(decoder.Decode),
	}
}

// FormSerialization packages the kv structure of http get request.
type FormSerialization struct {
	tagname string
	// MapType is used to determine the serialization method of a map,
	// which defaults to false and follows the logic of the original form/v4.
	MapType bool
	encode  func(interface{}) (url.Values, error)
	decode  func(interface{}, url.Values) error
}

// Unmarshal unpacks kv structure.
func (j *FormSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	switch body.(type) {
	// go-playground/form does not support map structure.
	case map[string]interface{}, *map[string]interface{}, map[string]string, *map[string]string,
		url.Values, *url.Values: // Essentially, the underlying type of 'url.Values' is also a map.
		return unmarshalValues(j.tagname, values, body)
	default:
	}
	// First try using go-playground/form, it can handle nested struct.
	// But it cannot handle Chinese characters in byte slice.
	err = j.decode(body, values)
	if err == nil {
		return nil
	}
	// Second try using mapstructure.
	if e := unmarshalValues(j.tagname, values, body); e != nil {
		return fmt.Errorf("unmarshal error: first try err = %+v, second try err = %w", err, e)
	}
	return nil
}

// wrapDecodeWithRecovery wraps the decode function, adding panic recovery to handle
// panics as errors. This function is designed to prevent malformed query parameters
// from causing a panic, which is the default behavior of the go-playground/form decoder
// implementation. This is because, in certain cases, it's more acceptable to receive
// a degraded result rather than experiencing a direct server crash.
// Besides, the behavior of not panicking also ensures backward compatibility (<v0.16.0).
// The original go-playground/form has an issue with introducing 'strict' behavior
// into its underlying implementation to replace hard panic. However, a promising
// outcome cannot be foreseen.
// Refer to: https://github.com/go-playground/form/issues/28.
func wrapDecodeWithRecovery(
	f func(interface{}, url.Values) error,
) func(interface{}, url.Values) error {
	return func(v interface{}, values url.Values) (err error) {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("panic: %+v", e)
			}
		}()
		return f(v, values)
	}
}

// unmarshalValues parses the corresponding fields in values according to tagname.
func unmarshalValues(tagname string, values url.Values, body interface{}) error {
	// To handle the scenario where the underlying type of 'body' is 'url.Values'.
	if b, ok := body.(url.Values); ok && b != nil {
		for k, v := range values {
			b[k] = v
		}
		return nil
	}
	params := map[string]interface{}{}
	for k, v := range values {
		if len(v) == 1 {
			params[k] = v[0]
		} else {
			params[k] = v
		}
	}
	config := &mapstructure.DecoderConfig{TagName: tagname, Result: body, WeaklyTypedInput: true, Metadata: nil}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(params)
}

// encodeArray recursively process nested array.
func encodeArray(prefix string, arr interface{}, values url.Values) {
	v := reflect.ValueOf(arr)
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			newPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			encodeArray(newPrefix, v.Index(i).Interface(), values)
		}
	default:
		values.Add(prefix, fmt.Sprintf("%v", arr))
	}
}

// processValue processes a value, handling maps, arrays, slices, and basic types.
func processValue(v reflect.Value, prefix string, values url.Values, root bool) error {
	switch v.Kind() {
	case reflect.Map:
		return processMap(v, prefix, values)
	case reflect.Array, reflect.Slice:
		if v.Len() > 0 && v.Len() > 0 && v.Index(0).Kind() != reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				values.Add(prefix, fmt.Sprintf("%v", v.Index(i)))
			}
		} else {
			encodeArray(prefix, v.Interface(), values)
		}
	default:
		values.Add(prefix, fmt.Sprintf("%v", v.Interface()))
	}
	return nil
}

// processMap recursively process nested maps.
func processMap(v reflect.Value, prefix string, values url.Values) error {
	if v.Kind() != reflect.Map {
		return fmt.Errorf("expected a map, got %s", v.Kind())
	}
	for _, key := range v.MapKeys() {
		var strKey string
		switch key.Kind() {
		case reflect.String:
			strKey = key.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strKey = strconv.FormatInt(key.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			strKey = strconv.FormatUint(key.Uint(), 10)
		default:
			strKey = fmt.Sprintf("%v", key)
			fmt.Printf("Warning: map key is of type %s, using %v as key\n", key.Kind(), strKey)
		}
		if prefix != "" {
			strKey = prefix + "." + strKey
		}
		value := v.MapIndex(key)
		if err := processValue(value, strKey, values, true); err != nil {
			return err
		}
	}
	return nil
}

// mapToUrlValues convert a map to url.Values.
func mapToUrlValues(body interface{}) ([]byte, error) {
	values := url.Values{}
	if err := processMap(reflect.ValueOf(body), "", values); err != nil {
		return nil, err
	}
	return []byte(values.Encode()), nil
}

func (j *FormSerialization) otherTypeToUrlValues(body interface{}) ([]byte, error) {
	val, err := j.encode(body)
	if err != nil {
		return nil, err
	}
	return []byte(val.Encode()), err
}

// Marshal packages kv structure.
func (j *FormSerialization) Marshal(body interface{}) ([]byte, error) {
	// Used to send form urlencode post request to backend.
	if req, ok := body.(url.Values); ok {
		return []byte(req.Encode()), nil
	}
	// Due to the inability of the form package to correctly serialize the map type, a special judgment is made here.
	if j.MapType == true && reflect.TypeOf(body).Kind() == reflect.Map {
		return mapToUrlValues(body)
	}
	return j.otherTypeToUrlValues(body)
}
