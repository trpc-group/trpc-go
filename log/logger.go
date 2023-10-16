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
	"io"
)

// Level is the log level.
type Level int

// Enums log level constants.
const (
	LevelNil Level = iota
	LevelTrace
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String turns the LogLevel to string.
func (lv *Level) String() string {
	return LevelStrings[*lv]
}

// LevelStrings is the map from log level to its string representation.
var LevelStrings = map[Level]string{
	LevelTrace: "trace",
	LevelDebug: "debug",
	LevelInfo:  "info",
	LevelWarn:  "warn",
	LevelError: "error",
	LevelFatal: "fatal",
}

// LevelNames is the map from string to log level.
var LevelNames = map[string]Level{
	"trace": LevelTrace,
	"debug": LevelDebug,
	"info":  LevelInfo,
	"warn":  LevelWarn,
	"error": LevelError,
	"fatal": LevelFatal,
}

// LoggerOptions is the log options.
type LoggerOptions struct {
	LogLevel Level
	Pattern  string
	Writer   io.Writer
}

// LoggerOption modifies the LoggerOptions.
type LoggerOption func(*LoggerOptions)

// Field is the user defined log field.
type Field struct {
	Key   string
	Value interface{}
}

// Logger is the underlying logging work for tRPC framework.
type Logger interface {
	// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Print.
	Trace(args ...interface{})
	// Tracef logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
	Tracef(format string, args ...interface{})
	// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Print.
	Debug(args ...interface{})
	// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
	Debugf(format string, args ...interface{})
	// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
	Info(args ...interface{})
	// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
	Infof(format string, args ...interface{})
	// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Print.
	Warn(args ...interface{})
	// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
	Warnf(format string, args ...interface{})
	// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
	Error(args ...interface{})
	// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
	Errorf(format string, args ...interface{})
	// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Print.
	// All Fatal logs will exit by calling os.Exit(1).
	// Implementations may also call os.Exit() with a non-zero exit code.
	Fatal(args ...interface{})
	// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
	Fatalf(format string, args ...interface{})

	// Sync calls the underlying Core's Sync method, flushing any buffered log entries.
	// Applications should take care to call Sync before exiting.
	Sync() error

	// SetLevel sets the output log level.
	SetLevel(output string, level Level)
	// GetLevel gets the output log level.
	GetLevel(output string) Level

	// With adds user defined fields to Logger. Fields support multiple values.
	With(fields ...Field) Logger
}

// OptionLogger defines logger with additional options.
type OptionLogger interface {
	WithOptions(opts ...Option) Logger
}
