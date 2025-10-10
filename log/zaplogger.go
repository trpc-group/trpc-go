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
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log/rollwriter"
)

var defaultConfig = []OutputConfig{
	{
		Writer:    "console",
		Level:     "debug",
		Formatter: "console",
	},
}

// Levels is the map from string to zapcore.Level.
var Levels = map[string]zapcore.Level{
	"":      zapcore.DebugLevel,
	"trace": zapcore.DebugLevel,
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
	"fatal": zapcore.FatalLevel,
}

var levelToZapLevel = map[Level]zapcore.Level{
	LevelTrace: zapcore.DebugLevel,
	LevelDebug: zapcore.DebugLevel,
	LevelInfo:  zapcore.InfoLevel,
	LevelWarn:  zapcore.WarnLevel,
	LevelError: zapcore.ErrorLevel,
	LevelFatal: zapcore.FatalLevel,
}

var zapLevelToLevel = map[zapcore.Level]Level{
	zapcore.DebugLevel: LevelDebug,
	zapcore.InfoLevel:  LevelInfo,
	zapcore.WarnLevel:  LevelWarn,
	zapcore.ErrorLevel: LevelError,
	zapcore.FatalLevel: LevelFatal,
}

// NewZapLog creates a trpc default Logger from zap whose caller skip is set to 2.
func NewZapLog(c Config) Logger {
	return NewZapLogWithCallerSkip(c, 2)
}

// NewZapLogWithCallerSkip creates a trpc default Logger from zap.
func NewZapLogWithCallerSkip(cfg Config, callerSkip int) Logger {
	cores := make([]zapcore.Core, 0, len(cfg))
	levels := make([]zap.AtomicLevel, 0, len(cfg))
	for _, c := range cfg {
		var (
			core  zapcore.Core
			level zap.AtomicLevel
			err   error
		)
		// The CoreLevelNewer interface always takes a higher precedence.
		if coreLevelNewer, ok := GetCoreLevelNewer(c.Writer); ok {
			core, level, err = coreLevelNewer.NewCoreLevel(c)
		} else if coreNewer, ok := GetCoreNewer(c.Writer); ok {
			core, err = coreNewer.New(c)
			level = zap.NewAtomicLevelAt(zapcore.LevelOf(core))
		} else {
			panic(fmt.Sprintf("log: getting CoreNewer failed: %s has not been registered yet", c.Writer))
		}
		if err != nil {
			panic(fmt.Sprintf("log: newing core from %s config failed: %v", c.Writer, err))
		}
		if c.LoggerName != "" {
			core = core.With([]zapcore.Field{zap.String("logger_name", c.LoggerName)})
		}
		cores = append(cores, core)
		levels = append(levels, level)
	}
	return &zapLog{
		levels: levels,
		logger: zap.New(
			zapcore.NewTee(cores...),
			zap.AddCallerSkip(callerSkip),
			zap.AddCaller(),
		),
	}
}

