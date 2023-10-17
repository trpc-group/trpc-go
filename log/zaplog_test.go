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
	"bytes"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapBufLogger return a buffer logger
func NewZapBufLogger(buf *bytes.Buffer, skip int) Logger {
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)
	return &zapLog{
		levels: []zap.AtomicLevel{},
		logger: zap.New(
			core,
			zap.AddCallerSkip(skip),
			zap.AddCaller(),
		),
	}
}

// NewZapFatalLogger return a fatal hook logger
func NewZapFatalLogger(h zapcore.CheckWriteHook) Logger {
	core, _ := newConsoleCore(&OutputConfig{
		Writer:    "console",
		Level:     "debug",
		Formatter: "console",
	})
	return &zapLog{
		levels: []zap.AtomicLevel{},
		logger: zap.New(
			core,
			zap.AddCallerSkip(1),
			zap.AddCaller(),
			zap.WithFatalHook(h),
		)}
}
