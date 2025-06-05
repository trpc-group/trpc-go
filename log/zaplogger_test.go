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

package log_test

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/log/internal/timeunit"
	"trpc.group/trpc-go/trpc-go/plugin"
)

var defaultConfig = []log.OutputConfig{
	{
		Writer:    "console",
		Level:     "debug",
		Formatter: "console",
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006.01.02 15:04:05",
		},
	},
	{
		Writer:    "file",
		Level:     "info",
		Formatter: "json",
		WriteConfig: log.WriteConfig{
			Filename:   "trpc_size.log",
			RollType:   "size",
			MaxAge:     7,
			MaxBackups: 10,
			MaxSize:    100,
		},
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006.01.02 15:04:05",
		},
	},
	{
		Writer:    "file",
		Level:     "info",
		Formatter: "json",
		WriteConfig: log.WriteConfig{
			Filename:   "trpc_time.log",
			RollType:   "time",
			MaxAge:     7,
			MaxBackups: 10,
			MaxSize:    100,
			TimeUnit:   timeunit.Day,
		},
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006-01-02 15:04:05",
		},
	},
	{
		Writer:    "file",
		Level:     "debug",
		Formatter: "json",
		WriteConfig: log.WriteConfig{
			Filename:   "trpc_time.log",
			RollType:   "timeunit",
			MaxAge:     7,
			MaxBackups: 10,
			MaxSize:    100,
			TimeUnit:   "%Y-%m-%d-%H-%M",
		},
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006-01-02 15:04:05",
		},
	},
	{
		Writer:    "file",
		Level:     "info",
		Formatter: "json",
		WriteConfig: log.WriteConfig{
			Filename:   "trpc_{time_format}.log",
			RollType:   "timeunit",
			MaxBackups: 10,
			MaxSize:    100,
			TimeUnit:   "%Y-%m-%d-%H-%M",
		},
	},
}

func TestNewZapLog(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		logger := log.NewZapLog(defaultConfig)
		assert.NotNil(t, logger)

		logger.SetLevel("0", log.LevelInfo)
		lvl := logger.GetLevel("0")
		assert.Equal(t, lvl, log.LevelInfo)

		l := logger.WithFields("test", "a")
		if tmp, ok := l.(*log.ZapLogWrapper); ok {
			tmp.GetLogger()
			tmp.Sync()
		}
		l.SetLevel("output", log.LevelDebug)
		assert.Equal(t, log.LevelDebug, l.GetLevel("output"))
	})
	t.Run("coreNewer", func(t *testing.T) {
		const name = "coreNewer"
		buf := &buffer{}
		log.RegisterWriter(name, &coreWriter{ws: buf})
		c := []log.OutputConfig{
			{
				Writer: name,
				Level:  "warn",
			},
		}
		l := log.NewZapLog(c)

		l.Debug("debug")
		require.NotContains(t, buf.message(), "debug")

		l.Info("info")
		require.NotContains(t, buf.message(), "info")

		l.Warn("warn")
		require.Contains(t, buf.message(), "warn")

		l.Error("error")
		require.Contains(t, buf.message(), "error")
	})
	t.Run("levelWriter", func(t *testing.T) {
		const name = "levelWriter"
		buf := &buffer{}
		log.RegisterWriter(name, &levelWriter{ws: buf})
		c := []log.OutputConfig{
			{
				Writer: name,
				Level:  "warn",
			},
		}
		l := log.NewZapLog(c)

		l.Debug("debug")
		require.NotContains(t, buf.message(), "debug")

		l.Info("info")
		require.NotContains(t, buf.message(), "info")

		l.Warn("warn")
		require.NotContains(t, buf.message(), "warn")

		l.Error("error")
		require.NotContains(t, buf.message(), "error")
	})
	t.Run("noopWriter", func(t *testing.T) {
		const name = "noopWriter"
		buf := &buffer{}
		log.RegisterWriter(name, &noopWriter{ws: buf})
		c := []log.OutputConfig{
			{
				Writer: name,
				Level:  "warn",
			},
		}
		l := log.NewZapLog(c)

		l.Debug("debug")
		require.NotContains(t, buf.message(), "debug")

		l.Info("info")
		require.NotContains(t, buf.message(), "info")

		l.Warn("warn")
		require.NotContains(t, buf.message(), "warn")

		l.Error("error")
		require.NotContains(t, buf.message(), "error")
	})
}

