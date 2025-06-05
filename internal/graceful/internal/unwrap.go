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

package graceful

// Unwrap recursively unwraps all wrapper and returns the most inner type T.
func Unwrap[T any](v T) T {
	if unwrapper, ok := (interface{})(v).(interface{ Unwrap() T }); ok {
		return Unwrap[T](unwrapper.Unwrap())
	}
	return v
}
