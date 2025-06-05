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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/internal/env"
)

func Test_traceEnableFromEnv(t *testing.T) {
	t.Run("disable trace", func(t *testing.T) {
		t.Setenv(env.LogTrace, "0")
		require.False(t, traceEnableFromEnv())
	})
	t.Run("enable trace", func(t *testing.T) {
		t.Setenv(env.LogTrace, "1")
		require.True(t, traceEnableFromEnv())
	})
	t.Run("empty env", func(t *testing.T) {
		t.Setenv(env.LogTrace, "")
		require.False(t, traceEnableFromEnv())
	})
	t.Run("other env", func(t *testing.T) {
		t.Setenv(env.LogTrace, "xxx")
		require.True(t, traceEnableFromEnv())
	})
}

func TestSetTraceEnabled(t *testing.T) { // set logger to file
	defer setTraceEnabled(false)
	logDir := t.TempDir()
	defaultLogger := DefaultLogger
	defer SetLogger(defaultLogger)
	logger := NewZapLog(Config{
		{
			Writer: OutputFile,
			WriteConfig: WriteConfig{
				LogPath:   logDir,
				Filename:  "trpc.",
				WriteMode: WriteSync,
			},
			Level: "debug",
		},
	})
	SetLogger(logger)

	// debug  ensure  file exists.
	Debug("debug msg")

	// trace is disable, msg will not exist in  file.
	setTraceEnabled(false)
	Trace("trace msg1")
	fp := filepath.Join(logDir, "trpc.")
	buf, err := os.ReadFile(fp)
	require.Nil(t, err)
	require.NotContains(t, string(buf), "trace msg1")

	// enable trace, msg will exist in  file.
	setTraceEnabled(true)
	Trace("trace msg2")
	buf, err = os.ReadFile(fp)
	require.Nil(t, err)
	require.Contains(t, string(buf), "trace msg2")

	// disable trace, msg will  not exist in  file.
	setTraceEnabled(false)
	Trace("trace msg3")
	buf, err = os.ReadFile(fp)
	require.Nil(t, err)
	require.NotContains(t, string(buf), "trace msg3")
}
