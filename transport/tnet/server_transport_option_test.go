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

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package tnet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	tnettrans "trpc.group/trpc-go/trpc-go/transport/tnet"
)

func TestSetNumPollers(t *testing.T) {
	err := tnettrans.SetNumPollers(2)
	assert.Nil(t, err)
}

func TestOptions(t *testing.T) {
	opts := &tnettrans.ServerTransportOptions{}
	tnettrans.WithKeepAlivePeriod(time.Second)(opts)
	assert.Equal(t, time.Second, opts.KeepAlivePeriod)
}
