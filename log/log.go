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

// Package log provides a log for the framework and applications.
package log

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/env"
)

var traceEnabled = traceEnableFromEnv()

// traceEnableFromEnv checks whether trace is enabled by reading from environment.
// Enable trace if env is empty or zero,  disable trace if env is not zero, default as disabled.
func traceEnableFromEnv() bool {
	if e := os.Getenv(env.LogTrace); e == "" || e == "0" {
		return false
	}
	return true
}

// EnableTrace enables trace.
func EnableTrace() {
	traceEnabled = true
}

// setTraceEnabled sets whether to enable trace.
func setTraceEnabled(enable bool) {
	traceEnabled = enable
}

// SetLevel sets log level for different output which may be "0", "1" or "2".
func SetLevel(output string, level Level) {
	GetDefaultLogger().SetLevel(output, level)
}

// GetLevel gets log level for different output.
func GetLevel(output string) Level {
	return GetDefaultLogger().GetLevel(output)
}

// With adds user defined fields to Logger. Field support multiple values.
func With(fields ...Field) Logger {
	return GetDefaultLogger().With(fields...)
}

// WithFields sets some user defined data to logs, such as, uid, imei. Fields must be paired.
// Deprecated: use With instead.
func WithFields(fields ...string) Logger {
	return GetDefaultLogger().WithFields(fields...)
}

// WithContext adds user defined fields to the Logger of context.
// Fields support multiple values.
func WithContext(ctx context.Context, fields ...Field) Logger {
	logger, ok := codec.Message(ctx).Logger().(Logger)
	if !ok {
		return With(fields...)
	}
	return logger.With(fields...)
}

// WithFieldsContext adds user defined data to the Logger of context.
// Data may be uid, imei, etc. Fields must be paired.
// Deprecated: use WithContext instead.
func WithFieldsContext(ctx context.Context, fields ...string) Logger {
	logger, ok := codec.Message(ctx).Logger().(Logger)
	if !ok {
		return WithFields(fields...)
	}
	return logger.WithFields(fields...)
}

// RedirectStdLog redirects std log to trpc logger as log level INFO.
// After redirection, log flag is zero, the prefix is empty.
// The returned function may be used to recover log flag and prefix, and redirect output to
// os.Stderr.
func RedirectStdLog(logger Logger) (func(), error) {
	return RedirectStdLogAt(logger, zap.InfoLevel)
}

// RedirectStdLogAt redirects std log to trpc logger with a specific level.
// After redirection, log flag is zero, the prefix is empty.
// The returned function may be used to recover log flag and prefix, and redirect output to
// os.Stderr.
func RedirectStdLogAt(logger Logger, level zapcore.Level) (func(), error) {
	if l, ok := logger.(*zapLog); ok {
		return zap.RedirectStdLogAt(l.logger, level)
	}
	if l, ok := logger.(*ZapLogWrapper); ok {
		return zap.RedirectStdLogAt(l.l.logger, level)
	}
	return nil, fmt.Errorf("log: only supports redirecting std logs to trpc zap logger")
}

// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Println.
func Trace(args ...interface{}) {
	if traceEnabled {
		GetDefaultLogger().Trace(args...)
	}
}

// Tracef logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func Tracef(format string, args ...interface{}) {
	if traceEnabled {
		GetDefaultLogger().Tracef(format, args...)
	}
}

// TraceContext logs to TRACE log. Arguments are handled in the manner of fmt.Println.
func TraceContext(ctx context.Context, args ...interface{}) {
	if !traceEnabled {
		return
	}
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l and l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Trace(args...)
			return
		}
		l.l.Trace(args...)
	case Logger:
		l.Trace(args...)
	default:
		GetDefaultLogger().Trace(args...)
	}
}

// TraceContextf logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func TraceContextf(ctx context.Context, format string, args ...interface{}) {
	if !traceEnabled {
		return
	}
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l and l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Tracef(format, args...)
			return
		}
		l.l.Tracef(format, args...)
	case Logger:
		l.Tracef(format, args...)
	default:
		GetDefaultLogger().Tracef(format, args...)
	}
}

// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Println.
func Debug(args ...interface{}) {
	GetDefaultLogger().Debug(args...)
}

// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Println.
func Info(args ...interface{}) {
	GetDefaultLogger().Info(args...)
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Println.
func Warn(args ...interface{}) {
	GetDefaultLogger().Warn(args...)
}

// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Println.
func Error(args ...interface{}) {
	GetDefaultLogger().Error(args...)
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Println.
// All Fatal logs will exit by calling os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func Fatal(args ...interface{}) {
	GetDefaultLogger().Fatal(args...)
}

// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}

// WithContextFields adds the provided fields into the logger within the context,
// rather than directly into the context itself. Fields must be paired.
// This function is useful for adding user-defined data to logger, such as uid, imei, etc.
// If ctx has already set a Msg, this function returns that ctx, otherwise, it returns a new one.
func WithContextFields(ctx context.Context, fields ...string) context.Context {
	tagCapacity := len(fields) / 2
	tags := make([]Field, 0, tagCapacity)
	for i := 0; i < tagCapacity; i++ {
		tags = append(tags, Field{
			Key:   fields[2*i],
			Value: fields[2*i+1],
		})
	}

	ctx, msg := codec.EnsureMessage(ctx)
	logger, ok := msg.Logger().(Logger)
	if ok && logger != nil {
		logger = logger.With(tags...)
	} else {
		logger = GetDefaultLogger().With(tags...)
	}

	msg.WithLogger(logger)
	return ctx
}

// DebugContext logs to DEBUG log. Arguments are handled in the manner of fmt.Println.
func DebugContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Debug(args...)
			return
		}
		l.l.Debug(args...)
	case Logger:
		l.Debug(args...)
	default:
		GetDefaultLogger().Debug(args...)
	}
}

// DebugContextf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func DebugContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Debugf(format, args...)
			return
		}
		l.l.Debugf(format, args...)
	case Logger:
		l.Debugf(format, args...)
	default:
		GetDefaultLogger().Debugf(format, args...)
	}
}

// InfoContext logs to INFO log. Arguments are handled in the manner of fmt.Println.
func InfoContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Info(args...)
			return
		}
		l.l.Info(args...)
	case Logger:
		l.Info(args...)
	default:
		GetDefaultLogger().Info(args...)
	}
}

// InfoContextf logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func InfoContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Infof(format, args...)
			return
		}
		l.l.Infof(format, args...)
	case Logger:
		l.Infof(format, args...)
	default:
		GetDefaultLogger().Infof(format, args...)
	}
}

// WarnContext logs to WARNING log. Arguments are handled in the manner of fmt.Println.
func WarnContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Warn(args...)
			return
		}
		l.l.Warn(args...)
	case Logger:
		l.Warn(args...)
	default:
		GetDefaultLogger().Warn(args...)
	}
}

// WarnContextf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func WarnContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Warnf(format, args...)
			return
		}
		l.l.Warnf(format, args...)
	case Logger:
		l.Warnf(format, args...)
	default:
		GetDefaultLogger().Warnf(format, args...)
	}
}

// ErrorContext logs to ERROR log. Arguments are handled in the manner of fmt.Println.
func ErrorContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Error(args...)
			return
		}
		l.l.Error(args...)
	case Logger:
		l.Error(args...)
	default:
		GetDefaultLogger().Error(args...)
	}
}

// ErrorContextf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func ErrorContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Errorf(format, args...)
			return
		}
		l.l.Errorf(format, args...)
	case Logger:
		l.Errorf(format, args...)
	default:
		GetDefaultLogger().Errorf(format, args...)
	}
}

// FatalContext logs to ERROR log. Arguments are handled in the manner of fmt.Println.
// All Fatal logs will exit by calling os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func FatalContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Fatal(args...)
			return
		}
		l.l.Fatal(args...)
	case Logger:
		l.Fatal(args...)
	default:
		GetDefaultLogger().Fatal(args...)
	}
}

// FatalContextf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func FatalContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// ensure l or l.l is not nil.
		if l == nil || l.l == nil {
			GetDefaultLogger().Fatalf(format, args...)
			return
		}
		l.l.Fatalf(format, args...)
	case Logger:
		l.Fatalf(format, args...)
	default:
		GetDefaultLogger().Fatalf(format, args...)
	}
}
