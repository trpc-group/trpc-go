// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package log_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func TestRegister(t *testing.T) {
	assert.Panics(t,
		func() {
			log.Register("panic", nil)
		})
	assert.Panics(t, func() {
		log.Register("dup", log.NewZapLog(log.Config{}))
		log.Register("dup", log.NewZapLog(log.Config{}))
	})
	log.Register("default", log.NewZapLog(log.Config{}))

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

	logger := log.With(log.Field{Key: "uid", Value: "1111"})
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

	ctx := context.Background()
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
	log.WarnContext(ctx, "test")
	log.WarnContextf(ctx, "test")
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
`

func TestLogFactorySetup(t *testing.T) {
	cfg := trpc.Config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	conf := cfg.Plugins["log"]["default"]
	err = plugin.Get("log", "default").Setup("default", &plugin.YamlNodeDecoder{Node: &conf})
	assert.Nil(t, err)

	log.Trace("test")
	log.Debug("test")
	log.Error("test")
	log.Info("test")
	log.Warn("test")

	// set default.
	log.DefaultLogger = log.NewZapLog([]log.OutputConfig{
		{
			Writer:    "console",
			Level:     "debug",
			Formatter: "console",
		},
	})
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
	cfg := trpc.Config{}
	err := yaml.Unmarshal([]byte(illConfigInfo), &cfg)
	assert.Nil(t, err)

	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	// NewRollWriter would return an error if file name is not configured.
	conf := cfg.Plugins["log"]["default"]
	plugin.Get("log", "default").Setup("default", &plugin.YamlNodeDecoder{Node: &conf})
}

type fakeDecoder struct{}

func (c *fakeDecoder) Decode(conf interface{}) error {
	return nil
}

func TestIllegalLogFactory(t *testing.T) {
	cfg := trpc.Config{}
	err := yaml.Unmarshal([]byte(configInfo), &cfg)
	assert.Nil(t, err)

	err = plugin.Get("log", "default").Setup("default", &fakeDecoder{})
	assert.NotNil(t, err)
}
