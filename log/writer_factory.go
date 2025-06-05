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
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/plugin"
)

var (
	// DefaultConsoleWriterFactory is the default console output implementation.
	DefaultConsoleWriterFactory = &ConsoleWriterFactory{}
	// DefaultFileWriterFactory is the default file output implementation.
	DefaultFileWriterFactory = &FileWriterFactory{}

	// Deprecated: use coreLevelNewers instead.
	// Because newers only be used by RegisterCoreNewer and GetCoreNewer, which both have been deprecated,
	// there is no reason to use it.
	newers          = make(map[string]CoreNewer)
	coreLevelNewers = make(map[string]CoreLevelNewer)
)

// RegisterWriter registers log output writer. Writer may have multiple implementations.
//
// Deprecated: use RegisterCoreLevelNewer instead.
// the type of the second input parameter of RegisterWriter is unreasonable,
// as it does not allow you to clearly know that you must set the log and zapCore.Core in the Setup method
// when implementing plugin.Factory. If the log level and zapCore are not set correctly in the Setup method,
// errors may occur when using the logger.
func RegisterWriter(name string, writer plugin.Factory) {
	coreLevelNewers[name] = &writerFactory{name: name, factory: writer}
}

// GetWriter gets log output writer, returns nil if not exist.
//
// Deprecated: use GetCoreLevelNewer instead.
// Because RegisterWriter has been deprecated, there is no reason to call GetWriter.
func GetWriter(name string) plugin.Factory {
	f, ok := coreLevelNewers[name].(*writerFactory)
	if !ok || f == nil {
		return nil
	}
	return f.factory
}

type writerFactory struct {
	name    string
	factory plugin.Factory
}

func (w *writerFactory) New(config OutputConfig) (zapcore.Core, error) {
	decoder := &Decoder{OutputConfig: &config, Core: zapcore.NewNopCore()}

	if err := w.factory.Setup(w.name, decoder); err != nil {
		return nil, fmt.Errorf("setting up %s failed: %v", w.name, err)
	}
	return decoder.Core, nil
}

// NewCoreLevel implements CoreLevelNewer interface.
func (w *writerFactory) NewCoreLevel(config OutputConfig) (zapcore.Core, zap.AtomicLevel, error) {
	decoder := &Decoder{OutputConfig: &config, Core: zapcore.NewNopCore(), ZapLevel: zap.NewAtomicLevel()}

	if err := w.factory.Setup(w.name, decoder); err != nil {
		return nil, zap.NewAtomicLevel(), fmt.Errorf("setting up %s failed: %v", w.name, err)
	}
	return decoder.Core, decoder.ZapLevel, nil
}

// RegisterCoreNewer registers a CoreNewer for log output writer with name.
// Deprecated: use RegisterCoreLevelNewer instead.
// Because CoreNewer.New does not return the level associated with the
// core, making it impossible to change the log level of the logger.
func RegisterCoreNewer(name string, newer CoreNewer) {
	newers[name] = newer
}

// GetCoreNewer returns a CoreNewer by name of log output writer.
// Deprecated: use GetCoreLevelNewer instead.
// Because RegisterCoreNewer has been deprecated, there is no reason to call GetCoreNewer.
func GetCoreNewer(name string) (CoreNewer, bool) {
	newer, ok := newers[name]
	return newer, ok
}

// CoreNewer is the interface that wraps the New method.
type CoreNewer interface {
	// New creates a zapcore.Core from OutputConfig.
	New(config OutputConfig) (zapcore.Core, error)
}

// RegisterCoreLevelNewer registers a CoreLevelNewer for log output writer with name.
func RegisterCoreLevelNewer(name string, newer CoreLevelNewer) {
	coreLevelNewers[name] = newer
}

// GetCoreLevelNewer returns a CoreLevelNewer by name of log output writer.
func GetCoreLevelNewer(name string) (CoreLevelNewer, bool) {
	newer, ok := coreLevelNewers[name]
	return newer, ok
}

// CoreLevelNewer is an interface that encapsulates the NewCoreLevel method.
// This interface has higher precedence than the embedded CoreNewer interface.
// To ensure a strong association between the returned core and level, users
// are strongly advised to implement this interface.
type CoreLevelNewer interface {
	// NewCoreLevel produces a zapcore.Core and yields the corresponding zap.AtomicLevel
	// from the OutputConfig.
	// The returned zap.AtomicLevel is required to maintain an intrinsic link with the returned zapcore.Core,
	// implying that any modifications to the returned zap.AtomicLevel will echo in the returned zapcore.Core,
	// as opposed to generating a transient one using zapcore.LevelOf(core).
	NewCoreLevel(config OutputConfig) (zapcore.Core, zap.AtomicLevel, error)
}

// ConsoleWriterFactory is the console writer instance.
type ConsoleWriterFactory struct {
}

// Type returns the log plugin type.
func (f *ConsoleWriterFactory) Type() string {
	return pluginType
}

// Setup starts, loads and registers console output writer.
func (f *ConsoleWriterFactory) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("console writer decoder empty")
	}
	decoder, ok := dec.(*Decoder)
	if !ok {
		return errors.New("console writer log decoder type invalid")
	}
	cfg := &OutputConfig{}
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}
	decoder.Core, decoder.ZapLevel = newConsoleCore(cfg)
	return nil
}

// FileWriterFactory is the file writer instance Factory.
type FileWriterFactory struct {
}

// Type returns log file type.
func (f *FileWriterFactory) Type() string {
	return pluginType
}

// Setup starts, loads and register file output writer.
func (f *FileWriterFactory) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("file writer decoder empty")
	}
	decoder, ok := dec.(*Decoder)
	if !ok {
		return errors.New("file writer log decoder type invalid")
	}
	if err := f.setupConfig(decoder); err != nil {
		return err
	}
	return nil
}

func (f *FileWriterFactory) setupConfig(decoder *Decoder) error {
	cfg := &OutputConfig{}
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}
	if cfg.WriteConfig.LogPath != "" {
		cfg.WriteConfig.Filename = filepath.Join(cfg.WriteConfig.LogPath, cfg.WriteConfig.Filename)
	}
	if cfg.WriteConfig.RollType == "" {
		cfg.WriteConfig.RollType = RollBySize
	}

	core, level, err := newFileCore(cfg)
	if err != nil {
		return err
	}
	decoder.Core, decoder.ZapLevel = core, level
	return nil
}
