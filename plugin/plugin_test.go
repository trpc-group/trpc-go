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

package plugin_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestMustRegister(t *testing.T) {
	t.Run("no registered plugin", func(t *testing.T) {
		assert.Nil(t, plugin.Get("testMustRegister", "no registered plugin"))
	})
	plugin.MustRegister("testMustRegister", &mockPlugin{})
	t.Run("registered plugin", func(t *testing.T) {
		assert.NotNil(t, plugin.Get("mock_type", "testMustRegister"))
	})
	t.Run("repeat register", func(t *testing.T) {
		assert.Panics(t, func() {
			plugin.MustRegister("testMustRegister", &mockPlugin{})
		})
	})
}

func TestRegisterSetupHook(t *testing.T) {
	const key = "a_pseudo_plugin_type-a_pseudo_plugin_name"
	plugin.RegisterSetupHook(key, func(setup func() error) error {
		if err := setup(); err != nil {
			t.Logf("setup error %+v is logged and somehow handled, it is not returned up", err)
		}
		return nil
	})
	require.Nil(t, plugin.GetSetupHook(key)(func() error {
		return errors.New("setup error")
	}))
}
