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
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPC(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.Nil(t, err)

	w, r := NewRpcWriter(fds[0]), NewRpcReader(fds[1])

	require.Nil(t, w.Encode(1))
	l, err := net.Listen("tcp", "")
	require.Nil(t, err)
	fd, err := sysConnFd(l)
	require.Nil(t, err)
	require.Nil(t, w.Flush([]int{fd}))

	require.Len(t, r.GetFds(), 0)
	var i int
	require.Nil(t, r.Decode(&i))
	require.Len(t, r.GetFds(), 1)
}
