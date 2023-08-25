[TOC]

# Introduction

This document introduces how to develop a log plug-in.

You need to know about the log concepts in the framework [trpc-go-log](https://git.woa.com/trpc-go/trpc-go/tree/master/log) in advance.

For more details, please refer to https://git.woa.com/trpc-go/trpc-log-atta.

# Principles

The framework `log` is implemented based on `zap` and supports registering custom `writers`.

The plug-in is used to adapt the `framework log interface` and the `log platform inerface`.

# Implementation

## Interface Definition

```go
    // AttaPlugin atta log trpc
    type AttaPlugin struct {
    }
    // Type atta log trpc
    func (p *AttaPlugin) Type() string {
        return "log"
    }
    // Setup atta
    func (p *AttaPlugin) Setup(name string, configDec plugin.Decoder) error {
        ...
    }
```

## Core Implementation

### setup

If you have set `pluginName` in configuration file, the framework will call the `Setup` function of the `writer` when initializing.

The specific implementation depends on the initilization prrocess of log platform. For example, `log-atta` reuses the channel of atta, we only need to initialize atta.

To improve the efficiency, atta-log write channels in real time, asynchronously reporting(supports batch), and starts consumers during initialization.

```go
// Parse configuration, SDK initialization
...
// Initialize attaloger
attaLogger := &AttaLogger{
...
}
// zap register new plugin
encoderCfg := zapcore.EncoderConfig{
   TimeKey:        cfg.TimeKey,
   LevelKey:       cfg.LevelKey,
   ...
}
encoder := zapcore.NewJSONEncoder(encoderCfg)
c := zapcore.NewCore(
   encoder,
   zapcore.AddSync(attaLogger),
   zap.NewAtomicLevelAt(log.Levels[conf.Level]),
)
decoder.Core = c
```

Notice: You can fully bind level in the following ways.

```go
encoder := zapcore.NewJSONEncoder(encoderCfg)
zl := zap.NewAtomicLevelAt(log.Levels[conf.Level])
decoder.Core = zapcore.NewCore(
    encoder,
    zapcore.AddSync(clsLogger),
    zl,
)
decoder.ZapLevel = zl
```

### write

During the log reporting process, the interfaces of the framework, like `log.ErrorContextf()、log.Errorf()` (log.ErrorContextf supports carrying extra context fields), will call the `Write` function of the instance of `attaLogger` registered by `zapcore.AddSync(attaLogger)`.

Notice that the pattern of `p` is configured by `encoder := zapcore.NewJSONEncoder(encoderCfg)`. The plugin could implement its own encoder if necessary.

```go
// Write log
func (l *AttaLogger) Write(p []byte) (n int, err error) {
  // report the content of the log (p) onto your own platform
  ...
   return len(p), nil
}
```

## Plugin Registration

Register writer, with customized plufginName.

AttaPlugin needs to implement the interface defined in 3.1

```go
const (
   pluginName = "atta"
)
func init() {
   log.RegisterWriter(pluginName, &AttaPlugin{})
}
```

# Example

## [log-atta](https://git.woa.com/trpc-go/trpc-log-atta)

## [log-zhiyan](https://git.woa.com/trpc-go/trpc-log-zhiyan)

## [log-uls](https://git.woa.com/trpc-go/trpc-log-uls)

## [log-tglog](https://git.woa.com/trpc-go/trpc-log-tglog)

# OWNER

evannzhang
