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

// Package env defines environment variables used inside the framework.
package env

// Defines all keys of the environment variables.
const (
	// LogTrace controls whether to output trace log.
	// To enable trace output, set TRPC_LOG_TRACE=1.
	//
	// This environment variable is needed because zap library lacks trace-level log.
	LogTrace = "TRPC_LOG_TRACE"
)