func newEncoder(c *OutputConfig) zapcore.Encoder {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        GetLogEncoderKey("T", c.FormatConfig.TimeKey),
		LevelKey:       GetLogEncoderKey("L", c.FormatConfig.LevelKey),
		NameKey:        GetLogEncoderKey("N", c.FormatConfig.NameKey),
		CallerKey:      GetLogEncoderKey("C", c.FormatConfig.CallerKey),
		FunctionKey:    GetLogEncoderKey(zapcore.OmitKey, c.FormatConfig.FunctionKey),
		MessageKey:     GetLogEncoderKey("M", c.FormatConfig.MessageKey),
		StacktraceKey:  GetLogEncoderKey("S", c.FormatConfig.StacktraceKey),
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     NewTimeEncoder(c.FormatConfig.TimeFmt),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if c.EnableColor {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	switch c.Formatter {
	case "console":
		return zapcore.NewConsoleEncoder(encoderCfg)
	case "json":
		return zapcore.NewJSONEncoder(encoderCfg)
	default:
		return zapcore.NewConsoleEncoder(encoderCfg)
	}
}

// GetLogEncoderKey gets user defined log output name, uses defKey if empty.
// If key is "none", return empty string to disable the corresponding field.
func GetLogEncoderKey(defKey, key string) string {
	const none = "none"
	if key == none {
		return ""
	}
	if key == "" {
		return defKey
	}
	return key
}

func newConsoleCore(c *OutputConfig) (zapcore.Core, zap.AtomicLevel) {
	lvl := zap.NewAtomicLevelAt(Levels[c.Level])
	return zapcore.NewCore(
		newEncoder(c),
		zapcore.Lock(os.Stdout),
		lvl), lvl
}

func newFileCore(c *OutputConfig) (zapcore.Core, zap.AtomicLevel, error) {
	opts := []rollwriter.Option{
		rollwriter.WithMaxAge(c.WriteConfig.MaxAge),
		rollwriter.WithMaxBackups(c.WriteConfig.MaxBackups),
		rollwriter.WithCompress(c.WriteConfig.Compress),
		rollwriter.WithMaxSize(c.WriteConfig.MaxSize),
	}
	// roll by time.
	if c.WriteConfig.RollType != RollBySize {
		opts = append(opts, rollwriter.WithRotationTime(c.WriteConfig.TimeUnit.Format()))
	}

	writer, err := rollwriter.NewRollWriter(c.WriteConfig.Filename, opts...)
	if err != nil {
		return nil, zap.AtomicLevel{}, err
	}

	// write mode.
	var ws zapcore.WriteSyncer
	switch m := c.WriteConfig.WriteMode; m {
	case 0, WriteFast:
		// Use WriteFast as default mode.
		// It has better performance, discards logs on full and avoid blocking service.
		ws = rollwriter.NewAsyncRollWriter(writer, rollwriter.WithDropLog(true))
	case WriteSync:
		ws = zapcore.AddSync(writer)
	case WriteAsync:
		ws = rollwriter.NewAsyncRollWriter(writer, rollwriter.WithDropLog(false))
	default:
		return nil, zap.AtomicLevel{}, fmt.Errorf("validating WriteMode parameter: got %d, "+
			"but expect one of WriteFast(%d), WriteAsync(%d), or WriteSync(%d)", m, WriteFast, WriteAsync, WriteSync)
	}

	// log level.
	lvl := zap.NewAtomicLevelAt(Levels[c.Level])
	return zapcore.NewCore(
		newEncoder(c),
		ws, lvl,
	), lvl, nil
}

// NewTimeEncoder creates a time format encoder.
func NewTimeEncoder(format string) zapcore.TimeEncoder {
	switch format {
	case "":
		return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendByteString(defaultTimeFormat(t))
		}
	case "seconds":
		return zapcore.EpochTimeEncoder
	case "milliseconds":
		return zapcore.EpochMillisTimeEncoder
	case "nanoseconds":
		return zapcore.EpochNanosTimeEncoder
	default:
		return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(format))
		}
	}
}

// defaultTimeFormat returns the default time format "2006-01-02 15:04:05.000",
// which performs better than https://pkg.go.dev/time#Time.AppendFormat.
func defaultTimeFormat(t time.Time) []byte {
	t = t.Local()
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	micros := t.Nanosecond() / 1000

	buf := make([]byte, 23)
	buf[0] = byte((year/1000)%10) + '0'
	buf[1] = byte((year/100)%10) + '0'
	buf[2] = byte((year/10)%10) + '0'
	buf[3] = byte(year%10) + '0'
	buf[4] = '-'
	buf[5] = byte((month)/10) + '0'
	buf[6] = byte((month)%10) + '0'
	buf[7] = '-'
	buf[8] = byte((day)/10) + '0'
	buf[9] = byte((day)%10) + '0'
	buf[10] = ' '
	buf[11] = byte((hour)/10) + '0'
	buf[12] = byte((hour)%10) + '0'
	buf[13] = ':'
	buf[14] = byte((minute)/10) + '0'
	buf[15] = byte((minute)%10) + '0'
	buf[16] = ':'
	buf[17] = byte((second)/10) + '0'
	buf[18] = byte((second)%10) + '0'
	buf[19] = '.'
	buf[20] = byte((micros/100000)%10) + '0'
	buf[21] = byte((micros/10000)%10) + '0'
	buf[22] = byte((micros/1000)%10) + '0'
	return buf
}

// ZapLogWrapper delegates zapLogger which was introduced in this
// By ZapLogWrapper proxy, we can add a layer to the debug series function calls, so that the caller
// information can be set correctly.
type ZapLogWrapper struct {
	l *zapLog
}

// GetLogger returns interval zapLog.
func (z *ZapLogWrapper) GetLogger() Logger {
	return z.l
}

// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Println.
func (z *ZapLogWrapper) Trace(args ...interface{}) {
	z.l.Trace(args...)
}

// Tracef logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func (z *ZapLogWrapper) Tracef(format string, args ...interface{}) {
	z.l.Tracef(format, args...)
}

// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Println.
func (z *ZapLogWrapper) Debug(args ...interface{}) {
	z.l.Debug(args...)
}

// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func (z *ZapLogWrapper) Debugf(format string, args ...interface{}) {
	z.l.Debugf(format, args...)
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Println.
func (z *ZapLogWrapper) Info(args ...interface{}) {
	z.l.Info(args...)
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func (z *ZapLogWrapper) Infof(format string, args ...interface{}) {
	z.l.Infof(format, args...)
}

// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Println.
func (z *ZapLogWrapper) Warn(args ...interface{}) {
	z.l.Warn(args...)
}

// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func (z *ZapLogWrapper) Warnf(format string, args ...interface{}) {
	z.l.Warnf(format, args...)
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Println.
func (z *ZapLogWrapper) Error(args ...interface{}) {
	z.l.Error(args...)
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func (z *ZapLogWrapper) Errorf(format string, args ...interface{}) {
	z.l.Errorf(format, args...)
}

// Fatal logs to FATAL log. Arguments are handled in the manner of fmt.Println.
func (z *ZapLogWrapper) Fatal(args ...interface{}) {
	z.l.Fatal(args...)
}

// Fatalf logs to FATAL log. Arguments are handled in the manner of fmt.Printf.
func (z *ZapLogWrapper) Fatalf(format string, args ...interface{}) {
	z.l.Fatalf(format, args...)
}

// Sync calls the zap logger's Sync method, and flushes any buffered log entries.
// Applications should take care to call Sync before exiting.
func (z *ZapLogWrapper) Sync() error {
	return z.l.Sync()
}

// SetLevel set output log level.
func (z *ZapLogWrapper) SetLevel(output string, level Level) {
	z.l.SetLevel(output, level)
}

// GetLevel gets output log level.
func (z *ZapLogWrapper) GetLevel(output string) Level {
	return z.l.GetLevel(output)
}

// WithFields set some user defined data to logs, such as uid, imei, etc.
// Use this function at the beginning of each request. The returned new Logger should be used to
// print logs.
// Fields must be paired.
// Deprecated: use With instead.
func (z *ZapLogWrapper) WithFields(fields ...string) Logger {
	return z.With(convertTo(fields...)...)
}

// With add user defined fields to Logger. Fields support multiple values.
func (z *ZapLogWrapper) With(fields ...Field) Logger {
	return z.l.With(fields...)
}

// WithOptions creates a new logger with the provided additional options.
func (z *ZapLogWrapper) WithOptions(opts ...Option) Logger {
	return &ZapLogWrapper{l: z.l.WithOptions(opts...).(*zapLog)}
}

// zapLog is a Logger implementation based on zaplogger.
type zapLog struct {
	levels []zap.AtomicLevel
	logger *zap.Logger
}

func (l *zapLog) WithOptions(opts ...Option) Logger {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return &zapLog{
		levels: l.levels,
		logger: l.logger.WithOptions(zap.AddCallerSkip(o.skip)),
	}
}

// WithFields set some user defined data to logs, such as uid, imei, etc.
// Use this function at the beginning of each request. The returned new Logger should be used to
// print logs.
// Fields must be paired.
// Deprecated: use With instead.
func (l *zapLog) WithFields(fields ...string) Logger {
	return l.With(convertTo(fields...)...)
}

func convertTo(ss ...string) []Field {
	fields := make([]Field, len(ss)/2)
	for i := range fields {
		fields[i] = Field{Key: ss[2*i], Value: ss[2*i+1]}
	}
	return fields
}

// With add user defined fields to Logger. Fields support multiple values.
func (l *zapLog) With(fields ...Field) Logger {
	zapFields := make([]zap.Field, len(fields))
	for i := range fields {
		zapFields[i] = zap.Any(fields[i].Key, fields[i].Value)
	}

	// By ZapLogWrapper proxy, we can add a layer to the debug series function calls, so that the
	// caller information can be set correctly.
	return &ZapLogWrapper{
		l: &zapLog{
			levels: l.levels,
			logger: l.logger.With(zapFields...)}}
}

func getLogMsg(args ...interface{}) string {
	msg := fmt.Sprintln(args...)
	msg = msg[:len(msg)-1]
	report.LogWriteSize.IncrBy(float64(len(msg)))
	return msg
}

func getLogMsgf(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	report.LogWriteSize.IncrBy(float64(len(msg)))
	return msg
}

// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Println.
func (l *zapLog) Trace(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		args, fields := pickZapFields(args)
		l.logger.Debug(getLogMsg(args...), fields...)
	}
}

// Tracef logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func (l *zapLog) Tracef(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		args, fields := pickZapFields(args)
		l.logger.Debug(getLogMsgf(format, args...), fields...)
	}
}

// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Println.
func (l *zapLog) Debug(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		args, fields := pickZapFields(args)
		l.logger.Debug(getLogMsg(args...), fields...)
	}
}

// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func (l *zapLog) Debugf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.DebugLevel) {
		args, fields := pickZapFields(args)
		l.logger.Debug(getLogMsgf(format, args...), fields...)
	}
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Println.
func (l *zapLog) Info(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.InfoLevel) {
		args, fields := pickZapFields(args)
		l.logger.Info(getLogMsg(args...), fields...)
	}
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func (l *zapLog) Infof(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.InfoLevel) {
		args, fields := pickZapFields(args)
		l.logger.Info(getLogMsgf(format, args...), fields...)
	}
}

// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Println.
func (l *zapLog) Warn(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.WarnLevel) {
		args, fields := pickZapFields(args)
		l.logger.Warn(getLogMsg(args...), fields...)
	}
}

// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func (l *zapLog) Warnf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.WarnLevel) {
		args, fields := pickZapFields(args)
		l.logger.Warn(getLogMsgf(format, args...), fields...)
	}
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Println.
func (l *zapLog) Error(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.ErrorLevel) {
		args, fields := pickZapFields(args)
		l.logger.Error(getLogMsg(args...), fields...)
	}
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func (l *zapLog) Errorf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.ErrorLevel) {
		args, fields := pickZapFields(args)
		l.logger.Error(getLogMsgf(format, args...), fields...)
	}
}

// Fatal logs to FATAL log. Arguments are handled in the manner of fmt.Println.
func (l *zapLog) Fatal(args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.FatalLevel) {
		args, fields := pickZapFields(args)
		l.logger.Fatal(getLogMsg(args...), fields...)
	}
}

// Fatalf logs to FATAL log. Arguments are handled in the manner of fmt.Printf.
func (l *zapLog) Fatalf(format string, args ...interface{}) {
	if l.logger.Core().Enabled(zapcore.FatalLevel) {
		args, fields := pickZapFields(args)
		l.logger.Fatal(getLogMsgf(format, args...), fields...)
	}
}

// Sync calls the zap logger's Sync method, and flushes any buffered log entries.
// Applications should take care to call Sync before exiting.
func (l *zapLog) Sync() error {
	return l.logger.Sync()
}

// SetLevel sets output log level.
func (l *zapLog) SetLevel(output string, level Level) {
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
func (l *zapLog) GetLevel(output string) Level {
	i, e := strconv.Atoi(output)
	if e != nil {
		return LevelDebug
	}
	if i < 0 || i >= len(l.levels) {
		return LevelDebug
	}
	return zapLevelToLevel[l.levels[i].Level()]
}

// CustomTimeFormat customize time format.
// Deprecated: Use https://pkg.go.dev/time#Time.Format instead.
func CustomTimeFormat(t time.Time, format string) string {
	return t.Format(format)
}

// DefaultTimeFormat returns the default time format "2006-01-02 15:04:05.000".
// Deprecated: Use https://pkg.go.dev/time#Time.AppendFormat instead.
func DefaultTimeFormat(t time.Time) []byte {
	return defaultTimeFormat(t)
}

func pickZapFields(args []interface{}) ([]interface{}, []zapcore.Field) {
	var fields []zapcore.Field
	var size int
	for idx, arg := range args {
		if field, ok := arg.(zapcore.Field); ok {
			fields = append(fields, field)
			continue
		}
		if size != idx { // Only make copies when there are `zapcore.Field`s present.
			args[size] = arg
		}
		size++
	}
	for i := size; i < len(args); i++ {
		args[i] = nil
	}
	return args[:size], fields
}
