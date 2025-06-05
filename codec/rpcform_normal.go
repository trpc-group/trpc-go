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

//go:build !optimization
// +build !optimization

package codec

import "strings"

// rpcNameIsTRPCForm checks whether the given string is of trpc form.
// It is equivalent to:
//
//	var r = regexp.MustCompile(`^/[^/.]+\.[^/]+/[^/.]+$`)
//
//	func rpcNameIsTRPCForm(s string) bool {
//		return r.MatchString(s)
//	}
//
// But regexp is much slower than the current version.
// Refer to BenchmarkRPCNameIsTRPCForm in message_bench_test.go.
func rpcNameIsTRPCForm(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] != '/' { // ^/
		return false
	}
	const start = 1
	firstDot := strings.Index(s[start:], ".")
	if firstDot == -1 || firstDot == 0 { // [^.]+\.
		return false
	}
	if strings.Contains(s[start:start+firstDot], "/") { // [^/]+\.
		return false
	}
	secondSlash := strings.Index(s[start+firstDot:], "/")
	if secondSlash == -1 || secondSlash == 1 { // [^/]+/
		return false
	}
	if start+firstDot+secondSlash == len(s)-1 { // The second slash should not be the last character.
		return false
	}
	const offset = 1
	if strings.ContainsAny(s[start+firstDot+secondSlash+offset:], "/.") { // [^/.]+$
		return false
	}
	return true
}
