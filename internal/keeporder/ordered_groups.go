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

// Package keeporder contains definitions for internal use.
package keeporder

// OrderedGroups keeps the order of requests by the given key for each group.
type OrderedGroups interface {
	Add(key string, fn func())
	Remove(key string)
	Stop()
}
