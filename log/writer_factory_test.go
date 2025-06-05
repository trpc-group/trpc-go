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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/log"
)

func ExampleRegisterCoreLevelNewer() {
	const name = "coreLevelNewer"
	log.RegisterCoreLevelNewer(name, &coreLevelNewer{})
	c := []log.OutputConfig{
		{
			Writer: name,
			Level:  "warn",
			FormatConfig: log.FormatConfig{
				MessageKey: "M",
			},
		},
	}
	l := log.NewZapLog(c)

	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")

	// Output:
	// warn
	// error
}

type coreLevelNewer struct{}

func (cw *coreLevelNewer) NewCoreLevel(config log.OutputConfig) (zapcore.Core, zap.AtomicLevel, error) {
	level := zap.NewAtomicLevelAt(log.Levels[config.Level])
	return zapcore.NewCore(
		zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        config.FormatConfig.TimeKey,
			LevelKey:       config.FormatConfig.LevelKey,
			NameKey:        config.FormatConfig.NameKey,
			CallerKey:      config.FormatConfig.CallerKey,
			FunctionKey:    config.FormatConfig.FunctionKey,
			MessageKey:     config.FormatConfig.MessageKey,
			StacktraceKey:  config.FormatConfig.StacktraceKey,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		zapcore.Lock(os.Stdout),
		&level), level, nil
}

func TestGetWriter(t *testing.T) {
	require.Nil(t, log.GetWriter(t.Name()))

	f := &log.ConsoleWriterFactory{}
	log.RegisterWriter(t.Name(), f)
	require.Equal(t, f, log.GetWriter(t.Name()))
}
