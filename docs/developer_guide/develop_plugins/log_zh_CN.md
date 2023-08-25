[TOC]

# 前言

本文介绍如何开发日志插件，具体细节可参考[鹰眼日志](https://git.woa.com/trpc-go/trpc-log-atta)，需要提前了解框架 [log](https://git.woa.com/trpc-go/trpc-go/tree/master/log) 的相关概念。

# 原理

框架 `log` 基于 `zap` 实现，支持注册自定义 `writer`。插件是用来适配框架 `log 接口`和`日志平台接口`。

# 具体实现

## 接口定义

```go
// AttaPlugin atta log trpc 插件实现
type AttaPlugin struct {
}
// Type atta log trpc 插件类型
func (p *AttaPlugin) Type() string {
    return "log"
}
// Setup atta 实例初始化
func (p *AttaPlugin) Setup(name string, configDec plugin.Decoder) error {
    ...
}
```

## 核心实现

### 插件初始化

若配置文件的 log 配置了 pluginName 值（见 3.3 注册），框架会在初始化时调用注册 writer 的 `Setup` 方法
具体实现依赖日志平台的初始化，比如：鹰眼日志复用 atta 的通道，这里初始化 atta 即可，为提高运行效率，鹰眼这里实时写管道，异步（支持批量）上报，初始化时启动了 consumer。

```go
// 配置解析，SDK 初始化
...
// 初始化 attaloger
attaLogger := &AttaLogger{
...
}
// zap 注册新插件
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

> 注：可以通过以下方式来完整对 level 的绑定

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

### 写日志

日志上报，`log.ErrorContextf、log.Errorf()`等框架日志接口（log.ErrorContextf 支持额外携带上下文字段），会调用`zapcore.AddSync(attaLogger)`注册的 attaLogger 实例的`Write`方法，注意这里 `p` 的格式受`encoder := zapcore.NewJSONEncoder(encoderCfg)`影响，这里就是 json 字符串。若需要，插件可以实现自己的 encoder。

```go
// Write 写 atta 日志
func (l *AttaLogger) Write(p []byte) (n int, err error) {
  // 上报日志
  ...
   return len(p), nil
}
```

插件将日志内容 p 上报（同步/异步）到自己平台即可。

## 插件注册

注册 writer，pluginName 自定义，AttaPlugin 要满足 3.1 接口定义。

```go
const (
   pluginName = "atta"
)
func init() {
   log.RegisterWriter(pluginName, &AttaPlugin{})
}
```

# 实例

## [鹰眼日志](https://git.woa.com/trpc-go/trpc-log-atta)

## [智研日志](https://git.woa.com/trpc-go/trpc-log-zhiyan)

## [uls 日志](https://git.woa.com/trpc-go/trpc-log-uls)

## [tglog 日志](https://git.woa.com/trpc-go/trpc-log-tglog)