const pluginType = "log"

type coreWriter struct {
	ws zapcore.WriteSyncer
}

func (cw *coreWriter) Type() string {
	return pluginType
}

func (cw *coreWriter) Setup(_ string, decoder plugin.Decoder) error {
	var d = decoder.(*log.Decoder)
	c := &log.OutputConfig{}
	if err := d.Decode(&c); err != nil {
		return err
	}

	d.Core = zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(cw.ws),
		zap.NewAtomicLevelAt(log.Levels[c.Level]))
	// d.ZapLevel is not set.
	return nil
}

type levelWriter struct {
	ws zapcore.WriteSyncer
}

func (lw *levelWriter) Type() string {
	return pluginType
}

func (lw *levelWriter) Setup(_ string, decoder plugin.Decoder) error {
	var d = decoder.(*log.Decoder)
	c := &log.OutputConfig{}
	if err := d.Decode(&c); err != nil {
		return err
	}

	d.ZapLevel = zap.NewAtomicLevelAt(log.Levels[c.Level])
	// d.Core is not set.
	return nil
}

type noopWriter struct {
	ws zapcore.WriteSyncer
}

func (lw *noopWriter) Type() string {
	return pluginType
}

func (lw *noopWriter) Setup(_ string, decoder plugin.Decoder) error {
	var d = decoder.(*log.Decoder)
	c := &log.OutputConfig{}
	if err := d.Decode(&c); err != nil {
		return err
	}
	// d.ZapLevel is not set.
	// d.Core is not set.
	return nil
}

type buffer struct {
	buf bytes.Buffer
}

func (b *buffer) Sync() error {
	return nil
}

func (b *buffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func (b *buffer) message() string {
	return b.buf.String()
}

func TestNewZapLog_WriteMode(t *testing.T) {
	logDir := t.TempDir()
	t.Run("invalid write mode", func(t *testing.T) {
		const invalidWriteMode = 4
		require.Panics(t, func() {
			log.NewZapLog([]log.OutputConfig{{
				Writer: log.OutputFile,
				WriteConfig: log.WriteConfig{
					LogPath:   logDir,
					Filename:  "trpc.log",
					WriteMode: invalidWriteMode,
				},
			}})
		})
	})
	t.Run("valid write mode", func(t *testing.T) {
		const (
			syncFileName  = "trpc.syncLog"
			asyncFileName = "trpc.asyncLog"
			fastFileName  = "trpc.fastLog"
		)
		tests := []struct {
			name   string
			config log.OutputConfig
		}{
			{"sync", log.OutputConfig{
				Writer: log.OutputFile,
				WriteConfig: log.WriteConfig{
					LogPath:   logDir,
					Filename:  syncFileName,
					WriteMode: log.WriteSync,
				},
			}},
			{"async", log.OutputConfig{
				Writer: log.OutputFile,
				WriteConfig: log.WriteConfig{
					LogPath:   logDir,
					Filename:  asyncFileName,
					WriteMode: log.WriteAsync,
				},
			}},
			{"fast", log.OutputConfig{
				Writer: log.OutputFile,
				WriteConfig: log.WriteConfig{
					LogPath:   logDir,
					Filename:  fastFileName,
					WriteMode: log.WriteFast,
				},
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.NotNil(t, log.NewZapLog([]log.OutputConfig{tt.config}))
			})
		}
	})
}

func TestZapLogWithLevel(t *testing.T) {
	logger := log.NewZapLog(defaultConfig)
	assert.NotNil(t, logger)

	l := logger.WithFields("field1")
	l.SetLevel("0", log.LevelFatal)
	assert.Equal(t, log.LevelFatal, l.GetLevel("0"))

	l = l.With(log.Field{Key: "key1", Value: "val1"})
	l.SetLevel("0", log.LevelError)
	assert.Equal(t, log.LevelError, l.GetLevel("0"))
}

func BenchmarkDefaultTimeFormat(b *testing.B) {
	t := time.Now()
	for i := 0; i < b.N; i++ {
		log.DefaultTimeFormat(t)
	}
}

func BenchmarkCustomTimeFormat(b *testing.B) {
	t := time.Now()
	for i := 0; i < b.N; i++ {
		log.CustomTimeFormat(t, "2006-01-02 15:04:05.000")
	}
}

func TestCustomTimeFormat(t *testing.T) {
	date := time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local)
	dateStr := log.CustomTimeFormat(date, "2006-01-02 15:04:05.000")
	assert.Equal(t, dateStr, "2006-01-02 15:04:05.000")
}

