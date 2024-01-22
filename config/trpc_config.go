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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cast"
	yaml "gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go/log"
)

var (
	// ErrConfigNotExist is config not exist error
	ErrConfigNotExist = errors.New("trpc/config: config not exist")

	// ErrProviderNotExist is provider not exist error
	ErrProviderNotExist = errors.New("trpc/config: provider not exist")

	// ErrCodecNotExist is codec not exist error
	ErrCodecNotExist = errors.New("trpc/config: codec not exist")
)

func init() {
	RegisterCodec(&YamlCodec{})
	RegisterCodec(&JSONCodec{})
	RegisterCodec(&TomlCodec{})
}

// LoadOption defines the option function for loading configuration.
type LoadOption func(*TrpcConfig)

// TrpcConfigLoader is a config loader for trpc.
type TrpcConfigLoader struct {
	configMap map[string]Config
	rwl       sync.RWMutex
}

// Load returns the config specified by input parameter.
func (loader *TrpcConfigLoader) Load(path string, opts ...LoadOption) (Config, error) {
	yc := newTrpcConfig(path)
	for _, o := range opts {
		o(yc)
	}
	if yc.decoder == nil {
		return nil, ErrCodecNotExist
	}
	if yc.p == nil {
		return nil, ErrProviderNotExist
	}

	key := fmt.Sprintf("%s.%s.%s", yc.decoder.Name(), yc.p.Name(), path)
	loader.rwl.RLock()
	if c, ok := loader.configMap[key]; ok {
		loader.rwl.RUnlock()
		return c, nil
	}
	loader.rwl.RUnlock()

	if err := yc.Load(); err != nil {
		return nil, err
	}

	loader.rwl.Lock()
	loader.configMap[key] = yc
	loader.rwl.Unlock()

	yc.p.Watch(func(p string, data []byte) {
		if p == path {
			loader.rwl.Lock()
			delete(loader.configMap, key)
			loader.rwl.Unlock()
		}
	})
	return yc, nil
}

// Reload reloads config data.
func (loader *TrpcConfigLoader) Reload(path string, opts ...LoadOption) error {
	yc := newTrpcConfig(path)
	for _, o := range opts {
		o(yc)
	}
	key := fmt.Sprintf("%s.%s.%s", yc.decoder.Name(), yc.p.Name(), path)
	loader.rwl.RLock()
	if config, ok := loader.configMap[key]; ok {
		loader.rwl.RUnlock()
		config.Reload()
		return nil
	}
	loader.rwl.RUnlock()
	return ErrConfigNotExist
}

func newTrpcConfigLoad() *TrpcConfigLoader {
	return &TrpcConfigLoader{configMap: map[string]Config{}, rwl: sync.RWMutex{}}
}

// DefaultConfigLoader is the default config loader.
var DefaultConfigLoader = newTrpcConfigLoad()

// YamlCodec is yaml codec.
type YamlCodec struct{}

// Name returns yaml codec's name.
func (*YamlCodec) Name() string {
	return "yaml"
}

// Unmarshal deserializes the in bytes into out parameter by yaml.
func (c *YamlCodec) Unmarshal(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, out)
}

// JSONCodec is json codec.
type JSONCodec struct{}

// Name returns json codec's name.
func (*JSONCodec) Name() string {
	return "json"
}

// Unmarshal deserializes the in bytes into out parameter by json.
func (c *JSONCodec) Unmarshal(in []byte, out interface{}) error {
	return json.Unmarshal(in, out)
}

// TomlCodec is toml codec.
type TomlCodec struct{}

// Name returns toml codec's name.
func (*TomlCodec) Name() string {
	return "toml"
}

// Unmarshal deserializes the in bytes into out parameter by toml.
func (c *TomlCodec) Unmarshal(in []byte, out interface{}) error {
	return toml.Unmarshal(in, out)
}

// TrpcConfig is used to parse yaml config file for trpc.
type TrpcConfig struct {
	p                DataProvider
	unmarshalledData interface{}
	path             string
	decoder          Codec
	rawData          []byte
	expandEnv        func([]byte) []byte
}

func newTrpcConfig(path string) *TrpcConfig {
	return &TrpcConfig{
		p:                GetProvider("file"),
		unmarshalledData: make(map[string]interface{}),
		path:             path,
		decoder:          &YamlCodec{},
		expandEnv:        func(bytes []byte) []byte { return bytes },
	}
}

// Unmarshal deserializes the config into input param.
func (c *TrpcConfig) Unmarshal(out interface{}) error {
	return c.decoder.Unmarshal(c.rawData, out)
}

// Load loads config.
func (c *TrpcConfig) Load() error {
	if c.p == nil {
		return ErrProviderNotExist
	}

	data, err := c.p.Read(c.path)
	if err != nil {
		return fmt.Errorf("trpc/config: failed to load %s: %s", c.path, err.Error())
	}

	c.rawData = c.expandEnv(data)
	if err := c.decoder.Unmarshal(c.rawData, &c.unmarshalledData); err != nil {
		return fmt.Errorf("trpc/config: failed to parse %s: %s", c.path, err.Error())
	}
	return nil
}

