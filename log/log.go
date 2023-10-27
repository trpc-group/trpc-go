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
	"errors"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/env"
)

var traceEnabled = traceEnableFromEnv()

// traceEnableFromEnv checks whether trace is enabled by reading from environment.
// Close trace if empty or zero, open trace if not zero, default as closed.
func traceEnableFromEnv() bool {
	switch os.Getenv(env.LogTrace) {
	case "":
		fallthrough
	case "0":
		return false
	default:
		return true
	}
}

// EnableTrace enables trace.
func EnableTrace() {
	traceEnabled = true
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
	if ol, ok := GetDefaultLogger().(OptionLogger); ok {
		return ol.WithOptions(WithAdditionalCallerSkip(-1)).With(fields...)
	}
	return GetDefaultLogger().With(fields...)
}

// WithContext add user defined fields to the Logger of context. Fields support multiple values.
func WithContext(ctx context.Context, fields ...Field) Logger {
	logger, ok := codec.Message(ctx).Logger().(Logger)
	if !ok {
		return With(fields...)
	}
	if ol, ok := logger.(OptionLogger); ok {
		return ol.WithOptions(WithAdditionalCallerSkip(-1)).With(fields...)
	}
	return logger.With(fields...)
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

	return nil, errors.New("log: only supports redirecting std logs to trpc zap logger")
}

// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Print.
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

// TraceContext logs to TRACE log. Arguments are handled in the manner of fmt.Print.
func TraceContext(ctx context.Context, args ...interface{}) {
	if !traceEnabled {
		return
	}
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Trace(args...)
		return
	}
	GetDefaultLogger().Trace(args...)
}

// TraceContextf logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func TraceContextf(ctx context.Context, format string, args ...interface{}) {
	if !traceEnabled {
		return
	}
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Tracef(format, args...)
		return
	}
	GetDefaultLogger().Tracef(format, args...)
}

// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Print.
func Debug(args ...interface{}) {
	GetDefaultLogger().Debug(args...)
}

// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
func Info(args ...interface{}) {
	GetDefaultLogger().Info(args...)
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func Warn(args ...interface{}) {
	GetDefaultLogger().Warn(args...)
}

// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func Error(args ...interface{}) {
	GetDefaultLogger().Error(args...)
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Print.
// All Fatal logs will exit by calling os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func Fatal(args ...interface{}) {
	GetDefaultLogger().Fatal(args...)
}

// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}

// WithContextFields sets some user defined data to logs, such as uid, imei, etc.
// Fields must be paired.
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

// DebugContext logs to DEBUG log. Arguments are handled in the manner of fmt.Print.
func DebugContext(ctx context.Context, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Debug(args...)
		return
	}
	GetDefaultLogger().Debug(args...)
}

// DebugContextf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func DebugContextf(ctx context.Context, format string, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Debugf(format, args...)
		return
	}
	GetDefaultLogger().Debugf(format, args...)
}

// InfoContext logs to INFO log. Arguments are handled in the manner of fmt.Print.
func InfoContext(ctx context.Context, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Info(args...)
		return
	}
	GetDefaultLogger().Info(args...)
}

// InfoContextf logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func InfoContextf(ctx context.Context, format string, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Infof(format, args...)
		return
	}
	GetDefaultLogger().Infof(format, args...)
}

// WarnContext logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func WarnContext(ctx context.Context, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Warn(args...)
		return
	}
	GetDefaultLogger().Warn(args...)
}

// WarnContextf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func WarnContextf(ctx context.Context, format string, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Warnf(format, args...)
		return
	}
	GetDefaultLogger().Warnf(format, args...)

}

// ErrorContext logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func ErrorContext(ctx context.Context, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Error(args...)
		return
	}
	GetDefaultLogger().Error(args...)
}

// ErrorContextf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func ErrorContextf(ctx context.Context, format string, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Errorf(format, args...)
		return
	}
	GetDefaultLogger().Errorf(format, args...)
}

// FatalContext logs to ERROR log. Arguments are handled in the manner of fmt.Print.
// All Fatal logs will exit by calling os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func FatalContext(ctx context.Context, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Fatal(args...)
		return
	}
	GetDefaultLogger().Fatal(args...)
}

// FatalContextf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func FatalContextf(ctx context.Context, format string, args ...interface{}) {
	if l, ok := codec.Message(ctx).Logger().(Logger); ok {
		l.Fatalf(format, args...)
		return
	}
	GetDefaultLogger().Fatalf(format, args...)
}
