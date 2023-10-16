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
	"net/http"
	"strings"
)

// Serializer is the interface for http body marshaling/unmarshalling.
type Serializer interface {
	// Marshal marshals the tRPC message itself or a field of it to http body.
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal unmarshalls http body to the tRPC message itself or a field of it.
	Unmarshal(data []byte, v interface{}) error
	// Name returns name of the Serializer.
	Name() string
	// ContentType returns the original media type indicated by Content-Encoding response header.
	ContentType() string
}

// jsonpb as default
var defaultSerializer Serializer = &JSONPBSerializer{AllowUnmarshalNil: true}

// serialization related http header
var (
	headerAccept      = http.CanonicalHeaderKey("Accept")
	headerContentType = http.CanonicalHeaderKey("Content-Type")
)

var serializers = make(map[string]Serializer)

// RegisterSerializer registers a Serializer.
// This function is not thread-safe, it should only be called in init() function.
func RegisterSerializer(s Serializer) {
	if s == nil || s.Name() == "" {
		panic("tried to register nil or anonymous serializer")
	}
	serializers[s.Name()] = s
}

// SetDefaultSerializer sets the default Serializer.
// This function is not thread-safe, it should only be called in init() function.
func SetDefaultSerializer(s Serializer) {
	if s == nil || s.Name() == "" {
		panic("tried to set nil or anonymous serializer as the default serializer")
	}
	defaultSerializer = s
}

// GetSerializer returns a Serializer by its name.
func GetSerializer(name string) Serializer {
	return serializers[name]
}

// serializerForTranscoding returns inbound/outbound Serializer for transcoding.
func serializerForTranscoding(contentTypes []string, accepts []string) (Serializer, Serializer) {
	var reqSerializer, respSerializer Serializer // neither should be nil

	// ContentType => Req Serializer
	for _, contentType := range contentTypes {
		if s := getSerializerWithDirectives(contentType); s != nil {
			reqSerializer = s
			break
		}
	}

	// Accept => Resp Serializer
	for _, accept := range accepts {
		if s := getSerializerWithDirectives(accept); s != nil {
			respSerializer = s
			break
		}
	}

	if reqSerializer == nil { // use defaultSerializer if reqSerializer is nil
		reqSerializer = defaultSerializer
	}
	if respSerializer == nil { // use reqSerializer if respSerializer is nil
		respSerializer = reqSerializer
	}

	return reqSerializer, respSerializer
}

// getSerializerWithDirectives get Serializer by Content-Type or Accept. The name may have directives after ';'.
// All Serializers are considered the same as the one with only one directive "charset=UTF-8".
// Other directives are not supported, and will cause the function to return nil.
func getSerializerWithDirectives(name string) Serializer {
	if s, ok := serializers[name]; ok {
		return s
	}
	pos := strings.Index(name, ";")
	const charsetUTF8 = "charset=utf-8"
	if pos == -1 || strings.ToLower(strings.TrimSpace(name[pos+1:])) != charsetUTF8 {
		return nil
	}
	if s, ok := serializers[name[:pos]]; ok {
		return s
	}
	return nil
}
