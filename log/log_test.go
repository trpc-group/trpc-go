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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func TestSetLevel(t *testing.T) {
	const level = "0"
	log.SetLevel(level, log.LevelInfo)
	require.Equal(t, log.LevelInfo, log.GetLevel(level))
}

func TestSetLogger(t *testing.T) {
	logger := log.NewZapLog(log.Config{})
	log.SetLogger(logger)
	require.Equal(t, log.GetDefaultLogger(), logger)
}

func TestLogXXX(t *testing.T) {
	log.Fatal("xxx")
}

func TestLoggerNil(t *testing.T) {
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithLogger((*log.ZapLogWrapper)(nil))

	log.EnableTrace()
	log.TraceContext(ctx, "test")
	log.TraceContextf(ctx, "test %s", "log")
	log.DebugContext(ctx, "test")
	log.DebugContextf(ctx, "test %s", "log")
	log.InfoContext(ctx, "test")
	log.InfoContextf(ctx, "test %s", "log")
	log.ErrorContext(ctx, "test")
	log.ErrorContextf(ctx, "test %s", "log")
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test %s", "log")
	log.FatalContext(ctx, "test")
	log.FatalContextf(ctx, "test %s", "log")

	l := msg.Logger()
	require.Nil(t, l)
}

func TestLoggerZapLogWrapper(t *testing.T) {
	ctx := context.Background()
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithLogger(log.NewZapLog(defaultConfig))

	log.EnableTrace()
	log.TraceContext(ctx, "test")
	log.TraceContextf(ctx, "test")
	log.DebugContext(ctx, "test")
	log.DebugContextf(ctx, "test")
	log.InfoContext(ctx, "test")
	log.InfoContextf(ctx, "test %s", "s")
	log.ErrorContext(ctx, "test")
	log.ErrorContextf(ctx, "test")
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test")

	msg.WithLogger(log.NewZapLog(defaultConfig))
	log.WithContextFields(ctx, "a", "a")
	log.SetLevel("console", log.LevelDebug)
	require.Equal(t, log.GetLevel("console"), log.LevelDebug)
}

func TestWithContextFields(t *testing.T) {
	ctx := trpc.BackgroundContext()
	log.WithContextFields(ctx, "k", "v")
	require.NotNil(t, codec.Message(ctx).Logger())

	ctx = context.Background()
	newCtx := log.WithContextFields(ctx, "k", "v")
	require.Nil(t, codec.Message(ctx).Logger())
	require.NotNil(t, codec.Message(newCtx).Logger())
}

func TestLogFactory(t *testing.T) {

	log.EnableTrace()

	f := &log.Factory{}

	assert.Equal(t, "log", f.Type())

	// empty decoder
	err := f.Setup("default", nil)
	assert.NotNil(t, err)

	log.Register("default", log.DefaultLogger)
	assert.Equal(t, log.DefaultLogger, log.Get("default"))
	assert.Nil(t, log.Get("empty"))
	log.Sync()

	logger := log.WithFields("uid", "1111")
	assert.NotNil(t, logger)
	logger.Debugf("test")

	log.Trace("test")
	log.Tracef("test %s", "s")
	log.Debug("test")
	log.Debugf("test %s", "s")
	log.Error("test")
	log.Errorf("test %s", "s")
	log.Info("test")
	log.Infof("test %s", "s")
	log.Warn("test")
	log.Warnf("test %s", "s")
	log.Fatal("test %s", "s")
	log.Fatalf("test %s", "s")

	ctx := context.Background()
	log.TraceContext(ctx, "test")
	log.TraceContextf(ctx, "test")
	log.DebugContext(ctx, "test")
	log.DebugContextf(ctx, "test")
	log.InfoContext(ctx, "test")
	log.InfoContextf(ctx, "test %s", "s")
	log.ErrorContext(ctx, "test")
	log.ErrorContextf(ctx, "test")
	log.FatalContext(ctx, "test")
	log.FatalContextf(ctx, "test")
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test")
	log.WithFieldsContext(ctx, "field", "testfield").Debugf("testdebug")
	log.WithFieldsContext(ctx, "field", "testfield").
		WithFields("field2", "testfield2").Debugf("testdebug")
	log.WithContext(ctx, log.Field{Key: "abc", Value: 123}).Debug("testdebug")

	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCallerServiceName("trpc.test.helloworld.Greeter")
	log.WithContextFields(ctx, "a", "a")
	log.TraceContext(ctx, "test")
	log.TraceContextf(ctx, "test")
	log.DebugContext(ctx, "test")
	log.DebugContextf(ctx, "test")
	log.InfoContext(ctx, "test")
	log.InfoContextf(ctx, "test %s", "s")
	log.ErrorContext(ctx, "test")
	log.ErrorContextf(ctx, "test")
	log.FatalContext(ctx, "test")
	log.FatalContextf(ctx, "test")
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test")
	log.WithFieldsContext(ctx, "test")
	log.WithFieldsContext(ctx, "field", "testfield").Debugf("testdebug")
	log.WithFieldsContext(ctx, "field", "testfield").
		WithFields("field2", "testfield2").Debugf("testdebug")

}

