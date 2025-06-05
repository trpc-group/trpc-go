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
	"context"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
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

func requestSerializer(contentTypes []string) Serializer {
	s, ok := serializer(contentTypes)
	if ok {
		return s
	}
	return defaultSerializer
}

func responseSerializer(accepts []string) (Serializer, bool) {
	return serializer(accepts)
}

func serializer(contentTypesOrAccepts []string) (Serializer, bool) {
	for _, contentTypesOrAccept := range contentTypesOrAccepts {
		if s := getSerializerWithDirectives(contentTypesOrAccept); s != nil {
			return s, true
		}
	}
	return nil, false
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

// RespSerializerGetter is used to retrieve the corresponding serializer.
type RespSerializerGetter func(ctx context.Context, r *http.Request) Serializer

// DefaultRespSerializerGetter returns a serializer through negotiation, defaulting to JSONPBSerializer.
var DefaultRespSerializerGetter = func(_ context.Context, r *http.Request) Serializer {
	s, ok := responseSerializer(r.Header[headerAccept])
	if !ok {
		s = requestSerializer(r.Header[headerContentType])
	}
	return s
}

type respSerializerGetterKey struct{}

func newContextWithRespSerializerGetter(ctx context.Context, sg RespSerializerGetter) context.Context {
	return context.WithValue(ctx, respSerializerGetterKey{}, sg)
}

func respSerializerGetterFromContext(ctx context.Context) (RespSerializerGetter, bool) {
	sg, ok := ctx.Value(respSerializerGetterKey{}).(RespSerializerGetter)
	return sg, ok
}

// FastHTTPRespSerializerGetter is used to retrieve the corresponding serializer for FastHTTP.
type FastHTTPRespSerializerGetter func(ctx context.Context, requestCtx *fasthttp.RequestCtx) Serializer

// DefaultFastHTTPRespSerializerGetter returns a serializer through negotiation,
// defaulting to JSONPBSerializer for FastHTTP.
var DefaultFastHTTPRespSerializerGetter = func(_ context.Context, requestCtx *fasthttp.RequestCtx) Serializer {
	s, ok := responseSerializer([]string{string(requestCtx.Request.Header.Peek(headerAccept))})
	if !ok {
		s = requestSerializer([]string{string(requestCtx.Request.Header.Peek(headerContentType))})
	}
	return s
}

type fastHTTPRespSerializerGetterKey struct{}

func newContextWithFastHTTPRespSerializerGetter(
	ctx context.Context, fsg FastHTTPRespSerializerGetter,
) context.Context {
	return context.WithValue(ctx, fastHTTPRespSerializerGetterKey{}, fsg)
}

func fastHTTPRespSerializerGetterFromContext(
	ctx context.Context,
) (FastHTTPRespSerializerGetter, bool) {
	fsg, ok := ctx.Value(fastHTTPRespSerializerGetterKey{}).(FastHTTPRespSerializerGetter)
	return fsg, ok
}
