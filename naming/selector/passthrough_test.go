// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package selector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPassthroughSelectorSelect(t *testing.T) {
	selector := &passthroughSelector{}
	n, err := selector.Select("passthrough")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "passthrough")
	assert.Equal(t, n.ServiceName, "passthrough")
}

func TestPassthroughSelectorReport(t *testing.T) {
	selector := &passthroughSelector{}
	n, err := selector.Select("passthrough")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "passthrough")
	assert.Equal(t, n.ServiceName, "passthrough")
	assert.Nil(t, selector.Report(n, 0, nil))
}
