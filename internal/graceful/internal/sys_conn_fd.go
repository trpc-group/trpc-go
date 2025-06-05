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

//go:build !windows

package graceful

import (
	"fmt"
	"syscall"
)

func sysConnFd[T any](l T) (int, error) {
	l = Unwrap(l)
	sysConn, ok := (interface{})(l).(syscall.Conn)
	if !ok {
		return 0, fmt.Errorf("%T is not a syscall.Conn", l)
	}
	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("failed to get rawConn: %w", err)
	}
	var fd int
	if err := rawConn.Control(func(fileDescriptor uintptr) {
		fd = int(fileDescriptor)
	}); err != nil {
		return 0, fmt.Errorf("failed to call Control: %w", err)
	}
	return fd, nil
}