func TestDefaultTimeFormat(t *testing.T) {
	date := time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local)
	dateStr := string(log.DefaultTimeFormat(date))
	assert.Equal(t, dateStr, "2006-01-02 15:04:05.000")
}

func TestGetLogEncoderKey(t *testing.T) {
	tests := []struct {
		name   string
		defKey string
		key    string
		want   string
	}{
		{"custom", "T", "Time", "Time"},
		{"default", "T", "", "T"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := log.GetLogEncoderKey(tt.defKey, tt.key); got != tt.want {
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func TestNewTimeEncoder(t *testing.T) {
	encoder := log.NewTimeEncoder("")
	assert.NotNil(t, encoder)

	encoder = log.NewTimeEncoder("2006-01-02 15:04:05")
	assert.NotNil(t, encoder)

	tests := []struct {
		name string
		fmt  string
	}{
		{"seconds timestamp", "seconds"},
		{"milliseconds timestamp", "milliseconds"},
		{"nanoseconds timestamp", "nanoseconds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := log.NewTimeEncoder(tt.fmt)
			assert.NotNil(t, got)
		})
	}
}

func TestWithFields(t *testing.T) {
	// register Writer.
	// use zap observer to support test.
	core, ob := observer.New(zap.InfoLevel)
	log.RegisterWriter(observewriter, &observeWriter{core: core})

	// config is configuration.
	cfg := []log.OutputConfig{
		{
			Writer: observewriter,
		},
	}

	// create a zap logger.
	zl := log.NewZapLog(cfg)
	assert.NotNil(t, zl)

	// test With.
	field := log.Field{Key: "abc", Value: int32(123)}
	logger := zl.With(field)
	assert.NotNil(t, logger)
	log.SetLogger(logger)
	log.Warn("with fields warning")
	assert.Equal(t, 1, ob.Len())
	entry := ob.All()[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "with fields warning", entry.Message)
	assert.Equal(t, []zapcore.Field{{Key: "abc", Type: zapcore.Int32Type, Integer: 123}}, entry.Context)
}

func TestOptionLogger(t *testing.T) {
	t.Run("test option logger add caller skip", func(t *testing.T) {
		core, ob := observer.New(zap.InfoLevel)
		log.RegisterWriter(observewriter, &observeWriter{core: core})
		cfg := []log.OutputConfig{{Writer: observewriter}}

		l := log.NewZapLogWithCallerSkip(cfg, 1)
		l.Info("this is option logger test, the current caller skip is correct")

		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		require.Equal(t, file, ob.All()[0].Caller.File)

		ol, ok := l.(log.OptionLogger)
		require.True(t, ok)
		l = ol.WithOptions(log.WithAdditionalCallerSkip(1))
		l.Info("this is option logger test, the current caller skip is incorrect(added 1)")

		_, file, _, ok = runtime.Caller(1)
		require.True(t, ok)
		require.Equal(t, file, ob.All()[1].Caller.File)
	})
	t.Run("test option logger wrapper add caller skip", func(t *testing.T) {
		core, ob := observer.New(zap.InfoLevel)
		log.RegisterWriter(observewriter, &observeWriter{core: core})
		cfg := []log.OutputConfig{{Writer: observewriter}}

		l := log.NewZapLogWithCallerSkip(cfg, 2)
		l = l.With(log.Field{Key: "k", Value: "v"})
		l.Info("this is option logger wrapper test, the current caller skip is correct")

		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		require.Equal(t, file, ob.All()[0].Caller.File)

		ol, ok := l.(log.OptionLogger)
		require.True(t, ok)
		l = ol.WithOptions(log.WithAdditionalCallerSkip(1))
		l.Info("this is option logger wrapper test, the current caller skip is incorrect(added 1)")

		_, file, _, ok = runtime.Caller(1)
		require.True(t, ok)
		require.Equal(t, file, ob.All()[1].Caller.File)
	})
}

const observewriter = "observewriter"

type observeWriter struct {
	core  zapcore.Core
	level zap.AtomicLevel
}

func (f *observeWriter) Type() string { return "log" }

func (f *observeWriter) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("empty decoder")
	}
	decoder, ok := dec.(*log.Decoder)
	if !ok {
		return errors.New("invalid decoder")
	}
	decoder.Core = f.core
	decoder.ZapLevel = f.level
	return nil
}

func TestLogLevel(t *testing.T) {
	t.Run("test log level", func(t *testing.T) {
		config := []log.OutputConfig{
			{
				Writer: "console",
				Level:  "",
			},
			{
				Writer: "console",
				Level:  "trace",
			},
			{
				Writer: "console",
				Level:  "debug",
			},
			{
				Writer: "console",
				Level:  "info",
			},
			{
				Writer: "console",
				Level:  "warn",
			},
			{
				Writer: "console",
				Level:  "error",
			},
			{
				Writer: "console",
				Level:  "fatal",
			},
		}
		l := log.NewZapLog(config)

		got := make([]string, 0, len(config))
		want := make([]string, 0, len(config))
		for i, c := range config {
			got = append(got, log.LevelStrings[l.GetLevel(fmt.Sprint(i))])
			want = append(want, log.Levels[c.Level].String())
		}
		require.Equal(t, want, got)
	})
	t.Run("test actual log output by setting log level", func(t *testing.T) {
		level := zap.NewAtomicLevelAt(zap.InfoLevel)
		core, ob := observer.New(&level)
		log.RegisterWriter(observewriter, &observeWriter{core: core, level: level})
		cfg := []log.OutputConfig{{Writer: observewriter}}
		l := log.NewZapLog(cfg)
		debugMsg := "this is a debug level log"
		infoMsg := "this is a info level log"

		// only info log, because log level is info
		l.Info(infoMsg)
		l.Debug(debugMsg)
		require.Equal(t, 1, len(ob.All()))
		require.Equal(t, infoMsg, ob.All()[0].Entry.Message)

		// set log level to debug
		// have info and debug level log
		l.SetLevel("0", log.LevelDebug)
		l.Info(infoMsg)
		l.Debug(debugMsg)
		require.Equal(t, 3, len(ob.All()))
		require.Equal(t, infoMsg, ob.All()[0].Entry.Message)
		require.Equal(t, infoMsg, ob.All()[1].Entry.Message)
		require.Equal(t, debugMsg, ob.All()[2].Entry.Message)
	})
}

func TestLogEnableColor(t *testing.T) {
	cfg := []log.OutputConfig{{Writer: "console", Level: "trace", EnableColor: true}}
	l := log.NewZapLog(cfg)
	l.Trace("hello")
	l.Debug("hello")
	l.Info("hello")
	l.Warn("hello")
	l.Error("hello")
}

func TestNewZapLogDisableField(t *testing.T) {
	logger := log.NewZapLog([]log.OutputConfig{{
		Writer:    "console",
		Level:     "debug",
		Formatter: "console",
		FormatConfig: log.FormatConfig{
			TimeKey:       "none",
			LevelKey:      "none",
			NameKey:       "none",
			CallerKey:     "none",
			FunctionKey:   "none",
			StacktraceKey: "none",
		},
	}})
	logger.Debugf("hello")
}

func TestLogWithAdhocFields(t *testing.T) {
	cfg := []log.OutputConfig{{Writer: "console", Level: "trace"}}
	l := log.NewZapLogWithCallerSkip(cfg, 1)
	l.Trace("hello", zap.String("key", "value"))
	l.Debug("hello", zap.String("key", "value"))
	l.Info("hello", zap.String("key", "value"))
	l.Warn("hello", zap.String("key", "value"))
	l.Error("hello", zap.String("key", "value"))
	l.Tracef("hello", zap.String("key", "value"))
	l.Debugf("hello", zap.String("key", "value"))
	l.Infof("hello", zap.String("key", "value"))
	l.Warnf("hello", zap.String("key", "value"))
	l.Errorf("hello", zap.String("key", "value"))
	// 2023-12-15 10:24:37.038 DEBUG   log/zaplogger_test.go:587       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 DEBUG   log/zaplogger_test.go:588       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 INFO    log/zaplogger_test.go:589       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 WARN    log/zaplogger_test.go:590       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 ERROR   log/zaplogger_test.go:591       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 DEBUG   log/zaplogger_test.go:592       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 DEBUG   log/zaplogger_test.go:593       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 INFO    log/zaplogger_test.go:594       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 WARN    log/zaplogger_test.go:595       hello   {"key": "value"}
	// 2023-12-15 10:24:37.038 ERROR   log/zaplogger_test.go:596       hello   {"key": "value"}

	log.Infof("test format int %d", 6, zap.Any("key", "any"))
	log.Infof("test format int %d", zap.Any("key", "any"), 6)
	log.Infof("test format int %d", zap.Any("key", "any"), 6, zap.Binary("key", []byte("value")))
	log.Infof("test format int %d", zap.Any("key", "any"), 6, zap.Complex128("key", 4+5i))
	log.Infof("test format int %d", zap.Any("key", "any"), 6, zap.Bool("key", true), zap.Duration("key", time.Second))
	log.Infof("test format int %d and string %s", 6, zap.ByteString("key", []byte("value")), "hh")
	log.Infof("test format int %d and string %s", zap.Int("key", 777), 6, "hh")
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:608       test format int 6       {"key": "any"}
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:609       test format int 6       {"key": "any"}
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:610       test format int 6       {"key": "any", "key": "dmFsdWU="}
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:611       test format int 6       {"key": "any", "key": "4+5i"}
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:612       test format int 6       {"key": "any", "key": true, "key": "1s"}
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:613       test format int 6 and string hh {"key": "value"}
	// 2023-12-15 16:38:01.896 INFO    log/zaplogger_test.go:614       test format int 6 and string hh {"key": 777}
}

func TestLogWithName(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	core, ob := observer.New(&level)
	log.RegisterWriter(observewriter, &observeWriter{core: core, level: level})
	cfg := []log.OutputConfig{
		{
			Writer:     observewriter,
			LoggerName: "test",
		},
	}
	l := log.NewZapLogWithCallerSkip(cfg, 1)

	infoMsg := "this is a info level log"
	l.Info(infoMsg)

	require.Equal(t, 1, len(ob.All()))
	require.Equal(t, 1, len(ob.All()[0].Context))
	require.Equal(t, "logger_name", ob.All()[0].Context[0].Key)
	require.Equal(t, cfg[0].LoggerName, ob.All()[0].Context[0].String)
}
