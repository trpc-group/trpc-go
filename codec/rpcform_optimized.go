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

//go:build optimization
// +build optimization

package codec

// rpcNameIsTRPCForm checks if the provided string conforms to the trpc format.
// In the optimized version, this function always returns false. This implies that even for
// the trpc protocol, the full rpc name will be utilized as the method name (instead of
// using only the segment following the last slash).
// This approach yields optimal performance. The trade-off, however, is that the method name
// displayed on the monitor will be incompatible with previous versions.
func rpcNameIsTRPCForm(s string) bool {
	return false
}
