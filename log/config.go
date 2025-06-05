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
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go/log/internal/timeunit"
)

// output name, default support console and file.
const (
	OutputConsole  = ConsoleZapCore
	OutputFile     = FileZapCore
	ConsoleZapCore = "console"
	FileZapCore    = "file"
)

// Config is the log config. Each log may have multiple outputs.
type Config []OutputConfig

// OutputConfig is the output config, includes console, file and remote.
type OutputConfig struct {
	// Writer is the output of log, such as console or file.
	Writer      string      `yaml:"writer,omitempty"`
	WriteConfig WriteConfig `yaml:"writer_config,omitempty"`

	// Formatter is the format of log, such as console or json.
	Formatter    string       `yaml:"formatter,omitempty"`
	FormatConfig FormatConfig `yaml:"formatter_config,omitempty"`

	// RemoteConfig is the remote config. It's defined by business and should be registered by
	// third-party modules.
	RemoteConfig yaml.Node `yaml:"remote_config,omitempty"`

	// Level controls the log level, like debug, info or error.
	Level string `yaml:"level,omitempty"`

	// CallerSkip controls the nesting depth of log function.
	CallerSkip int `yaml:"caller_skip,omitempty"`

	// EnableColor determines if the output is colored. The default value is false.
	EnableColor bool `yaml:"enable_color,omitempty"`

	// LoggerName add new field for enabling zap logger name.
	LoggerName string `yaml:"logger_name,omitempty"`
}

// WriteConfig is the local file config.
type WriteConfig struct {
	// LogPath is the log path like /usr/local/trpc/log/.
	LogPath string `yaml:"log_path,omitempty"`
	// Filename is the file name like trpc.log.
	Filename string `yaml:"filename,omitempty"`
	// WriteMode is the log write mod. 1: sync, 2: async, 3: fast(maybe dropped), default as 3.
	WriteMode int `yaml:"write_mode,omitempty"`
	// RollType is the log rolling type. Split files by size/time, default by size.
	RollType string `yaml:"roll_type,omitempty"`
	// MaxAge is the max expire times(day).
	MaxAge int `yaml:"max_age,omitempty"`
	// MaxBackups is the max backup files.
	MaxBackups int `yaml:"max_backups,omitempty"`
	// Compress defines whether log should be compressed.
	Compress bool `yaml:"compress,omitempty"`
	// MaxSize is the max size of log file(MB).
	MaxSize int `yaml:"max_size,omitempty"`

	// TimeUnit splits files by time unit, like year/month/hour/minute, default day.
	// It takes effect only when split by time.
	// You can use the syntax supported by https://github.com/lestrrat-go/strftime to represent the time format,
	// and TimeUnit is the smallest time unit in the time format.
	TimeUnit timeunit.TimeUnit `yaml:"time_unit,omitempty"`
}

// FormatConfig is the log format config.
type FormatConfig struct {
	// TimeFmt is the time format of log output, default as "2006-01-02 15:04:05.000" on empty.
	TimeFmt string `yaml:"time_fmt,omitempty"`

	// TimeKey is the time key of log output, default as "T".
	// Example: 2023-07-03 20:42:24.624.
	// Use "none" to disable this field.
	TimeKey string `yaml:"time_key,omitempty"`
	// LevelKey is the level key of log output, default as "L".
	// Example: DEBUG.
	// Use "none" to disable this field.
	LevelKey string `yaml:"level_key,omitempty"`
	// NameKey is the name key of log output, default as "N".
	// Example: logger name.
	// Use "none" to disable this field.
	NameKey string `yaml:"name_key,omitempty"`
	// CallerKey is the caller key of log output, default as "C".
	// Example: testing/testing.go:1576.
	// Use "none" to disable this field.
	CallerKey string `yaml:"caller_key,omitempty"`
	// FunctionKey is the function key of log output, default as "", which means not to print
	// function name.
	// Example: testing.tRunner.
	// Use "F" to show the function name field.
	FunctionKey string `yaml:"function_key,omitempty"`
	// MessageKey is the message key of log output, default as "M".
	// Example: helloworld.
	// Use "none" to disable this field.
	MessageKey string `yaml:"message_key,omitempty"`
	// StackTraceKey is the stack trace key of log output, default as "S".
	// Use "none" to disable this field.
	StacktraceKey string `yaml:"stacktrace_key,omitempty"`
}

// WriteMode is the log write mode, one of 1, 2, 3.
type WriteMode int

const (
	// WriteSync writes synchronously.
	WriteSync = 1
	// WriteAsync writes asynchronously.
	WriteAsync = 2
	// WriteFast writes fast(may drop logs asynchronously).
	WriteFast = 3
)

// By which log rolls.
const (
	// RollBySize rolls logs by file size.
	RollBySize = "size"
	// RollByTime rolls logs by time.
	RollByTime = "time"
)

// Some common used timeunit formats.
const (
	// TimeFormatMinute is accurate to the minute.
	TimeFormatMinute = timeunit.TimeFormatMinute
	// TimeFormatHour is accurate to the hour.
	TimeFormatHour = timeunit.TimeFormatHour
	// TimeFormatDay is accurate to the day.
	TimeFormatDay = timeunit.TimeFormatDay
	// TimeFormatMonth is accurate to the month.
	TimeFormatMonth = timeunit.TimeFormatMonth
	// TimeFormatYear is accurate to the year.
	TimeFormatYear = timeunit.TimeFormatYear
)

// TimeUnit is the timeunit unit by which files are split, one of minute/hour/day/month/year.
type TimeUnit = timeunit.TimeUnit

const (
	// Minute splits by the minute.
	Minute = timeunit.Minute
	// Hour splits by the hour.
	Hour = timeunit.Hour
	// Day splits by the day.
	Day = timeunit.Day
	// Month splits by the month.
	Month = timeunit.Month
	// Year splits by the year.
	Year = timeunit.Year
)
