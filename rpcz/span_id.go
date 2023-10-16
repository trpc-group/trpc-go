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

package rpcz

// SpanID is a unique identity of a span.
// a valid span ID is always a non-negative integer
type SpanID int64

// nilSpanID is a invalid span ID.
const nilSpanID SpanID = -1
