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

package transport

import "context"

// KeepOrderPreDecodeExtractor extracts a key for keeping request order from the decoded request body.
type KeepOrderPreDecodeExtractor func(ctx context.Context, reqBody []byte) (string, bool)

// KeepOrderPreUnmarshalExtractor extracts a key for keeping request order from the unmarshaled request body.
type KeepOrderPreUnmarshalExtractor func(ctx context.Context, reqBody interface{}) (string, bool)

// OrderedGroups keeps requests ordered by key.
type OrderedGroups interface {
	Add(key string, fn func())
	Remove(key string)
	Stop()
}
