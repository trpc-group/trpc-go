# 前言

本文介绍如何开发日志插件，具体细节可参考[CLS日志](https://github.com/trpc-ecosystem/go-log-cls)，需要提前了解框架 [log](/log) 的相关概念。

# 原理

框架 `log` 基于 `zap` 实现，支持注册自定义 `writer`。插件是用来适配框架 `log 接口`和`日志平台接口`。

# 具体实现

## 接口定义

```go
// Plugin cls log trpc 插件实现
type Plugin struct {
}
// Type cls log trpc 插件类型
func (p *Plugin) Type() string {
    return "log"
}
// Setup cls 实例初始化
func (p *Plugin) Setup(name string, configDec plugin.Decoder) error {
    ...
}
```

## 核心实现

### 插件初始化

若配置文件的 log 配置了 pluginName 值（见注册一节），框架会在初始化时调用注册 writer 的 `Setup` 方法
具体实现依赖日志平台的初始化。

```go
// 配置解析，SDK 初始化
...
// 初始化 attaloger
logger := &Logger{
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
   zapcore.AddSync(logger),
   zap.NewAtomicLevelAt(log.Levels[conf.Level]),
)
decoder.Core = c
```

> 注：可以通过以下方式来完成对 level 的绑定

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

日志上报，`log.ErrorContextf、log.Errorf()`等框架日志接口（log.ErrorContextf 支持额外携带上下文字段），会调用`zapcore.AddSync(logger)`注册的 logger 实例的`Write`方法，注意这里 `p` 的格式受`encoder := zapcore.NewJSONEncoder(encoderCfg)`影响，这里就是 json 字符串。若需要，插件可以实现自己的 encoder。

```go
func (l *Logger) Write(p []byte) (n int, err error) {
  // 上报日志
  ...
   return len(p), nil
}
```

插件将日志内容 p 上报（同步/异步）到自己平台即可。

## 插件注册

注册 writer，pluginName 自定义，AttaPlugin 要满足接口定义一节。

```go
const (
   pluginName = "cls"
)
func init() {
   log.RegisterWriter(pluginName, &Plugin{})
}
```
