// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

// Package frame contains transport-level frame utilities.
package frame

// ShouldCopy judges whether to enable frame copy according to the current framer and options.
func ShouldCopy(isCopyOption, serverAsync, isSafeFramer bool) bool {
	// The following two scenarios do not need to copy frame.
	// Scenario 1: Framer is already safe on concurrent read.
	if isSafeFramer {
		return false
	}
	// Scenario 2: The server is in sync mod, and the caller does not want to copy frame(not stream RPC).
	if !serverAsync && !isCopyOption {
		return false
	}

	// Otherwise:
	// To avoid data overwriting of the concurrent reading Framer, enable copy frame by default.
	return true
}
