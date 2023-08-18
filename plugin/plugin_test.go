// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package plugin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go/plugin"
)

type mockPlugin struct{}

func (p *mockPlugin) Type() string {
	return pluginType
}

func (p *mockPlugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}

func TestGet(t *testing.T) {
	plugin.Register(pluginName, &mockPlugin{})
	// test duplicate registration
	plugin.Register(pluginName, &mockPlugin{})
	p := plugin.Get(pluginType, pluginName)
	assert.NotNil(t, p)

	pNo := plugin.Get("notexist", pluginName)
	assert.Nil(t, pNo)
}
