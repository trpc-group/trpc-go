[TOC]

# tRPC-Go 开发分布式追踪插件

## 介绍

本文介绍的是如何开发分布式追踪链路插件。

利用 tRPC-Go 过滤器能力，在请求前打点，请求后打点并上报，完全使用 [opentracing](https://github.com/opentracing/opentracing-go) 的标准接口，跟具体追踪的实现进行解耦。

## 实现

首先需要在框架启动时，初始化插件，将具体 tracer 实例注册到 opentracing 中：

```go
func (p *Plugin) Setup(name string, decoder plugin.Decoder) error {
    tracer := xxx.NewTracer()  // 具体平台的实现，如 OpenTelemetry
    opentracing.SetGlobalTracer(tracer)  // 注册到 opentracing 即可，后续操作直接使用 opentracing 的接口
}
```

### 解析上下文 SpanContext

从 tRPC 协议的头部解析出跟踪的上下文信息 `SpanContext`，如果没有这些内容，那么本服务为 `Root`。

```go
// 在 server 拦截器里面解析 span
func serverFilter(ctx context.Context, req interface{}, rsp interface{}) error {
    msg := trpc.Message(ctx) 
    parentSpanContext, err := tracer.Extract(opentracing.TextMap, metadataTextMap(md)) // err != nil 说明是 root
}
```

### Create new Span

利用 opentracing 接口 `StartSpan` 创建 `Span `实例，必要参数 `Name`，在 tRPC-Go 中填充的是被调用服务方法名。

```go
serverSpan := tracer.StartSpan(msg.CalleeMethod())
```

此外在 `StartSpan ` 还可以指定一系列的 `StartSpanOption`，用于设置 Span 的类型，以及 Tag 附加信息。（对端 ip，本机端口，tRPC 环境名 等信息）

```go
spanOpt := []opentracing.StartSpanOption{ 
    ext.RPCServerOption(parentSpanContext), 
    opentracing.Tag{Key: string(TraceExtNamespace), Value: trpc.GlobalConfig().Global.Namespace}, 
    opentracing.Tag{Key: string(TraceExtEnvName), Value: trpc.GlobalConfig().Global.EnvName}, 
} 
serverSpan := tracer.StartSpan(msg.CalleeMethod())
```

### Span 注入 ctx

更新 ctx，将 span 注入到 ctx 中，这个的目的是我们在业务逻辑处理时可以重新拿到 span，然后进行 Tag 上报，日志上报等逻辑。

```go
ctx = opentracing.ContextWithSpan(ctx, serverSpan)
```

### 调用业务逻辑

### Span 上报

业务逻辑处理完成，进行 `Span` 的上报（调用 `Finish` 方法），如果出现错误，可以记录一下标签和 Log。

```go
if err != nil { 
    ext.Error.Set(serverSpan, true) 
    serverSpan.LogFields(tracelog.String("event", "error"), tracelog.String("message", err.Error())) 
} 
serverSpan.Finish()
```

## 示例

## Owner

tensorchen
