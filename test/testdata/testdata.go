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

package testdata

import (
	"path/filepath"
	"runtime"
)

// basePath is the root directory of this package.
var basePath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basePath = filepath.Dir(currentFile)
}

// BasePath returns the root directory of this package.
func BasePath() string {
	return basePath
}

// Path returns the absolute path the given relative file or directory path,
// relative to the trpc.group/trpc-go/trpc-go/test/testdata directory in the user's GOPATH.
// If relativePath is already absolute, it is returned unmodified.
func Path(relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return relativePath
	}

	return filepath.Join(basePath, relativePath)
}
