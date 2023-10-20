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

package test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

type bufferWriter struct {
	buf buffer
}

func (w *bufferWriter) Type() string {
	return "log"
}

func (w *bufferWriter) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("console writer decoder empty")
	}
	decoder, ok := dec.(*log.Decoder)
	if !ok {
		return errors.New("console writer log decoder type invalid")
	}
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:    decoder.OutputConfig.FormatConfig.TimeKey,
		EncodeTime: zapcore.TimeEncoderOfLayout(decoder.OutputConfig.FormatConfig.TimeFmt),

		CallerKey:    decoder.OutputConfig.FormatConfig.CallerKey,
		EncodeCaller: zapcore.ShortCallerEncoder,

		LevelKey:    decoder.OutputConfig.FormatConfig.LevelKey,
		EncodeLevel: zapcore.CapitalLevelEncoder,

		MessageKey:     decoder.OutputConfig.FormatConfig.MessageKey,
		NameKey:        decoder.OutputConfig.FormatConfig.NameKey,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	level := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	decoder.Core = zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapcore.Lock(&w.buf), level)
	decoder.ZapLevel = level
	return nil
}

type buffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *buffer) Sync() error {
	return nil
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *buffer) message() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestBasicLogLevel(t *testing.T) {
	w := bufferWriter{buf: buffer{}}
	mustRegisterLogWriter(t, "buffer", &w)
	l := log.NewZapLog([]log.OutputConfig{
		{
			Writer:    "buffer",
			Level:     "debug",
			Formatter: "json",
			FormatConfig: log.FormatConfig{
				TimeKey:    "",
				MessageKey: "msg",
				LevelKey:   "level",
			},
		},
	})

	l = l.With(log.Field{Key: "tRPC-Go", Value: "log"})
	const defaultLoggerName = "default"
	oldDefaultLogger := log.GetDefaultLogger()
	log.Register(defaultLoggerName, l)
	defer func() {
		log.Register(defaultLoggerName, oldDefaultLogger)
	}()

	ctx := trpc.BackgroundContext()
	ctx = log.WithContextFields(ctx, "caller function", "TestBasicLogLevel")

	log.Debug("hello world")
	log.Debugf("%s", "hello world")
	log.DebugContext(ctx, "hello world")
	log.DebugContextf(ctx, "%s", "hello world")

	log.Info("hello world")
	log.Infof("%s", "hello world")
	log.InfoContext(ctx, "hello world")
	log.InfoContextf(ctx, "%s", "hello world")

	log.Warn("hello world")
	log.Warnf("%s", "hello world")
	log.WarnContext(ctx, "hello world")
	log.WarnContextf(ctx, "%s", "hello world")

	log.Error("hello world")
	log.Errorf("%s", "hello world")
	log.ErrorContext(ctx, "hello world")
	log.ErrorContextf(ctx, "%s", "hello world")

	require.Equal(t, `{"level":"DEBUG","msg":"hello world","tRPC-Go":"log"}
{"level":"DEBUG","msg":"hello world","tRPC-Go":"log"}
{"level":"DEBUG","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"DEBUG","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"INFO","msg":"hello world","tRPC-Go":"log"}
{"level":"INFO","msg":"hello world","tRPC-Go":"log"}
{"level":"INFO","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"INFO","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"WARN","msg":"hello world","tRPC-Go":"log"}
{"level":"WARN","msg":"hello world","tRPC-Go":"log"}
{"level":"WARN","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"WARN","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"ERROR","msg":"hello world","tRPC-Go":"log"}
{"level":"ERROR","msg":"hello world","tRPC-Go":"log"}
{"level":"ERROR","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"ERROR","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
`, w.buf.message())

}

