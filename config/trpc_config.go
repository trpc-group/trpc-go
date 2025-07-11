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
	"trpc.group/trpc-go/trpc-go/internal/expandenv"

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
	watchers sync.Map
}

// Load returns the config specified by input parameter.
func (loader *TrpcConfigLoader) Load(path string, opts ...LoadOption) (Config, error) {
	c, err := newTrpcConfig(path, opts...)
	if err != nil {
		return nil, err
	}

	w := &watcher{}
	i, loaded := loader.watchers.LoadOrStore(c.p, w)
	if !loaded {
		c.p.Watch(w.watch)
	} else {
		w = i.(*watcher)
	}

	c = w.getOrCreate(c.path).getOrStore(c)
	if err = c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

// Reload reloads config data.
func (loader *TrpcConfigLoader) Reload(path string, opts ...LoadOption) error {
	c, err := newTrpcConfig(path, opts...)
	if err != nil {
		return err
	}

	v, ok := loader.watchers.Load(c.p)
	if !ok {
		return ErrConfigNotExist
	}
	w := v.(*watcher)

	s := w.get(path)
	if s == nil {
		return ErrConfigNotExist
	}

	oc := s.get(c.id)
	if oc == nil {
		return ErrConfigNotExist
	}

	return oc.Load()
}

func newTrpcConfigLoad() *TrpcConfigLoader {
	return &TrpcConfigLoader{}
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

// watch manage one data provider
type watcher struct {
	sets sync.Map // *set
}

// get config item by path
func (w *watcher) get(path string) *set {
	if i, ok := w.sets.Load(path); ok {
		return i.(*set)
	}
	return nil
}

// getOrCreate get config item by path if not exist and create and return
func (w *watcher) getOrCreate(path string) *set {
	i, _ := w.sets.LoadOrStore(path, &set{})
	return i.(*set)
}

// watch func
func (w *watcher) watch(path string, data []byte) {
	if v := w.get(path); v != nil {
		v.watch(data)
	}
}

// set manages configs with same provider and name with different type
// used config.id as unique identifier
type set struct {
	path  string
	mutex sync.RWMutex
	items []*TrpcConfig
}

// get data
func (s *set) get(id string) *TrpcConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, v := range s.items {
		if v.id == id {
			return v
		}
	}
	return nil
}

func (s *set) getOrStore(tc *TrpcConfig) *TrpcConfig {
	if v := s.get(tc.id); v != nil {
		return v
	}

	s.mutex.Lock()
	for _, item := range s.items {
		if item.id == tc.id {
			s.mutex.Unlock()
			return item
		}
	}
	// not found and add
	s.items = append(s.items, tc)
	s.mutex.Unlock()
	return tc
}

// watch data change, delete no watch model config and update watch model config and target notify
func (s *set) watch(data []byte) {
	var items []*TrpcConfig
	var del []*TrpcConfig
	s.mutex.Lock()
	for _, v := range s.items {
		if v.watch {
			items = append(items, v)
		} else {
			del = append(del, v)
		}
	}
	s.items = items
	s.mutex.Unlock()

	for _, item := range items {
		err := item.doWatch(data)
		item.notify(data, err)
	}

	for _, item := range del {
		item.notify(data, nil)
	}
}

// defaultNotifyChange default hook for notify config changed
var defaultWatchHook = func(message WatchMessage) {}

// SetDefaultWatchHook set default hook notify when config changed
func SetDefaultWatchHook(f func(message WatchMessage)) {
	defaultWatchHook = f
}

// WatchMessage change message
type WatchMessage struct {
	Provider  string // provider name
	Path      string // config path
	ExpandEnv bool   // expend env status
	Codec     string // codec
	Watch     bool   // status for start watch
	Value     []byte // config content diff ?
	Error     error  // load error message, success is empty string
}

var _ Config = (*TrpcConfig)(nil)

// TrpcConfig is used to parse yaml config file for trpc.
type TrpcConfig struct {
	id  string       // config identity
	msg WatchMessage // new to init message for notify only copy

	p         DataProvider // config provider
	path      string       // config name
	decoder   Codec        // config codec
	expandEnv bool         // status for whether replace the variables in the configuration with environment variables

	// because function is not support comparable in singleton, so the following options work only for the first load
	watch     bool
	watchHook func(message WatchMessage)

	mutex sync.RWMutex
	value *entity // store config value
}

type entity struct {
	raw  []byte      // current binary data
	data interface{} // unmarshal type to use point type, save latest no error data
}

func newEntity() *entity {
	return &entity{
		data: make(map[string]interface{}),
	}
}

func newTrpcConfig(path string, opts ...LoadOption) (*TrpcConfig, error) {
	c := &TrpcConfig{
		path:    path,
		p:       GetProvider("file"),
		decoder: GetCodec("yaml"),
		watchHook: func(message WatchMessage) {
			defaultWatchHook(message)
		},
	}
	for _, o := range opts {
		o(c)
	}
	if c.p == nil {
		return nil, ErrProviderNotExist
	}
	if c.decoder == nil {
		return nil, ErrCodecNotExist
	}

	c.msg.Provider = c.p.Name()
	c.msg.Path = c.path
	c.msg.Codec = c.decoder.Name()
	c.msg.ExpandEnv = c.expandEnv
	c.msg.Watch = c.watch

	// since reflect.String() cannot uniquely identify a type, this id is used as a preliminary judgment basis
	const idFormat = "provider:%s path:%s codec:%s env:%t watch:%t"
	c.id = fmt.Sprintf(idFormat, c.p.Name(), c.path, c.decoder.Name(), c.expandEnv, c.watch)
	return c, nil
}

func (c *TrpcConfig) get() *entity {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.value != nil {
		return c.value
	}
	return newEntity()
}

// init return config entity error when entity is empty and load run loads config once
func (c *TrpcConfig) init() error {
	c.mutex.RLock()
	if c.value != nil {
		c.mutex.RUnlock()
		return nil
	}
	c.mutex.RUnlock()

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.value != nil {
		return nil
	}

	data, err := c.p.Read(c.path)
	if err != nil {
		return fmt.Errorf("trpc/config failed to load error: %w config id: %s", err, c.id)
	}
	return c.set(data)
}
func (c *TrpcConfig) doWatch(data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.set(data)
}
func (c *TrpcConfig) set(data []byte) error {
	if c.expandEnv {
		data = expandenv.ExpandEnv(data)
	}

	e := newEntity()
	e.raw = data
	err := c.decoder.Unmarshal(data, &e.data)
	if err != nil {
		return fmt.Errorf("trpc/config: failed to parse:%w, id:%s", err, c.id)
	}
	c.value = e
	return nil
}
func (c *TrpcConfig) notify(data []byte, err error) {
	m := c.msg

	m.Value = data
	if err != nil {
		m.Error = err
	}

	c.watchHook(m)
}

// Load loads config.
func (c *TrpcConfig) Load() error {
	if c.p == nil {
		return ErrProviderNotExist
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	data, err := c.p.Read(c.path)
	if err != nil {
		return fmt.Errorf("trpc/config failed to load error: %w config id: %s", err, c.id)
	}

	return c.set(data)
}

// Reload reloads config.
func (c *TrpcConfig) Reload() {
	if err := c.Load(); err != nil {
		log.Tracef("trpc/config: failed to reload %s: %v", c.id, err)
	}
}

// Get returns config value by key. If key is absent will return the default value.
func (c *TrpcConfig) Get(key string, defaultValue interface{}) interface{} {
	if v, ok := c.search(key); ok {
		return v
	}
	return defaultValue
}

// Unmarshal deserializes the config into input param.
func (c *TrpcConfig) Unmarshal(out interface{}) error {
	return c.decoder.Unmarshal(c.get().raw, out)
}

// Bytes returns original config data as bytes.
func (c *TrpcConfig) Bytes() []byte {
	return c.get().raw
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
	e := c.get()

	unmarshalledData, ok := e.data.(map[string]interface{})
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
