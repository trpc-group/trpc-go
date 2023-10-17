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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/plugin"

	"gopkg.in/yaml.v3"
)

type config struct {
	Plugins plugin.Config
}

const (
	pluginType        = "mock_type"
	pluginName        = "mock_name"
	pluginFailName    = "mock_fail_name"
	pluginTimeoutName = "mock_timeout_name"
	pluginDependName  = "mock_depend_name"
)

func TestConfig_Setup(t *testing.T) {
	const configInfoNotRegister = `
plugins:
  mock_type:
    mock_not_register:
      address: localhost:8000
`
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfoNotRegister), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)

	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	cfg = config{}
	err = yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	clo, err := cfg.Plugins.SetupClosables()
	assert.Nil(t, err)
	require.Nil(t, clo())
}

type mockTimeoutPlugin struct{}

func (p *mockTimeoutPlugin) Type() string {
	return pluginType
}

func (p *mockTimeoutPlugin) Setup(name string, decoder plugin.Decoder) error {
	time.Sleep(time.Second * 5)
	return nil
}
func TestConfig_TimeoutSetup(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_timeout_name:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register(pluginTimeoutName, &mockTimeoutPlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

type mockDependPlugin struct{}

func (p *mockDependPlugin) Type() string {
	return pluginType
}

func (p *mockDependPlugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockDependPlugin) DependsOn() []string {
	return []string{"mock_type-mock_name"}
}
func TestConfig_DependSetup(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_depend_name:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register(pluginDependName, &mockDependPlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	clo, err := cfg.Plugins.SetupClosables()
	assert.Nil(t, err)
	require.Nil(t, clo())
}
func TestConfig_ExceedSetup(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_depend_name:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register(pluginDependName, &mockDependPlugin{})
	tmp := plugin.MaxPluginSize
	plugin.MaxPluginSize = 1
	defer func() {
		plugin.MaxPluginSize = tmp
	}()
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

type mockDependNonePlugin struct{}

func (p *mockDependNonePlugin) Type() string {
	return pluginType
}

func (p *mockDependNonePlugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockDependNonePlugin) DependsOn() []string {
	return []string{"mock_type-mock_none_name"}
}
func TestConfig_DependNoneSetup(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_depend_name:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register(pluginDependName, &mockDependNonePlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

type mockDependSelfPlugin struct{}

func (p *mockDependSelfPlugin) Type() string {
	return pluginType
}

func (p *mockDependSelfPlugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockDependSelfPlugin) DependsOn() []string {
	return []string{"mock_type-mock_depend_name"}
}
func TestConfig_DependSelfSetup(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_depend_name:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register(pluginDependName, &mockDependSelfPlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

type mockDependCycle1Plugin struct{}

func (p *mockDependCycle1Plugin) Type() string {
	return pluginType
}

func (p *mockDependCycle1Plugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockDependCycle1Plugin) DependsOn() []string {
	return []string{"mock_type-mock_cycle2_name"}
}

type mockDependCycle2Plugin struct{}

func (p *mockDependCycle2Plugin) Type() string {
	return pluginType
}

func (p *mockDependCycle2Plugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockDependCycle2Plugin) DependsOn() []string {
	return []string{"mock_type-mock_cycle1_name"}
}
func TestConfig_DependCycleSetup(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_cycle1_name:
      address: localhost:8000
    mock_cycle2_name:
      address: localhost:8000
`
	plugin.Register("mock_cycle1_name", &mockDependCycle1Plugin{})
	plugin.Register("mock_cycle2_name", &mockDependCycle2Plugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

type mockFailPlugin struct{}

func (p *mockFailPlugin) Type() string {
	return pluginType
}

func (p *mockFailPlugin) Setup(name string, decoder plugin.Decoder) error {
	return errors.New("mock fail")
}
func TestConfig_SetupFail(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_fail_name:
      address: localhost:8000
`
	plugin.Register(pluginFailName, &mockFailPlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

func TestYamlNodeDecoder_Decode(t *testing.T) {
	var nodeCfg = struct {
		Address string
	}{}
	const configInfo = `
plugins:
  mock_type:
    mock_fail_name:
      address: localhost:8000
`
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	node := cfg.Plugins["mock_type"]["mock_fail_name"]
	d := &plugin.YamlNodeDecoder{Node: &node}
	err = d.Decode(&nodeCfg)
	assert.Nil(t, err)
	assert.Equal(t, "localhost:8000", nodeCfg.Address)

	// Node 为空判断失败
	d.Node = nil
	err = d.Decode(&nodeCfg)
	assert.NotNil(t, err)
}

type mockFlexDependerPlugin1 struct {
	testOrderCh chan int
}

func (p *mockFlexDependerPlugin1) Type() string {
	return pluginType
}

func (p *mockFlexDependerPlugin1) Setup(name string, decoder plugin.Decoder) error {
	p.testOrderCh <- 1
	return nil
}

func (p *mockFlexDependerPlugin1) FlexDependsOn() []string {
	return []string{"mock_type-mock_flex_depender_name3", "anything", "mock_type-mock_flex_depender_name2"}
}

type mockFlexDependerPlugin2 struct {
	testOrderCh chan int
}

func (p *mockFlexDependerPlugin2) Type() string {
	return pluginType
}

func (p *mockFlexDependerPlugin2) Setup(name string, decoder plugin.Decoder) error {
	p.testOrderCh <- 2
	return nil
}

func (p *mockFlexDependerPlugin2) FlexDependsOn() []string {
	return []string{"anything", "mock_type-mock_flex_depender_name3"}
}

type mockFlexDependerPlugin3 struct {
	testOrderCh chan int
}

func (p *mockFlexDependerPlugin3) Type() string {
	return pluginType
}

func (p *mockFlexDependerPlugin3) Setup(name string, decoder plugin.Decoder) error {
	p.testOrderCh <- 3
	return nil
}
func TestFlexDepender(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_flex_depender_name1:
      address: localhost:8000
    mock_flex_depender_name2:
      address: localhost:8000
    mock_flex_depender_name3:
      address: localhost:8000
`
	testOrderCh := make(chan int, 3)
	plugin.Register("mock_flex_depender_name1", &mockFlexDependerPlugin1{
		testOrderCh: testOrderCh,
	})
	plugin.Register("mock_flex_depender_name2", &mockFlexDependerPlugin2{
		testOrderCh: testOrderCh,
	})
	plugin.Register("mock_flex_depender_name3", &mockFlexDependerPlugin3{
		testOrderCh: testOrderCh,
	})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	clo, err := cfg.Plugins.SetupClosables()
	assert.Nil(t, err)
	require.Nil(t, clo())
	v, ok := <-testOrderCh
	assert.True(t, ok)
	assert.Equal(t, 3, v)
	v, ok = <-testOrderCh
	assert.True(t, ok)
	assert.Equal(t, 2, v)
	v, ok = <-testOrderCh
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

type mockBothDepender struct {
	testOrderCh chan int
}

func (p *mockBothDepender) Type() string {
	return pluginType
}

func (p *mockBothDepender) Setup(name string, decoder plugin.Decoder) error {
	p.testOrderCh <- 4
	return nil
}
func (p *mockBothDepender) DependsOn() []string {
	return []string{"mock_type-mock_flex_depender_name2"}
}

func (p *mockBothDepender) FlexDependsOn() []string {
	return []string{"mock_type-mock_flex_depender_name3"}
}
func TestBothDepender(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_both_depender:
      address: localhost:8000
    mock_flex_depender_name2:
      address: localhost:8000
    mock_flex_depender_name3:
      address: localhost:8000
`
	testOrderCh := make(chan int, 3)
	plugin.Register("mock_flex_depender_name3", &mockFlexDependerPlugin3{
		testOrderCh: testOrderCh,
	})
	plugin.Register("mock_flex_depender_name2", &mockFlexDependerPlugin2{
		testOrderCh: testOrderCh,
	})
	plugin.Register("mock_both_depender", &mockBothDepender{
		testOrderCh: testOrderCh,
	})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	clo, err := cfg.Plugins.SetupClosables()
	assert.Nil(t, err)
	require.Nil(t, clo())
	v, ok := <-testOrderCh
	assert.True(t, ok)
	assert.Equal(t, v, 3)
	v, ok = <-testOrderCh
	assert.True(t, ok)
	assert.Equal(t, v, 2)
	v, ok = <-testOrderCh
	assert.True(t, ok)
	assert.Equal(t, 4, v)
}

type mockFinishSuccPlugin struct{}

func (p *mockFinishSuccPlugin) Type() string {
	return pluginType
}

func (p *mockFinishSuccPlugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockFinishSuccPlugin) OnFinish(name string) error {
	return nil
}
func TestConfig_OnFinishSucc(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_finish_succ:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register("mock_finish_succ", &mockFinishSuccPlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	clo, err := cfg.Plugins.SetupClosables()
	assert.Nil(t, err)
	require.Nil(t, clo())
}

type mockFinishFailPlugin struct{}

func (p *mockFinishFailPlugin) Type() string {
	return pluginType
}

func (p *mockFinishFailPlugin) Setup(name string, decoder plugin.Decoder) error {
	return nil
}
func (p *mockFinishFailPlugin) OnFinish(name string) error {
	return errors.New("on finish fail")
}
func TestConfig_OnFinishFail(t *testing.T) {
	const configInfo = `
plugins:
  mock_type:
    mock_name:
      address: localhost:8000
    mock_finish_fail:
      address: localhost:8000
`
	plugin.Register(pluginName, &mockPlugin{})
	plugin.Register("mock_finish_fail", &mockFinishFailPlugin{})
	cfg := config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	_, err = cfg.Plugins.SetupClosables()
	assert.NotNil(t, err)
}

type mockClosablePlugin struct {
	onClose func() error
}

func (*mockClosablePlugin) Type() string {
	return pluginType
}

func (*mockClosablePlugin) Setup(string, plugin.Decoder) error {
	return nil
}

func (p *mockClosablePlugin) Close() error {
	return p.onClose()
}

func TestPluginClose(t *testing.T) {
	t.Run("close success", func(t *testing.T) {
		var closed bool
		plugin.Register("close success", &mockClosablePlugin{onClose: func() error {
			closed = true
			return nil
		}})
		close, err := plugin.Config{pluginType: {"close success": yaml.Node{}}}.SetupClosables()
		require.Nil(t, err)
		require.Nil(t, close())
		require.True(t, closed)
	})
	t.Run("close error", func(t *testing.T) {
		plugin.Register("close error", &mockClosablePlugin{onClose: func() error {
			return errors.New("close error")
		}})
		close, err := plugin.Config{pluginType: {"close error": yaml.Node{}}}.SetupClosables()
		require.Nil(t, err)
		require.Error(t, close())
	})
}
