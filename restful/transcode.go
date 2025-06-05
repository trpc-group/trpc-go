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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/restful/dat"
)

const (
	// default size of http req body buffer
	defaultBodyBufferSize = 4096
)

// transcoder is for tRPC/httpjson transcoding.
type transcoder struct {
	name                 string
	input                func() ProtoMessage
	output               func() ProtoMessage
	handler              HandleFunc
	httpMethod           string
	pat                  *Pattern
	body                 BodyLocator
	respBody             ResponseBodyLocator
	router               *Router
	dat                  *dat.DoubleArrayTrie
	discardUnknownParams bool
	serviceImpl          interface{}
}

// requestParams are params required for transcoding.
type requestParams struct {
	compressor  Compressor
	serializer  Serializer
	body        io.Reader
	fieldValues map[string]string
	form        url.Values
}

// bodyBufferPool is the pool of http request body buffer.
var bodyBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, defaultBodyBufferSize))
	},
}

func (tr *transcoder) transcodeRequest(p requestParams) (ProtoMessage, error) {
	protoReq := tr.input()

	if err := tr.transcodeBody(protoReq, p.body, p.compressor, p.serializer); err != nil {
		return nil, errs.Wrapf(err, errs.RetServerDecodeFail, "transcoding body %v", p.body)
	}

	if err := transcodeFieldValues(protoReq, p.fieldValues); err != nil {
		return nil, errs.Wrapf(err, errs.RetServerDecodeFail, "transcoding field values %v", p.fieldValues)
	}

	if err := tr.transcodeQueryParams(protoReq, p.form); err != nil {
		return nil, errs.Wrapf(err, errs.RetServerDecodeFail, "transcoding query parameters %v", p.form)
	}

	return protoReq, nil
}

// transcodeBody transcodes tRPC/httpjson by http request body.
func (tr *transcoder) transcodeBody(protoReq proto.Message, body io.Reader, c Compressor, s Serializer) error {
	// HttpRule body is not specified
	if tr.body == nil {
		return nil
	}

	// decompress
	var reader io.Reader
	var err error
	if c != nil {
		if reader, err = c.Decompress(body); err != nil {
			return fmt.Errorf("failed to decompress request body: %w", err)
		}
	} else {
		reader = body
	}

	// read body
	buffer := bodyBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bodyBufferPool.Put(buffer)
	if _, err := io.Copy(buffer, reader); err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// unmarshal
	if err := s.Unmarshal(buffer.Bytes(), tr.body.Locate(protoReq)); err != nil {
		return fmt.Errorf("failed to unmarshal req body: %w", err)
	}

	// field mask will be set for PATCH method.
	if tr.httpMethod == http.MethodPatch && tr.body.Body() != "*" {
		return setFieldMask(protoReq.ProtoReflect(), tr.body.Body())
	}

	return nil
}

// transcodeFieldValues transcodes tRPC/httpjson by fieldValues from url path matching.
func transcodeFieldValues(msg proto.Message, fieldValues map[string]string) error {
	for fieldPath, value := range fieldValues {
		if err := PopulateMessage(msg, strings.Split(fieldPath, "."), []string{value}); err != nil {
			return err
		}
	}
	return nil
}

// transcodeQueryParams transcodes tRPC/httpjson by query params.
func (tr *transcoder) transcodeQueryParams(msg proto.Message, form url.Values) error {
	// Query params will be ignored if HttpRule body is *.
	if tr.body != nil && tr.body.Body() == "*" {
		return nil
	}

	for key, values := range form {
		fieldPath := strings.Split(key, ".")
		// filter fields specified by HttpRule pattern and body
		if tr.dat != nil && tr.dat.CommonPrefixSearch(fieldPath) {
			continue
		}
		// populate proto message
		if err := PopulateMessage(msg, fieldPath, values); err != nil {
			if !tr.discardUnknownParams || !errors.Is(err, ErrTraverseNotFound) {
				return err
			}
		}
	}

	return nil
}

// handle does tRPC Stub handling.
func (tr *transcoder) handle(ctx context.Context, reqBody interface{}) (proto.Message, error) {
	filters := tr.router.opts.FilterFunc()
	serviceImpl := tr.serviceImpl
	handleFunc := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
		return tr.handler(serviceImpl, ctx, reqBody)
	}
	rsp, err := filters.Filter(ctx, reqBody, handleFunc)
	if err != nil {
		return nil, err
	}

	if rsp == nil {
		// this may happen when cors filter fires preflight logic:
		//   https://git.woa.com/trpc-go/trpc-filter/blob/cors/v0.1.4/cors/cors.go#L217
		return tr.output(), nil
	}
	r, ok := rsp.(proto.Message)
	if !ok {
		return nil, fmt.Errorf(
			"expected a proto.Message as the response type during restful transcoding, but received %T",
			rsp)
	}
	return r, nil
}

// transcodeResponse transcodes tRPC/httpjson by response.
// HttpRule.response_body only specifies serialization of fields.
// So compression would be custom.
func (tr *transcoder) transcodeResponse(m proto.Message, s Serializer) ([]byte, error) {
	if tr.respBody == nil {
		return s.Marshal(m)
	}
	return s.Marshal(tr.respBody.Locate(m))
}
