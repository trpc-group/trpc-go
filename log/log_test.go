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
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
)

func TestSetLevel(t *testing.T) {
	const level = "0"
	log.SetLevel(level, log.LevelInfo)
	require.Equal(t, log.LevelInfo, log.GetLevel(level))
}

func TestSetLogger(t *testing.T) {
	logger := log.NewZapLog(log.Config{})
	log.SetLogger(logger)
	require.Equal(t, log.GetDefaultLogger(), logger)
}

func TestLogXXX(t *testing.T) {
	log.Fatal("xxx")
}

func TestLoggerNil(t *testing.T) {
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithLogger((log.Logger)(nil))
	log.EnableTrace()
	log.TraceContext(ctx, "test")
	log.TraceContextf(ctx, "test %s", "log")
	log.DebugContext(ctx, "test")
	log.DebugContextf(ctx, "test %s", "log")
	log.InfoContext(ctx, "test")
	log.InfoContextf(ctx, "test %s", "log")
	log.ErrorContext(ctx, "test")
	log.ErrorContextf(ctx, "test %s", "log")
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test %s", "log")
	log.FatalContext(ctx, "test")
	log.FatalContextf(ctx, "test %s", "log")

	l := msg.Logger()
	require.Nil(t, l)
}

func TestLoggerZapLogWrapper(t *testing.T) {
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithLogger(log.NewZapLog(defaultConfig))

	log.EnableTrace()
	log.TraceContext(ctx, "test")
	log.TraceContextf(ctx, "test")
	log.DebugContext(ctx, "test")
	log.DebugContextf(ctx, "test")
	log.InfoContext(ctx, "test")
	log.InfoContextf(ctx, "test %s", "s")
	log.ErrorContext(ctx, "test")
	log.ErrorContextf(ctx, "test")
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test")

	msg.WithLogger(log.NewZapLog(defaultConfig))
	log.WithContextFields(ctx, "a", "a")
	log.SetLevel("console", log.LevelDebug)
	require.Equal(t, log.GetLevel("console"), log.LevelDebug)
}

func TestWithContextFields(t *testing.T) {
	ctx := trpc.BackgroundContext()
	log.WithContextFields(ctx, "k", "v")
	require.NotNil(t, codec.Message(ctx).Logger())

	ctx = context.Background()
	newCtx := log.WithContextFields(ctx, "k", "v")
	require.Nil(t, codec.Message(ctx).Logger())
	require.NotNil(t, codec.Message(newCtx).Logger())
}

func TestOptionLogger1(t *testing.T) {
	log.Debug("test1")
	log.Debug("test2")
	log.Debug("test3")
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCallerServiceName("trpc.test.helloworld.Greeter")
	log.WithContextFields(ctx, "a", "a")
	log.TraceContext(ctx, "test")
	log.InfoContext(ctx, "test")
	log.WithContextFields(ctx, "b", "b")
	log.InfoContext(ctx, "test")

	trpc.Message(ctx).WithLogger(log.Get("default"))
	log.DebugContext(ctx, "custom log msg")
}

func TestCustomLogger(t *testing.T) {
	log.Register("custom", log.NewZapLogWithCallerSkip(log.Config{log.OutputConfig{Writer: "console"}}, 1))
	log.Get("custom").Debug("test")
}

const (
	noOptionBufLogger = "noOptionBuf"
	customBufLogger   = "customBuf"
)

func getCtxFuncs() []func() context.Context {
	return []func() context.Context{
		func() context.Context {
			return context.Background()
		},
		func() context.Context {
			ctx, msg := codec.WithNewMessage(context.Background())
			msg.WithCallerServiceName("trpc.test.helloworld.Greeter")
			return ctx
		}, func() context.Context {
			ctx, msg := codec.WithNewMessage(context.Background())
			msg.WithLogger(log.GetDefaultLogger())
			msg.WithCallerServiceName("trpc.test.helloworld.Greeter")
			return ctx
		}, func() context.Context {
			ctx, msg := codec.WithNewMessage(context.Background())
			msg.WithLogger(log.Get(noOptionBufLogger))
			msg.WithCallerServiceName("trpc.test.helloworld.Greeter")
			return ctx
		},
	}
}

func TestWithContext(t *testing.T) {
	old := log.GetDefaultLogger()
	defer log.SetLogger(old)
	for _, ctxFunc := range getCtxFuncs() {
		ctx := ctxFunc()
		checkTrace(t, func() {
			log.WithContext(ctx, log.Field{Key: "123", Value: 123}).Debugf("test")
			log.WithContext(ctx, log.Field{Key: "123", Value: 123}).With(log.Field{Key: "k2", Value: "v2"}).Debugf("test")
			log.WithContext(ctx, log.Field{Key: "123", Value: 123}).With(log.Field{Key: "k2", Value: "v2"}).With(log.Field{Key: "k2", Value: "v2"}).Debugf("test")
		}, nil)
	}
}

func TestStacktrace(t *testing.T) {
	checkTrace(t, func() {
		log.Debug("test")
		log.Error("test")
	}, nil)
}

func check(t *testing.T, out *bytes.Buffer, fn func()) {
	fn()

	_, file, start, ok := runtime.Caller(2)
	assert.True(t, ok)

	pathPre := filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)) + ":"
	trace := out.String()
	count := strings.Count(trace, pathPre)
	fmt.Println(" line count:", count, "start:", start+1, "end:", start+count)
	fmt.Println(trace)
	for line := start + 1; line <= start+count; line++ {
		path := pathPre + strconv.Itoa(line)
		require.Contains(t, out.String(), path, "log trace error")
	}

	verifyNoZap(t, trace)
}

var (
	buf = &bytes.Buffer{}
)

func init() {
	log.Register(customBufLogger, log.NewZapBufLogger(buf, 1))
	log.Register(noOptionBufLogger, newNoOptionBufLogger(buf, 1))
}

// checkTrace set buf log to check trace.
func checkTrace(t *testing.T, fn func(), setLog func()) {
	*buf = bytes.Buffer{}
	log.SetLogger(log.NewZapBufLogger(buf, 2))
	if setLog != nil {
		setLog()
	}
	check(t, buf, fn)
}

func verifyNoZap(t *testing.T, logs string) {
	for _, fnPrefix := range zapPackages {
		require.NotContains(t, logs, fnPrefix, "should not contain zap package")
	}
}

// zapPackages are packages that we search for in the logging output to match a
// zap stack frame.
var zapPackages = []string{
	"go.uber.org/zap",
	"go.uber.org/zap/zapcore",
}

func TestLogFatal(t *testing.T) {
	old := log.GetDefaultLogger()
	defer log.SetLogger(old)

	var h customWriteHook
	log.SetLogger(log.NewZapFatalLogger(&h))
	log.Fatal("test")
	assert.True(t, h.called)
	h.called = false
	log.Fatalf("test")
	assert.True(t, h.called)
	h.called = false
	ctx := context.Background()
	log.FatalContext(ctx, "test")
	assert.True(t, h.called)
	h.called = false
	log.FatalContextf(ctx, "test")
	assert.True(t, h.called)

	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithLogger(log.GetDefaultLogger())
	msg.WithCallerServiceName("trpc.test.helloworld.Greeter")
	h.called = false
	log.FatalContext(ctx, "test")
	assert.True(t, h.called)
	h.called = false
	log.FatalContextf(ctx, "test")
	assert.True(t, h.called)
}

type customWriteHook struct {
	called bool
}

func (h *customWriteHook) OnWrite(_ *zapcore.CheckedEntry, _ []zap.Field) {
	h.called = true
}
