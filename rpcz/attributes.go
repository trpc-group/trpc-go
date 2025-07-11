//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package rpcz

import "strings"

// Attribute records attribute with (Name, Value) pair.
type Attribute struct {
	// Name of Attribute.
	Name string
	// Value of Attribute.
	Value interface{}
}

const (
	// TRPCAttributeRPCName is used to set the RPCName attribute of span.
	TRPCAttributeRPCName = "__@*TRPCAttribute(RPCName)*@__"
	// TRPCAttributeError is used to set the Error attribute of span.
	TRPCAttributeError = "__@*TRPCAttribute(Error)*@__"
	// TRPCAttributeResponseSize is used to set the ResponseSize attribute of span.
	TRPCAttributeResponseSize = "__@*TRPCAttribute(ResponseSize)*@__"
	// TRPCAttributeRequestSize is used to set the RequestSize attribute of span.
	TRPCAttributeRequestSize = "__@*TRPCAttribute(RequestSize)*@__"
	// TRPCAttributeFilterNames is used to set the FilterNames attribute of span.
	TRPCAttributeFilterNames = "__@*TRPCAttribute(FilterNames)*@__"

	// HTTPAttributeURL is used to set the URL attribute of span.
	HTTPAttributeURL = "__@*HTTPAttribute(URL)*@__"
	// HTTPAttributeRequestContentLength is used to set the Request's ContentLength attribute of span.
	HTTPAttributeRequestContentLength = "__@*HTTPAttribute(RequestContentLength)*@__"
)

func parsedTRPCAttribute(name string) string {
	return strings.TrimSuffix(strings.TrimPrefix(name, "__@*TRPCAttribute("), ")*@__")
}
