// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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
	// xxx	DEBUG	log/example_test.go:22	hello world	{"tRPC-Go": "log"}
	// xxx	DEBUG	log/example_test.go:23	hello world	{"tRPC-Go": "log"}
	// xxx	INFO	log/example_test.go:24	hello world	{"tRPC-Go": "log"}
	// xxx	WARN	log/example_test.go:25	hello world	{"tRPC-Go": "log"}
	// xxx	ERROR	log/example_test.go:26	hello world	{"tRPC-Go": "log"}
	// xxx	DEBUG	log/example_test.go:27	hello world	{"tRPC-Go": "log"}
	// xxx	DEBUG	log/example_test.go:28	hello world	{"tRPC-Go": "log"}
	// xxx	INFO	log/example_test.go:29	hello world	{"tRPC-Go": "log"}
	// xxx	WARN	log/example_test.go:30	hello world	{"tRPC-Go": "log"}
	// xxx	ERROR	log/example_test.go:31	hello world	{"tRPC-Go": "log"}
}