func TestWriterFactory(t *testing.T) {
	t.Run("console", func(t *testing.T) {
		f := &log.ConsoleWriterFactory{}
		require.Equal(t, "log", f.Type())

		err := f.Setup("default", nil)
		require.Contains(t, err.Error(), "decoder empty")
	})
	t.Run("file", func(t *testing.T) {
		f := &log.FileWriterFactory{}
		require.Equal(t, "log", f.Type())

		err := f.Setup("default", nil)
		require.Contains(t, err.Error(), "decoder empty")
	})

}

const configInfo = `
plugins:
  log:
    default:
     - writer: console # default as console std output
       level: debug # std log level
     - writer: file # local log file
       level: debug # std log level
       writer_config: # config of local file output
         filename: trpc_time.log # the path of local rolling log files
         roll_type: time    # file rolling type
         max_age: 7         # max expire days
         time_unit: day     # rolling time interval
     - writer: file # local file log
       level: debug # std output log level
       writer_config: # config of local file output
         filename: trpc_size.log # the path of local rolling log files
         roll_type: size    # file rolling type
         max_age: 7         # max expire days
         max_size: 100      # size of local rolling file, unit MB
         max_backups: 10    # max number of log files
         compress:  false   # should compress log file
     - writer: file # local file log
       level: debug # std output log level
       writer_config: # config of local file output
         filename: "trpc_size_{time_format}.log" # the path of local rolling log files
         roll_type: time    # file rolling type
         max_age: 7         # max expire days
         max_size: 100      # size of local rolling file, unit MB
         max_backups: 10    # max number of log files
         compress:  false   # should compress log file
`

func TestLogFactorySetup(t *testing.T) {
	oldDefaultLogger := log.GetDefaultLogger()
	defer func() {
		log.Register("default", oldDefaultLogger)
	}()

	var cfg trpc.Config
	mustYamlUnmarshal(t, []byte(configInfo), &cfg)
	conf := cfg.Plugins["log"]["default"]
	err := plugin.Get("log", "default").Setup("default", &plugin.YamlNodeDecoder{Node: &conf})
	assert.Nil(t, err)

	log.Trace("test")
	log.Debug("test")
	log.Error("test")
	log.Info("test")
	log.Warn("test")
}

func TestIllegalLogFactory(t *testing.T) {
	var cfg trpc.Config
	mustYamlUnmarshal(t, []byte(configInfo), &cfg)
	err := plugin.Get("log", "default").Setup("default", &fakeDecoder{})
	require.Contains(t, err.Error(), "log config output empty")
}

const illConfigInfo = `
plugins:
  log:
    default:
     - writer: file # local file log
       level: debug # std output log level
       writer_config: # config of local file output
         filename:  # path of local file rolling log files
         roll_type: time    # rolling file type
         max_age: 7         # max expire days
         time_unit: day     # rolling time interval
`

func TestIllLogConfigPanic(t *testing.T) {
	var cfg trpc.Config
	mustYamlUnmarshal(t, []byte(illConfigInfo), &cfg)
	conf := cfg.Plugins["log"]["default"]
	require.Panicsf(t, func() {
		plugin.Get("log", "default").Setup("default", &plugin.YamlNodeDecoder{Node: &conf})
	}, "NewRollWriter would return an error if file name is not configured")
}

type fakeDecoder struct{}

func (c *fakeDecoder) Decode(conf interface{}) error {
	return nil
}

func mustYamlUnmarshal(t *testing.T, in []byte, out interface{}) {
	t.Helper()

	if err := yaml.Unmarshal(in, out); err != nil {
		t.Fatal(err)
	}
}
