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

import "trpc.group/trpc-go/trpc-go/log"

func Example() {
	l := log.NewZapLog([]log.OutputConfig{
		{
			Writer:       "console",
			Level:        "debug",
			Formatter:    "console",
			FormatConfig: log.FormatConfig{TimeFmt: "xxx"},
		},
	})
	const defaultLoggerName = "default"
	oldDefaultLogger := log.GetDefaultLogger()
	log.Register(defaultLoggerName, l)
	defer func() {
		log.Register(defaultLoggerName, oldDefaultLogger)
	}()

	l = log.With(log.Field{Key: "tRPC-Go", Value: "log"})
	l.Trace("hello world")
	l.Debug("hello world")
	l.Info("hello world")
	l.Warn("hello world")
	l.Error("hello world")
	l.Tracef("hello world")
	l.Debugf("hello world")
	l.Infof("hello world")
	l.Warnf("hello world")
	l.Errorf("hello world")

	// Output:
	// xxx	DEBUG	log/example_test.go:35	hello world	{"tRPC-Go": "log"}
	// xxx	DEBUG	log/example_test.go:36	hello world	{"tRPC-Go": "log"}
	// xxx	INFO	log/example_test.go:37	hello world	{"tRPC-Go": "log"}
	// xxx	WARN	log/example_test.go:38	hello world	{"tRPC-Go": "log"}
	// xxx	ERROR	log/example_test.go:39	hello world	{"tRPC-Go": "log"}
	// xxx	DEBUG	log/example_test.go:40	hello world	{"tRPC-Go": "log"}
	// xxx	DEBUG	log/example_test.go:41	hello world	{"tRPC-Go": "log"}
	// xxx	INFO	log/example_test.go:42	hello world	{"tRPC-Go": "log"}
	// xxx	WARN	log/example_test.go:43	hello world	{"tRPC-Go": "log"}
	// xxx	ERROR	log/example_test.go:44	hello world	{"tRPC-Go": "log"}
}
