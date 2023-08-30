# Introduction

This document introduces how to develop a log plug-in.

You need to know about the log concepts in the framework [trpc-go-log](/log) in advance.

For more details, please refer to https://github.com/trpc-ecosystem/go-log-cls.

# Principles

The framework `log` is implemented based on `zap` and supports registering custom `writers`.

The plug-in is used to adapt the `framework log interface` and the `log platform interface`.

# Implementation

## Interface Definition

```go
    // Plugin cls log trpc
    type Plugin struct {
    }
    // Type cls log trpc
    func (p *Plugin) Type() string {
        return "log"
    }
    // Setup cls
    func (p *Plugin) Setup(name string, configDec plugin.Decoder) error {
        ...
    }
```

## Core Implementation

### setup

If you have set `pluginName` in configuration file, the framework will call the `Setup` function of the `writer` when initializing.

The specific implementation depends on the initialization prrocess of log platform.

To improve the efficiency, cls-log write channels in real time, asynchronously reporting(supports batch), and starts consumers during initialization.

```go
// Parse configuration, SDK initialization
...
// Initialize attaloger
logger := &Logger{
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
   zapcore.AddSync(logger),
   zap.NewAtomicLevelAt(log.Levels[conf.Level]),
)
decoder.Core = c
```

Notice: You can bind level in the following ways.

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

During the log reporting process, the interfaces of the framework, like `log.ErrorContextf()、log.Errorf()` (log.ErrorContextf supports carrying extra context fields), will call the `Write` function of the instance of `logger` registered by `zapcore.AddSync(logger)`.

Notice that the pattern of `p` is configured by `encoder := zapcore.NewJSONEncoder(encoderCfg)`. The plugin could implement its own encoder if necessary.

```go
func (l *Logger) Write(p []byte) (n int, err error) {
  // report the content of the log (p) onto your own platform
  ...
   return len(p), nil
}
```

## Plugin Registration

Register writer, with customized plufginName.

Plugin needs to implement the interface defined in 3.1

```go
const (
   pluginName = "cls"
)
func init() {
   log.RegisterWriter(pluginName, &Plugin{})
}
```
