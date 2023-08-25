[TOC]

# tRPC-Go Develop distributed tracing plugin

## Introduction

This page described how to implement a distributed tracing plugin.

By utilziing tRPC-Go's custom filter, we can add instrumentation before and after a request being processed with standard [opentracing](https://github.com/opentracing/opentracing-go) interfrace.

## Implementation

On framework startup, initialize and register the `tracer` instance into opentracing:

```go
func (p *Plugin) Setup(name string, decoder plugin.Decoder) error {
    tracer := xxx.NewTracer()  // Specific platform implementation, e.g. OpenTelemtry
    opentracing.SetGlobalTracer(tracer)  // Register to opentracing, use public interface afterwards
}
```

### Extract SpanContext

Extract `SpanContext` from tRPC protocol header. If span is not present, this is `Root`.

```go
// Extract SpanContext in server filter
func serverFilter(ctx context.Context, req interface{}, rsp interface{}) error {
    msg := trpc.Message(ctx) 
    parentSpanContext, err := tracer.Extract(opentracing.TextMap, metadataTextMap(md)) // err != nil indicates this is root
}
```

### Create new Span

Use opentracing `StartSpan` to create a new `Span` instance. Use tRPC callee method name as the `Name` argument.

```go
serverSpan := tracer.StartSpan(msg.CalleeMethod())
```

We can also provide a set of `StartSpanOption` to `StartSpan`, to set `Span` type and additional information. (peer IP, local port, tRPC environment name, etc.)

```go
spanOpt := []opentracing.StartSpanOption{ 
    ext.RPCServerOption(parentSpanContext), 
    opentracing.Tag{Key: string(TraceExtNamespace), Value: trpc.GlobalConfig().Global.Namespace}, 
    opentracing.Tag{Key: string(TraceExtEnvName), Value: trpc.GlobalConfig().Global.EnvName}, 
} 
serverSpan := tracer.StartSpan(msg.CalleeMethod())
```

### Inject Span into Context

Inject created `Span` into `ctx`, so the user handler function can obtain the correct `Span` and do tag / log reports.

```go
ctx = opentracing.ContextWithSpan(ctx, serverSpan)
```

### Call user handler function

### Report Span

After user handler function finished, report the `Span` by calling `Finish()`. If the handler returned error, log the error message.

```go
if err != nil { 
    ext.Error.Set(serverSpan, true) 
    serverSpan.LogFields(tracelog.String("event", "error"), tracelog.String("message", err.Error())) 
} 
serverSpan.Finish()
```

## Example

