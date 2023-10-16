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
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/dat"
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

// transcodeParams are params required for transcoding.
type transcodeParams struct {
	reqCompressor  Compressor
	respCompressor Compressor
	reqSerializer  Serializer
	respSerializer Serializer
	body           io.Reader
	fieldValues    map[string]string
	form           url.Values
}

// paramsPool is the transcodeParams pool.
var paramsPool = sync.Pool{
	New: func() interface{} {
		return &transcodeParams{}
	},
}

// putBackParams puts transcodeParams back to pool.
func putBackParams(params *transcodeParams) {
	params.reqCompressor = nil
	params.respCompressor = nil
	params.reqSerializer = nil
	params.respSerializer = nil
	params.body = nil
	params.fieldValues = nil
	params.form = nil
	paramsPool.Put(params)
}

// transcode transcodes tRPC/httpjson.
func (tr *transcoder) transcode(
	stubCtx context.Context,
	params *transcodeParams,
) (proto.Message, []byte, error) {
	// init tRPC request
	protoReq := tr.input()

	// transcode body
	if err := tr.transcodeBody(protoReq, params.body, params.reqCompressor,
		params.reqSerializer); err != nil {
		return nil, nil, errs.New(errs.RetServerDecodeFail, err.Error())
	}

	// transcode fieldValues from url path matching
	if err := tr.transcodeFieldValues(protoReq, params.fieldValues); err != nil {
		return nil, nil, errs.New(errs.RetServerDecodeFail, err.Error())
	}

	// transcode query params
	if err := tr.transcodeQueryParams(protoReq, params.form); err != nil {
		return nil, nil, errs.New(errs.RetServerDecodeFail, err.Error())
	}

	// tRPC Stub handling
	rsp, err := tr.handle(stubCtx, protoReq)
	if err != nil {
		return nil, nil, err
	}
	var protoResp proto.Message
	if rsp == nil {
		protoResp = tr.output()
	} else {
		protoResp = rsp.(proto.Message)
	}

	// response
	// HttpRule.response_body only specifies serialization of fields.
	// So compression would be custom.
	buf, err := tr.transcodeResp(protoResp, params.respSerializer)
	if err != nil {
		return nil, nil, errs.New(errs.RetServerEncodeFail, err.Error())
	}
	return protoResp, buf, nil
}

// bodyBufferPool is the pool of http request body buffer.
var bodyBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, defaultBodyBufferSize))
	},
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
	if tr.httpMethod == "PATCH" && tr.body.Body() != "*" {
		return setFieldMask(protoReq.ProtoReflect(), tr.body.Body())
	}

	return nil
}

// transcodeFieldValues transcodes tRPC/httpjson by fieldValues from url path matching.
func (tr *transcoder) transcodeFieldValues(msg proto.Message, fieldValues map[string]string) error {
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
		// filter fields specified by HttpRule pattern and body
		if tr.dat != nil && tr.dat.CommonPrefixSearch(strings.Split(key, ".")) {
			continue
		}
		// populate proto message
		if err := PopulateMessage(msg, strings.Split(key, "."), values); err != nil {
			if !tr.discardUnknownParams || !errors.Is(err, ErrTraverseNotFound) {
				return err
			}
		}
	}

	return nil
}

// handle does tRPC Stub handling.
func (tr *transcoder) handle(ctx context.Context, reqBody interface{}) (interface{}, error) {
	filters := tr.router.opts.FilterFunc()
	serviceImpl := tr.serviceImpl
	handleFunc := func(ctx context.Context, reqBody interface{}) (interface{}, error) {
		return tr.handler(serviceImpl, ctx, reqBody)
	}
	return filters.Filter(ctx, reqBody, handleFunc)
}

// transcodeResp transcodes tRPC/httpjson by response.
func (tr *transcoder) transcodeResp(protoResp proto.Message, s Serializer) ([]byte, error) {
	// marshal
	var obj interface{}
	if tr.respBody == nil {
		obj = protoResp
	} else {
		obj = tr.respBody.Locate(protoResp)
	}
	return s.Marshal(obj)
}
