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

// Package config provides common config interface.
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/BurntSushi/toml"

	yaml "gopkg.in/yaml.v3"
)

// ErrConfigNotSupport is not supported config error
var ErrConfigNotSupport = errors.New("trpc/config: not support")

// GetString returns string value get from
// kv storage by key.
func GetString(key string) (string, error) {
	val, err := globalKV.Get(context.Background(), key)
	if err != nil {
		return "", err
	}
	return val.Value(), nil
}

// GetStringWithDefault returns string value get by key.
// If anything wrong, returns default value specified by input param def.
func GetStringWithDefault(key, def string) string {
	val, err := globalKV.Get(context.Background(), key)
	if err != nil {
		return def
	}
	return val.Value()
}

// GetInt returns int value get by key.
func GetInt(key string) (int, error) {
	val, err := globalKV.Get(context.Background(), key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val.Value())
}

// GetIntWithDefault returns int value get by key.
// If anything wrong, returns default value specified by input param def.
func GetIntWithDefault(key string, def int) int {
	val, err := globalKV.Get(context.Background(), key)
	if err != nil {
		return def
	}
	i, err := strconv.Atoi(val.Value())
	if err != nil {
		return def
	}
	return i
}

// GetWithUnmarshal gets the specific encoding data
// by key. the encoding type is defined by unmarshalName parameter.
func GetWithUnmarshal(key string, val interface{}, unmarshalName string) error {
	v, err := globalKV.Get(context.Background(), key)
	if err != nil {
		return err
	}
	return GetUnmarshaler(unmarshalName).Unmarshal([]byte(v.Value()), val)
}

// GetWithUnmarshalProvider gets the specific encoding data by key
// the encoding type is defined by unmarshalName parameter
// the provider name is defined by provider parameter.
func GetWithUnmarshalProvider(key string, val interface{}, unmarshalName string, provider string) error {
	p := Get(provider)
	if p == nil {
		return fmt.Errorf("trpc/config: failed to get %s", provider)
	}
	v, err := p.Get(context.Background(), key)
	if err != nil {
		return err
	}
	return GetUnmarshaler(unmarshalName).Unmarshal([]byte(v.Value()), val)
}

// GetJSON gets json data by key. The value will unmarshal into val parameter.
func GetJSON(key string, val interface{}) error {
	return GetWithUnmarshal(key, val, "json")
}

// GetJSONWithProvider gets json data by key. The value will unmarshal into val parameter
// the provider name is defined by provider parameter.
func GetJSONWithProvider(key string, val interface{}, provider string) error {
	return GetWithUnmarshalProvider(key, val, "json", provider)
}

// GetYAML gets yaml data by key. The value will unmarshal into val parameter.
func GetYAML(key string, val interface{}) error {
	return GetWithUnmarshal(key, val, "yaml")
}

// GetYAMLWithProvider gets yaml data by key. The value will unmarshal into val parameter
// the provider name is defined by provider parameter.
func GetYAMLWithProvider(key string, val interface{}, provider string) error {
	return GetWithUnmarshalProvider(key, val, "yaml", provider)
}

// GetTOML gets toml data by key. The value will unmarshal into val parameter.
func GetTOML(key string, val interface{}) error {
	return GetWithUnmarshal(key, val, "toml")
}

// GetTOMLWithProvider gets toml data by key. The value will unmarshal into val parameter
// the provider name is defined by provider parameter.
func GetTOMLWithProvider(key string, val interface{}, provider string) error {
	return GetWithUnmarshalProvider(key, val, "toml", provider)
}

// Unmarshaler defines a unmarshal interface, this will
// be used to parse config data.
type Unmarshaler interface {
	// Unmarshal deserializes the data bytes into value parameter.
	Unmarshal(data []byte, value interface{}) error
}

var (
	unmarshalers = make(map[string]Unmarshaler)
)

// YamlUnmarshaler is yaml unmarshaler.
type YamlUnmarshaler struct{}

// Unmarshal deserializes the data bytes into parameter val in yaml protocol.
func (yu *YamlUnmarshaler) Unmarshal(data []byte, val interface{}) error {
	return yaml.Unmarshal(data, val)
}

// JSONUnmarshaler is json unmarshaler.
type JSONUnmarshaler struct{}

// Unmarshal deserializes the data bytes into parameter val in json protocol.
func (ju *JSONUnmarshaler) Unmarshal(data []byte, val interface{}) error {
	return json.Unmarshal(data, val)
}

// TomlUnmarshaler is toml unmarshaler.
type TomlUnmarshaler struct{}

// Unmarshal deserializes the data bytes into parameter val in toml protocol.
func (tu *TomlUnmarshaler) Unmarshal(data []byte, val interface{}) error {
	return toml.Unmarshal(data, val)
}

func init() {
	RegisterUnmarshaler("yaml", &YamlUnmarshaler{})
	RegisterUnmarshaler("json", &JSONUnmarshaler{})
	RegisterUnmarshaler("toml", &TomlUnmarshaler{})
}

// RegisterUnmarshaler registers an unmarshaler by name.
func RegisterUnmarshaler(name string, us Unmarshaler) {
	unmarshalers[name] = us
}

// GetUnmarshaler returns an unmarshaler by name.
func GetUnmarshaler(name string) Unmarshaler {
	return unmarshalers[name]
}

var (
	configMap = make(map[string]KVConfig)
)

