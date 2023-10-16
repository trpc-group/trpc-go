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

package log

import (
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/plugin"
)

func init() {
	RegisterWriter(OutputConsole, DefaultConsoleWriterFactory)
	RegisterWriter(OutputFile, DefaultFileWriterFactory)
	Register(defaultLoggerName, NewZapLog(defaultConfig))
	plugin.Register(defaultLoggerName, DefaultLogFactory)
}

const (
	pluginType        = "log"
	defaultLoggerName = "default"
)

var (
	// DefaultLogger the default Logger. The initial output is console. When frame start, it is
	// over write by configuration.
	DefaultLogger Logger
	// DefaultLogFactory is the default log loader. Users may replace it with their own
	// implementation.
	DefaultLogFactory = &Factory{}

	mu      sync.RWMutex
	loggers = make(map[string]Logger)
)

// Register registers Logger. It supports multiple Logger implementation.
func Register(name string, logger Logger) {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		panic("log: Register logger is nil")
	}
	if _, dup := loggers[name]; dup && name != defaultLoggerName {
		panic("log: Register called twiced for logger name " + name)
	}
	loggers[name] = logger
	if name == defaultLoggerName {
		DefaultLogger = logger
	}
}

// GetDefaultLogger gets the default Logger.
// To configure it, set key in configuration file to default.
// The console output is the default value.
func GetDefaultLogger() Logger {
	mu.RLock()
	l := DefaultLogger
	mu.RUnlock()
	return l
}

// SetLogger sets the default Logger.
func SetLogger(logger Logger) {
	mu.Lock()
	DefaultLogger = logger
	mu.Unlock()
}

// Get returns the Logger implementation by log name.
// log.Debug use DefaultLogger to print logs. You may also use log.Get("name").Debug.
func Get(name string) Logger {
	mu.RLock()
	l := loggers[name]
	mu.RUnlock()
	return l
}

// Sync syncs all registered loggers.
func Sync() {
	mu.RLock()
	defer mu.RUnlock()
	for _, logger := range loggers {
		_ = logger.Sync()
	}
}

// Decoder decodes the log.
type Decoder struct {
	OutputConfig *OutputConfig
	Core         zapcore.Core
	ZapLevel     zap.AtomicLevel
}

// Decode decodes writer configuration, copy one.
func (d *Decoder) Decode(cfg interface{}) error {
	output, ok := cfg.(**OutputConfig)
	if !ok {
		return fmt.Errorf("decoder config type:%T invalid, not **OutputConfig", cfg)
	}
	*output = d.OutputConfig
	return nil
}

// Factory is the log plugin factory.
// When server start, the configuration is feed to Factory to generate a log instance.
type Factory struct{}

// Type returns the log plugin type.
func (f *Factory) Type() string {
	return pluginType
}

// Setup starts, load and register logs.
func (f *Factory) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("log config decoder empty")
	}
	cfg, callerSkip, err := f.setupConfig(dec)
	if err != nil {
		return err
	}
	logger := NewZapLogWithCallerSkip(cfg, callerSkip)
	if logger == nil {
		return errors.New("new zap logger fail")
	}
	Register(name, logger)
	return nil
}

func (f *Factory) setupConfig(configDec plugin.Decoder) (Config, int, error) {
	cfg := Config{}
	if err := configDec.Decode(&cfg); err != nil {
		return nil, 0, err
	}
	if len(cfg) == 0 {
		return nil, 0, errors.New("log config output empty")
	}

	// If caller skip is not configured, use 2 as default.
	callerSkip := 2
	for i := 0; i < len(cfg); i++ {
		if cfg[i].CallerSkip != 0 {
			callerSkip = cfg[i].CallerSkip
		}
	}
	return cfg, callerSkip, nil
}
