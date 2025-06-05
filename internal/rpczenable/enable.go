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

// Package rpczenable provides global variabls for enabling rpcz.
// The reason why this package is located in trpc-go/internal instead of
// trpc-go/rpcz/internal is because the trpc-go/subdirs will not be able to
// import it if it is located within an inner internal package.
// It cannot be located in trpc-go/internal/rpcz either,
// as trpc-go/internal/rpcz imports trpc/rpcz.
package rpczenable

// Enabled indicates whether the rpcz recording is globally activated.
//
// We intentionally avoid using a lock to protect this global variable,
// as it is only initialized during startup and remains read-only thereafter.
//
// Although rpcz was initially designed as a separate module, not a comprehensive
// global package, we discovered that performance issues necessitated the
// introduction of this global variable to mitigate the unnecessary overhead
// caused by rpcz (even when it's a noop implementation).
// We sacrifice readability for performance. Curious readers should
// understand the tough decision we've made and comprehend the dilemmas
// and challenges faced by the framework developers.
var Enabled bool
