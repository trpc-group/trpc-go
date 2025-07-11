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

package servicerouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceRouterRegister(t *testing.T) {
	Register("noop", &NoopServiceRouter{})
	assert.NotNil(t, Get("noop"))
	unregisterForTesting("noop")
}

func TestSetDefaultServiceRouter(t *testing.T) {
	noop := &NoopServiceRouter{}
	SetDefaultServiceRouter(noop)
	assert.Equal(t, noop, DefaultServiceRouter)
	nodes, err := noop.Filter("noop_service", nil)
	assert.Nil(t, err)
	assert.Len(t, nodes, 0)
}
