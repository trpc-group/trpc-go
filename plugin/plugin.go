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

// Package plugin implements a general plugin factory system which provides plugin registration and loading.
// It is mainly used when certain plugins must be loaded by configuration.
// This system is not supposed to register plugins that do not rely on configuration like codec. Instead, plugins
// that do not rely on configuration should be registered by calling methods in certain packages.
package plugin

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jinzhu/copier"
)

var plugins = make(map[string]map[string]Factory) // plugin type => { plugin name => plugin factory }

// Factory is the interface for plugin factory abstraction.
// Custom Plugins need to implement this interface to be registered as a plugin with certain type.
type Factory interface {
	// Type returns type of the plugin, i.e. selector, log, config, tracing.
	Type() string
	// Setup loads plugin by configuration.
	// The data structure of the configuration of the plugin needs to be defined in advance。
	Setup(name string, dec Decoder) error
}

// Decoder is the interface used to decode plugin configuration.
type Decoder interface {
	Decode(cfg interface{}) error // the input param is the custom configuration of the plugin
}

// newCopierDecoder returns a copierDecoder holding the configuration, cfg must be a pointer.
func newCopierDecoder(cfg interface{}) Decoder {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		panic(fmt.Sprintf("The config %T must be a pointer", cfg))
	}
	return &copierDecoder{cfg: cfg}
}

// copierDecoder implements the Decoder interface and is responsible for
// copying the cfg field.
type copierDecoder struct {
	cfg interface{}
}

// Decode assigns the sd.cfg to dst.
func (sd *copierDecoder) Decode(dst interface{}) error {
	if reflect.TypeOf(sd.cfg) != reflect.TypeOf(dst) {
		return fmt.Errorf("parameter config is unexpected type, raw: %T, decoding: %T", sd.cfg, dst)
	}
	return copier.Copy(dst, sd.cfg)
}

// Register registers a plugin factory.
// Name of the plugin should be specified.
// It is supported to register instances which are the same implementation of plugin Factory
// but use different configuration.
func Register(name string, f Factory) {
	factories, ok := plugins[f.Type()]
	if !ok {
		factories = make(map[string]Factory)
		plugins[f.Type()] = factories
	}
	factories[name] = f
}

// MustRegister registers a plugin factory.
// It will panic if the plugin has been registered.
//
// In most cases, the framework uses the init + Register method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegister to forcibly register a component 'xxx', while the framework
// uses init + Register to register another component 'yyy', conflicts may occur. If the init function
// for MustRegister is executed before the conflicting init function, MustRegister might not raise an
// error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegister and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegister(name string, f Factory) {
	if Get(f.Type(), name) != nil {
		panic("plugin already registered: " + name)
	}
	Register(name, f)
}

// Get returns a plugin Factory by its type and name.
func Get(typ string, name string) Factory {
	return plugins[typ][name]
}

// RegisterSetupHook is used to register a setupHook for the specified plugin key.
// The plugin key is of format 'type-name', e.g. 'config-rainbow', 'naming-polaris'.
// The default implementation involves invoking the "setup" function in a separate goroutine,
// and using plugin.SetupTimeout to explicitly control the timeout.
// If the setup or timeout error is returned from the hook, the framework
// will panic during trpc.NewServer.
// If you want to avoid the panic, you can choose to register a setupHook for
// the plugin you are interested in, and return nil after handling the setup error.
//
// Some possible implementations:
//
//	// Use degradation strategies to handle the error.
//	plugin.RegisterSetupHook("plugin_type-plugin_name", func(setup func() error) error {
//		if err := setup(); err != nil {
//			// Implement degradation strategies to handle the error.
//		}
//		return nil // Return nil to avoid panic.
//	})
//
//	// After a certain timeout, use degradation strategies to handle the error.
//	plugin.RegisterSetupHook("plugin_type-plugin_name", func(setup func() error) error {
//		ch := make(chan error)
//		go func() { ch <- setup() }()
//		select {
//		case err := <-ch:
//			// Implement degradation strategies to handle the error.
//		case <-time.After(certainTimeout):
//			// Implement degradation strategies to handle the error.
//		}
//		return nil // Return nil to avoid panic.
//	})
func RegisterSetupHook(key string, hook setupHook) {
	setupHooks[key] = hook
}

// GetSetupHook retrieves the setup hook for the specified plugin key.
// The plugin key is of format 'type-name', e.g. 'config-rainbow', 'naming-polaris'.
// The default setup hook involves invoking the "setup" function in a separate goroutine,
// with the timeout being controlled explicitly using plugin.SetupTimeout.
func GetSetupHook(key string) setupHook {
	if hook, ok := setupHooks[key]; ok {
		return hook
	}
	return func(setup func() error) error {
		ch := make(chan error)
		start := time.Now()
		go func() { ch <- setup() }()
		select {
		case err := <-ch:
			if err != nil {
				return fmt.Errorf("setup plugin %s error: %w", key, err)
			}
			// Use fmt.Printf instead of log.Infof because the logger in trpc-go may also need to be set up in plugins.
			fmt.Printf("plugin %s setup succeed, time elapsed: %v\n", key, time.Since(start))
			return nil
		case <-time.After(SetupTimeout):
			return fmt.Errorf("timeout occurred while setting up plugin %s after %v. "+
				"you can edit the plugin.SetupTimeout (or global.plugin_setup_timeout in trpc_go.yaml) "+
				"to increase the timeout, "+
				"or you can use plugin.RegisterSetupHook(\"%s\", func(..){..}) "+
				"to manually call the setup function and handle errors on your own to avoid panic",
				key, SetupTimeout, key)
		}
	}
}

type setupHook = func(setup func() error) error

var setupHooks = make(map[string]setupHook)