// Reload reloads config.
func (c *TrpcConfig) Reload() {
	if c.p == nil {
		return
	}

	data, err := c.p.Read(c.path)
	if err != nil {
		log.Tracef("trpc/config: failed to reload %s: %v", c.path, err)
		return
	}

	c.rawData = c.expandEnv(data)
	if err := c.decoder.Unmarshal(c.rawData, &c.unmarshalledData); err != nil {
		log.Tracef("trpc/config: failed to parse %s: %v", c.path, err)
		return
	}
}

// Get returns config value by key. If key is absent will return the default value.
func (c *TrpcConfig) Get(key string, defaultValue interface{}) interface{} {
	if v, ok := c.search(key); ok {
		return v
	}
	return defaultValue
}

// Bytes returns original config data as bytes.
func (c *TrpcConfig) Bytes() []byte {
	return c.rawData
}

// GetInt returns int value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetInt(key string, defaultValue int) int {
	return c.findWithDefaultValue(key, defaultValue).(int)
}

// GetInt32 returns int32 value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetInt32(key string, defaultValue int32) int32 {
	return c.findWithDefaultValue(key, defaultValue).(int32)
}

// GetInt64 returns int64 value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetInt64(key string, defaultValue int64) int64 {
	return c.findWithDefaultValue(key, defaultValue).(int64)
}

// GetUint returns uint value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetUint(key string, defaultValue uint) uint {
	return c.findWithDefaultValue(key, defaultValue).(uint)
}

// GetUint32 returns uint32 value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetUint32(key string, defaultValue uint32) uint32 {
	return c.findWithDefaultValue(key, defaultValue).(uint32)
}

// GetUint64 returns uint64 value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetUint64(key string, defaultValue uint64) uint64 {
	return c.findWithDefaultValue(key, defaultValue).(uint64)
}

// GetFloat64 returns float64 value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetFloat64(key string, defaultValue float64) float64 {
	return c.findWithDefaultValue(key, defaultValue).(float64)
}

// GetFloat32 returns float32 value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetFloat32(key string, defaultValue float32) float32 {
	return c.findWithDefaultValue(key, defaultValue).(float32)
}

// GetBool returns bool value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetBool(key string, defaultValue bool) bool {
	return c.findWithDefaultValue(key, defaultValue).(bool)
}

// GetString returns string value by key, the second parameter
// is default value when key is absent or type conversion fails.
func (c *TrpcConfig) GetString(key string, defaultValue string) string {
	return c.findWithDefaultValue(key, defaultValue).(string)
}

// IsSet returns if the config specified by key exists.
func (c *TrpcConfig) IsSet(key string) bool {
	_, ok := c.search(key)
	return ok
}

// findWithDefaultValue ensures that the type of `value` is same as `defaultValue`
func (c *TrpcConfig) findWithDefaultValue(key string, defaultValue interface{}) (value interface{}) {
	v, ok := c.search(key)
	if !ok {
		return defaultValue
	}

	var err error
	switch defaultValue.(type) {
	case bool:
		v, err = cast.ToBoolE(v)
	case string:
		v, err = cast.ToStringE(v)
	case int:
		v, err = cast.ToIntE(v)
	case int32:
		v, err = cast.ToInt32E(v)
	case int64:
		v, err = cast.ToInt64E(v)
	case uint:
		v, err = cast.ToUintE(v)
	case uint32:
		v, err = cast.ToUint32E(v)
	case uint64:
		v, err = cast.ToUint64E(v)
	case float64:
		v, err = cast.ToFloat64E(v)
	case float32:
		v, err = cast.ToFloat32E(v)
	default:
	}

	if err != nil {
		return defaultValue
	}
	return v
}

func (c *TrpcConfig) search(key string) (interface{}, bool) {
	unmarshalledData, ok := c.unmarshalledData.(map[string]interface{})
	if !ok {
		return nil, false
	}

	subkeys := strings.Split(key, ".")
	value, err := search(unmarshalledData, subkeys)
	if err != nil {
		log.Debugf("trpc config: search key %s failed: %+v", key, err)
		return value, false
	}

	return value, true
}

func search(unmarshalledData map[string]interface{}, keys []string) (interface{}, error) {
	if len(keys) == 0 {
		return nil, ErrConfigNotExist
	}

	key, ok := unmarshalledData[keys[0]]
	if !ok {
		return nil, ErrConfigNotExist
	}

	if len(keys) == 1 {
		return key, nil
	}
	switch key := key.(type) {
	case map[interface{}]interface{}:
		return search(cast.ToStringMap(key), keys[1:])
	case map[string]interface{}:
		return search(key, keys[1:])
	default:
		return nil, ErrConfigNotExist
	}
}