// KVConfig defines a kv config interface.
type KVConfig interface {
	KV
	Watcher
	Name() string
}

// Register registers a kv config by its name.
func Register(c KVConfig) {
	lock.Lock()
	configMap[c.Name()] = c
	lock.Unlock()
}

// Get returns a kv config by name.
func Get(name string) KVConfig {
	lock.RLock()
	c := configMap[name]
	lock.RUnlock()
	return c
}

// GlobalKV returns an instance of kv config center.
func GlobalKV() KV {
	return globalKV
}

// SetGlobalKV sets the instance of kv config center.
func SetGlobalKV(kv KV) {
	globalKV = kv
}

// EventType defines the event type of config change.
type EventType uint8

const (
	// EventTypeNull represents null event.
	EventTypeNull EventType = 0

	// EventTypePut represents set or update config event.
	EventTypePut EventType = 1

	// EventTypeDel represents delete config event.
	EventTypeDel EventType = 2
)

// Response defines config center's response interface.
type Response interface {
	// Value returns config value as string.
	Value() string

	// MetaData returns extra metadata. With option,
	// we can implement some extra features for different config center,
	// such as namespace, group, lease, etc.
	MetaData() map[string]string

	// Event returns the type of watch event.
	Event() EventType
}

// KV defines a kv storage for config center.
type KV interface {
	// Put puts or updates config value by key.
	Put(ctx context.Context, key, val string, opts ...Option) error

	// Get returns config value by key.
	Get(ctx context.Context, key string, opts ...Option) (Response, error)

	// Del deletes config value by key.
	Del(ctx context.Context, key string, opts ...Option) error
}

// Watcher defines the interface of config center watch event.
type Watcher interface {
	// Watch watches the config key change event.
	Watch(ctx context.Context, key string, opts ...Option) (<-chan Response, error)
}

var globalKV KV = &noopKV{}

// noopKV is an empty implementation of KV interface.
type noopKV struct{}

// Put does nothing but returns nil.
func (kv *noopKV) Put(ctx context.Context, key, val string, opts ...Option) error {
	return nil
}

// Get returns not supported error.
func (kv *noopKV) Get(ctx context.Context, key string, opts ...Option) (Response, error) {
	return nil, ErrConfigNotSupport
}

// Del does nothing but returns nil.
func (kv *noopKV) Del(ctx context.Context, key string, opts ...Option) error {
	return nil
}

// Config defines the common config interface. We can
// implement different config center by this interface.
type Config interface {
	// Load loads config.
	Load() error

	// Reload reloads config.
	Reload()

	// Get returns config by key.
	Get(string, interface{}) interface{}

	// Unmarshal deserializes the config into input param.
	Unmarshal(interface{}) error

	// IsSet returns if the config specified by key exists.
	IsSet(string) bool

	// GetInt returns int value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetInt(string, int) int

	// GetInt32 returns int32 value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetInt32(string, int32) int32

	// GetInt64 returns int64 value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetInt64(string, int64) int64

	// GetUint returns uint value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetUint(string, uint) uint

	// GetUint32 returns uint32 value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetUint32(string, uint32) uint32

	// GetUint64 returns uint64 value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetUint64(string, uint64) uint64

	// GetFloat32 returns float32 value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetFloat32(string, float32) float32

	// GetFloat64 returns float64 value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetFloat64(string, float64) float64

	// GetString returns string value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetString(string, string) string

	// GetBool returns bool value by key, the second parameter
	// is default value when key is absent or type conversion fails.
	GetBool(string, bool) bool

	// Bytes returns config data as bytes.
	Bytes() []byte
}

// ProviderCallback is callback function for provider to handle
// config change.
type ProviderCallback func(string, []byte)

// DataProvider defines common data provider interface.
// we can implement this interface to define different
// data provider( such as file, TConf, ETCD, configmap)
// and parse config data to standard format( such as json,
// toml, yaml, etc.) by codec.
type DataProvider interface {
	// Name returns the data provider's name.
	Name() string

	// Read reads the specific path file, returns
	// it content as bytes.
	Read(string) ([]byte, error)

	// Watch watches config changing. The change will
	// be handled by callback function.
	Watch(ProviderCallback)
}

// Codec defines codec interface.
type Codec interface {

	// Name returns codec's name.
	Name() string

	// Unmarshal deserializes the config data bytes into
	// the second input parameter.
	Unmarshal([]byte, interface{}) error
}

var providerMap = make(map[string]DataProvider)

// RegisterProvider registers a data provider by its name.
func RegisterProvider(p DataProvider) {
	providerMap[p.Name()] = p
}

// GetProvider returns the provider by name.
func GetProvider(name string) DataProvider {
	return providerMap[name]
}

var (
	codecMap = make(map[string]Codec)
	lock     = sync.RWMutex{}
)

// RegisterCodec registers codec by its name.
func RegisterCodec(c Codec) {
	lock.Lock()
	codecMap[c.Name()] = c
	lock.Unlock()
}

// GetCodec returns the codec by name.
func GetCodec(name string) Codec {
	lock.RLock()
	c := codecMap[name]
	lock.RUnlock()
	return c
}

// Load returns the config specified by input parameter.
func Load(path string, opts ...LoadOption) (Config, error) {
	return DefaultConfigLoader.Load(path, opts...)
}

// Reload reloads config data.
func Reload(path string, opts ...LoadOption) error {
	return DefaultConfigLoader.Reload(path, opts...)
}
