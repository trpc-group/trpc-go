// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package log_test

import (
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
	"trpc.group/trpc-go/trpc-go/plugin"
)

func TestNewZapLog(t *testing.T) {
	logger := log.NewZapLog(defaultConfig)
	assert.NotNil(t, logger)

	logger.SetLevel("0", log.LevelInfo)
	lvl := logger.GetLevel("0")
	assert.Equal(t, lvl, log.LevelInfo)

	l := logger.With(log.Field{Key: "test", Value: "a"})
	l.SetLevel("output", log.LevelDebug)
	assert.Equal(t, log.LevelDebug, l.GetLevel("output"))
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

	l := logger.With(log.Field{Key: "test", Value: "a"})
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

func TestOptionLogger2(t *testing.T) {
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

		l := log.NewZapLogWithCallerSkip(cfg, 1)
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
	core zapcore.Core
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
	decoder.ZapLevel = zap.NewAtomicLevel()
	return nil
}

func TestLogLevel(t *testing.T) {
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

	var (
		got  []string
		want []string
	)
	for i, c := range config {
		got = append(got, log.LevelStrings[l.GetLevel(fmt.Sprint(i))])
		want = append(want, log.Levels[c.Level].String())
	}
	require.Equal(t, want, got)
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
