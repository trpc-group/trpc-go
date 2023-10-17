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

// This file shows how to load plugins through a yaml config file.

package plugin

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	// SetupTimeout is the timeout for initialization of each plugin.
	// Modify it if some plugins' initialization does take a long time.
	SetupTimeout = 3 * time.Second

	// MaxPluginSize is the max number of plugins.
	MaxPluginSize = 1000
)

// Config is the configuration of all plugins. plugin type => { plugin name => plugin config }
type Config map[string]map[string]yaml.Node

// SetupClosables loads plugins and returns a function to close them in reverse order.
func (c Config) SetupClosables() (close func() error, err error) {
	// load plugins one by one through the config file and put them into an ordered plugin queue.
	plugins, status, err := c.loadPlugins()
	if err != nil {
		return nil, err
	}

	// remove and setup plugins one by one from the front of the ordered plugin queue.
	pluginInfos, closes, err := c.setupPlugins(plugins, status)
	if err != nil {
		return nil, err
	}

	// notifies all plugins that plugin initialization is done.
	if err := c.onFinish(pluginInfos); err != nil {
		return nil, err
	}

	return func() error {
		for i := len(closes) - 1; i >= 0; i-- {
			if err := closes[i](); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func (c Config) loadPlugins() (chan pluginInfo, map[string]bool, error) {
	var (
		plugins = make(chan pluginInfo, MaxPluginSize) // use channel as plugin queue
		// plugins' status. plugin key => {true: init done, false: init not done}.
		status = make(map[string]bool)
	)
	for typ, factories := range c {
		for name, cfg := range factories {
			factory := Get(typ, name)
			if factory == nil {
				return nil, nil, fmt.Errorf("plugin %s:%s no registered or imported, do not configure", typ, name)
			}
			p := pluginInfo{
				factory: factory,
				typ:     typ,
				name:    name,
				cfg:     cfg,
			}
			select {
			case plugins <- p:
			default:
				return nil, nil, fmt.Errorf("plugin number exceed max limit:%d", len(plugins))
			}
			status[p.key()] = false
		}
	}
	return plugins, status, nil
}

func (c Config) setupPlugins(plugins chan pluginInfo, status map[string]bool) ([]pluginInfo, []func() error, error) {
	var (
		result []pluginInfo
		closes []func() error
		num    = len(plugins)
	)
	for num > 0 {
		for i := 0; i < num; i++ {
			p := <-plugins
			// check if plugins that current plugin depends on have been initialized
			if deps, err := p.hasDependence(status); err != nil {
				return nil, nil, err
			} else if deps {
				// There are plugins that current plugin depends on haven't been initialized,
				// move current plugin to tail of the channel.
				plugins <- p
				continue
			}
			if err := p.setup(); err != nil {
				return nil, nil, err
			}
			if closer, ok := p.asCloser(); ok {
				closes = append(closes, closer.Close)
			}
			status[p.key()] = true
			result = append(result, p)
		}
		if len(plugins) == num { // none plugin is setup, circular dependency exists.
			return nil, nil, fmt.Errorf("cycle depends, not plugin is setup")
		}
		num = len(plugins) // continue to process plugins that were moved to tail of the channel.
	}
	return result, closes, nil
}

func (c Config) onFinish(plugins []pluginInfo) error {
	for _, p := range plugins {
		if err := p.onFinish(); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------------------------- //

// pluginInfo is the information of a plugin.
type pluginInfo struct {
	factory Factory
	typ     string
	name    string
	cfg     yaml.Node
}

// hasDependence decides if any other plugins that this plugin depends on haven't been initialized.
// The input param is the initial status of all plugins.
// The output bool param being true means there are plugins that this plugin depends on haven't been initialized,
// while being false means this plugin doesn't depend on any other plugin or all the plugins that his plugin depends
// on have already been initialized.
func (p *pluginInfo) hasDependence(status map[string]bool) (bool, error) {
	deps, ok := p.factory.(Depender)
	if ok {
		hasDeps, err := p.checkDependence(status, deps.DependsOn(), false)
		if err != nil {
			return false, err
		}
		if hasDeps { // 个别插件会同时强依赖和弱依赖多个不同插件，当所有强依赖满足后需要再判断弱依赖关系
			return true, nil
		}
	}
	fd, ok := p.factory.(FlexDepender)
	if ok {
		return p.checkDependence(status, fd.FlexDependsOn(), true)
	}
	// This plugin doesn't depend on any other plugin.
	return false, nil
}

// Depender is the interface for "Strong Dependence".
// If plugin a "Strongly" depends on plugin b, b must exist and
// a will be initialized after b's initialization.
type Depender interface {
	// DependsOn returns a list of plugins that are relied upon.
	// The list elements are in the format of "type-name" like [ "selector-polaris" ].
	DependsOn() []string
}

// FlexDepender is the interface for "Weak Dependence".
// If plugin a "Weakly" depends on plugin b and b does exist,
// a will be initialized after b's initialization.
type FlexDepender interface {
	FlexDependsOn() []string
}

func (p *pluginInfo) checkDependence(status map[string]bool, dependences []string, flexible bool) (bool, error) {
	for _, name := range dependences {
		if name == p.key() {
			return false, errors.New("plugin not allowed to depend on itself")
		}
		setup, ok := status[name]
		if !ok {
			if flexible {
				continue
			}
			return false, fmt.Errorf("depends plugin %s not exists", name)
		}
		if !setup {
			return true, nil
		}
	}
	return false, nil
}

// setup initializes a single plugin.
func (p *pluginInfo) setup() error {
	var (
		ch  = make(chan struct{})
		err error
	)
	go func() {
		err = p.factory.Setup(p.name, &YamlNodeDecoder{Node: &p.cfg})
		close(ch)
	}()
	select {
	case <-ch:
	case <-time.After(SetupTimeout):
		return fmt.Errorf("setup plugin %s timeout", p.key())
	}
	if err != nil {
		return fmt.Errorf("setup plugin %s error: %v", p.key(), err)
	}
	return nil
}

// YamlNodeDecoder is a decoder for a yaml.Node of the yaml config file.
type YamlNodeDecoder struct {
	Node *yaml.Node
}

// Decode decodes a yaml.Node of the yaml config file.
func (d *YamlNodeDecoder) Decode(cfg interface{}) error {
	if d.Node == nil {
		return errors.New("yaml node empty")
	}
	return d.Node.Decode(cfg)
}

// key returns the unique index of plugin in the format of 'type-name'.
func (p *pluginInfo) key() string {
	return p.typ + "-" + p.name
}

// onFinish notifies the plugin that all plugins' loading has been done by tRPC-Go.
func (p *pluginInfo) onFinish() error {
	f, ok := p.factory.(FinishNotifier)
	if !ok {
		// FinishNotifier not being implemented means notification of
		// completion of all plugins' loading is not needed.
		return nil
	}
	return f.OnFinish(p.name)
}

// FinishNotifier is the interface used to notify that all plugins' loading has been done by tRPC-Go.
// Some plugins need to implement this interface to be notified when all other plugins' loading has been done.
type FinishNotifier interface {
	OnFinish(name string) error
}

func (p *pluginInfo) asCloser() (Closer, bool) {
	closer, ok := p.factory.(Closer)
	return closer, ok
}

// Closer is the interface used to provide a close callback of a plugin.
type Closer interface {
	Close() error
}