func TestTraceLogLevel(t *testing.T) {
	w := bufferWriter{buf: buffer{}}
	mustRegisterLogWriter(t, "buffer", &w)
	l := log.NewZapLog([]log.OutputConfig{
		{
			Writer:    "buffer",
			Level:     "debug",
			Formatter: "json",
			FormatConfig: log.FormatConfig{
				TimeKey:    "",
				MessageKey: "msg",
				LevelKey:   "level",
			},
		},
	})
	l = l.With(log.Field{Key: "tRPC-Go", Value: "log"})
	const defaultLoggerName = "default"
	oldDefaultLogger := log.GetDefaultLogger()
	log.Register(defaultLoggerName, l)
	defer func() {
		log.Register(defaultLoggerName, oldDefaultLogger)
	}()

	ctx := trpc.BackgroundContext()
	ctx = log.WithContextFields(ctx, "caller function", "TestBasicLogLevel")
	log.Trace("hello world")
	log.Tracef("%s", "hello world")
	log.TraceContext(ctx, "hello world")
	log.TraceContextf(ctx, "%s", "hello world")
	require.Empty(t, w.buf.message())

	log.EnableTrace()

	log.Trace("hello world")
	log.Tracef("%s", "hello world")
	log.TraceContext(ctx, "hello world")
	log.TraceContextf(ctx, "%s", "hello world")
	require.Equal(t, `{"level":"DEBUG","msg":"hello world","tRPC-Go":"log"}
{"level":"DEBUG","msg":"hello world","tRPC-Go":"log"}
{"level":"DEBUG","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
{"level":"DEBUG","msg":"hello world","tRPC-Go":"log","caller function":"TestBasicLogLevel"}
`, w.buf.message())
}

func TestLogWriter(t *testing.T) {
	w := bufferWriter{buf: buffer{}}
	mustRegisterLogWriter(t, "buffer", &w)
	mustRegisterLogWriter(t, log.OutputConsole, log.DefaultConsoleWriterFactory)
	mustRegisterLogWriter(t, log.OutputFile, log.DefaultFileWriterFactory)
	logDir := t.TempDir()
	const (
		syncFileName  = "trpc.syncLog"
		asyncFileName = "trpc.asyncLog"
	)
	l := log.NewZapLog([]log.OutputConfig{
		{
			Writer: "buffer",
			Level:  "debug",
			FormatConfig: log.FormatConfig{
				TimeKey:  "T",
				TimeFmt:  "2006-01-02 15:04:05.000",
				LevelKey: "L", MessageKey: "M", CallerKey: "C", FunctionKey: ""},
			Formatter: "json",
		},
		{
			Writer: log.OutputFile,
			Level:  "debug",
			WriteConfig: log.WriteConfig{
				LogPath:   logDir,
				Filename:  syncFileName,
				WriteMode: log.WriteSync,
			},
			Formatter: "json",
		},
		{
			Writer: log.OutputFile,
			Level:  "debug",
			WriteConfig: log.WriteConfig{
				LogPath:   logDir,
				Filename:  asyncFileName,
				WriteMode: log.WriteAsync,
			},
			Formatter: "json",
		},
		{
			Writer:    log.OutputConsole,
			Level:     "debug",
			Formatter: "console",
		},
	})

	const defaultLoggerName = "default"
	oldDefaultLogger := log.GetDefaultLogger()
	log.Register(defaultLoggerName, l)
	defer func() {
		log.Register(defaultLoggerName, oldDefaultLogger)
	}()

	log.Debug("hello world")
	log.Info("hello world")
	log.Warn("hello world")
	log.Error("hello world")
	log.Trace("hello world")
	require.Equal(t, w.buf.message(), mustReadFile(t, path.Join(logDir, syncFileName)))

	log.Sync()
	require.Equal(t, w.buf.message(), mustReadFile(t, path.Join(logDir, asyncFileName)))
}

func mustReadFile(t *testing.T, name string) string {
	t.Helper()

	file, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}

	bts, err := io.ReadAll(file)
	if err != nil {
		t.Fatal()
	}

	return string(bts)
}

func mustRegisterLogWriter(t *testing.T, name string, writer plugin.Factory) {
	t.Helper()

	log.RegisterWriter(name, writer)
	if want, got := "log", log.GetWriter(name).Type(); want != got {
		t.Fatalf("type of writer is not log, want: %s, got: %s", want, got)
	}
}
