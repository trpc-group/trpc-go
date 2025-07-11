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

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpDiscovery(t *testing.T) {
	d := &IPDiscovery{}
	list, err := d.List("ipdiscovery.ip.62:8989", nil)
	assert.Nil(t, err)
	assert.Equal(t, len(list), 1)
	assert.Equal(t, list[0].Address, "ipdiscovery.ip.62:8989")
}
