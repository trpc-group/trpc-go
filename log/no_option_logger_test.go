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
	"fmt"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

// newNoOptionBufLogger creates a no option buf Logger from zap.
func newNoOptionBufLogger(buf *bytes.Buffer, skip int) log.Logger {
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)
	return &noOptionLog{
		levels: []zap.AtomicLevel{},
		logger: zap.New(
			core,
			zap.AddCallerSkip(skip),
			zap.AddCaller(),
		),
	}
}

// noOptionLog is a log.Logger implementation based on zaplogger, but without option.
type noOptionLog struct {
	levels []zap.AtomicLevel
	logger *zap.Logger
}

// With add user defined fields to log.Logger. Fields support multiple values.
func (l *noOptionLog) With(fields ...log.Field) log.Logger {
	zapFields := make([]zap.Field, len(fields))
	for i := range fields {
		zapFields[i] = zap.Any(fields[i].Key, fields[i].Value)
	}

	return &noOptionLog{
		levels: l.levels,
		logger: l.logger.With(zapFields...)}
}

func getLogMsg(args ...interface{}) string {
	msg := fmt.Sprint(args...)
	report.LogWriteSize.IncrBy(float64(len(msg)))
	return msg
}

func getLogMsgf(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	report.LogWriteSize.IncrBy(float64(len(msg)))
	return msg
}

// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Print.
func (l *noOptionLog) Trace(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		l.logger.Debug(getLogMsg(args...))
	}
}

// Tracef logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func (l *noOptionLog) Tracef(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		l.logger.Debug(getLogMsgf(format, args...))
	}
}

// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Print.
func (l *noOptionLog) Debug(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		l.logger.Debug(getLogMsg(args...))
	}
}

// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func (l *noOptionLog) Debugf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		l.logger.Debug(getLogMsgf(format, args...))
	}
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
func (l *noOptionLog) Info(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.InfoLevel) {
		l.logger.Info(getLogMsg(args...))
	}
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func (l *noOptionLog) Infof(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.InfoLevel) {
		l.logger.Info(getLogMsgf(format, args...))
	}
}

// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func (l *noOptionLog) Warn(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.WarnLevel) {
		l.logger.Warn(getLogMsg(args...))
	}
}

// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func (l *noOptionLog) Warnf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.WarnLevel) {
		l.logger.Warn(getLogMsgf(format, args...))
	}
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func (l *noOptionLog) Error(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.ErrorLevel) {
		l.logger.Error(getLogMsg(args...))
	}
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func (l *noOptionLog) Errorf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.ErrorLevel) {
		l.logger.Error(getLogMsgf(format, args...))
	}
}

// Fatal logs to FATAL log. Arguments are handled in the manner of fmt.Print.
func (l *noOptionLog) Fatal(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.FatalLevel) {
		l.logger.Fatal(getLogMsg(args...))
	}
}

// Fatalf logs to FATAL log. Arguments are handled in the manner of fmt.Printf.
func (l *noOptionLog) Fatalf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.FatalLevel) {
		l.logger.Fatal(getLogMsgf(format, args...))
	}
}

// Sync calls the zap logger's Sync method, and flushes any buffered log entries.
// Applications should take care to call Sync before exiting.
func (l *noOptionLog) Sync() error {
	return l.logger.Sync()
}

// SetLevel sets output log level.
func (l *noOptionLog) SetLevel(output string, level log.Level) {
	i, e := strconv.Atoi(output)
	if e != nil {
		return
	}
	if i < 0 || i >= len(l.levels) {
		return
	}
	l.levels[i].SetLevel(levelToZapLevel[level])
}

// GetLevel gets output log level.
func (l *noOptionLog) GetLevel(output string) log.Level {
	i, e := strconv.Atoi(output)
	if e != nil {
		return log.LevelDebug
	}
	if i < 0 || i >= len(l.levels) {
		return log.LevelDebug
	}
	return zapLevelToLevel[l.levels[i].Level()]
}

var levelToZapLevel = map[log.Level]zapcore.Level{
	log.LevelTrace: zapcore.DebugLevel,
	log.LevelDebug: zapcore.DebugLevel,
	log.LevelInfo:  zapcore.InfoLevel,
	log.LevelWarn:  zapcore.WarnLevel,
	log.LevelError: zapcore.ErrorLevel,
	log.LevelFatal: zapcore.FatalLevel,
}

var zapLevelToLevel = map[zapcore.Level]log.Level{
	zapcore.DebugLevel: log.LevelDebug,
	zapcore.InfoLevel:  log.LevelInfo,
	zapcore.WarnLevel:  log.LevelWarn,
	zapcore.ErrorLevel: log.LevelError,
	zapcore.FatalLevel: log.LevelFatal,
}
