// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNodeString(t *testing.T) {
	n := &Node{
		ServiceName: "name",
		Address:     "127.0.0.1:8080",
		CostTime:    time.Second,
	}
	assert.Equal(t, n.String(), fmt.Sprintf("service:%s, addr:%s, cost:%s",
		n.ServiceName, n.Address, n.CostTime))
}
