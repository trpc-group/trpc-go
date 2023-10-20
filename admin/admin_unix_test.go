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
// +build !windows

package admin

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/transport"
)

func TestReuseListener(t *testing.T) {
	value, ok := os.LookupEnv(transport.EnvGraceRestart)
	t.Cleanup(func() {
		if ok {
			if err := os.Setenv(transport.EnvGraceRestart, value); err != nil {
				t.Log(err)
			}
		} else {
			if err := os.Unsetenv(transport.EnvGraceRestart); err != nil {
				t.Log(err)
			}
		}
	})

	// listen success and save listener
	tln := mustListenTCP(t)
	t.Cleanup(func() {
		if err := tln.Close(); err != nil {
			t.Log(err)
		}
	})
	file, err := tln.File()
	assert.Nil(t, err)
	assert.NotNil(t, file)

	// reuse saved listener
	err = os.Setenv(transport.EnvGraceRestart, "1")
	assert.Nil(t, err)
	err = os.Setenv(transport.EnvGraceFirstFd, fmt.Sprint(file.Fd()))
	assert.Nil(t, err)
	err = os.Setenv(transport.EnvGraceRestartFdNum, "1")
	assert.Nil(t, err)

	s := NewServer()
	ln1, err := s.listen("tcp", tln.Addr().String())
	assert.Nil(t, err)
	assert.Equal(t, tln.Addr(), ln1.Addr())
	t.Cleanup(func() {
		if err := ln1.Close(); err != nil {
			t.Log(err)
		}
	})

	// listen fail on other addr if enable grace restart
	ln2, err := s.listen("tcp", testAddress)
	assert.NotNil(t, err)
	assert.Nil(t, ln2)
}
