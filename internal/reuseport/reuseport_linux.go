// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

//go:build linux
// +build linux

package reuseport

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var reusePort = 0x0F
var maxConnFileName = "/proc/sys/net/core/somaxconn"

func maxListenerBacklog() int {
	fd, err := os.Open(maxConnFileName)
	if err != nil {
		return syscall.SOMAXCONN
	}
	defer fd.Close()

	rd := bufio.NewReader(fd)
	line, err := rd.ReadString('\n')
	if err != nil {
		return syscall.SOMAXCONN
	}

	f := strings.Fields(line)
	if len(f) < 1 {
		return syscall.SOMAXCONN
	}

	n, err := strconv.Atoi(f[0])
	return defaultBacklog(uint32(n), err)
}

func defaultBacklog(n uint32, err error) int {
	if n == 0 || err != nil {
		return syscall.SOMAXCONN
	}

	// Linux stores the backlog in a uint16.
	// Truncate number to avoid wrapping.
	// See issue 5030.
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return int(n)
}
