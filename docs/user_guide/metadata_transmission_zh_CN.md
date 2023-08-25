[TOC]

# 前言
全链路透传：框架支持在 client 和 server 之间透传字段，并在整个调用链路自动透传下去。
字段以 key-value 的形式存在，key 是 string 类型，value 是 []byte 类型，value 可以是任何数据，透传字段对于 RPC 请求来说是透明的，提供了关于本次 RPC 请求的额外信息。同时框架通过 ctx 来透传字段。

下面文档描述怎么样在框架中实现字段透传。

# 原理及实现
通过 tRPC 协议头部中的 transinfo 字段来透传数据，用户把需要透传的字段通过框架的 api 设置到 context 里面，框架在打解包的时候，会把用户设置的字段设置到协议相应的字段上面，然后进行透传，收到数据的一方会把对应的透传字段解析出来，用户可以通过接口获取到透传的数据。

# 示例
##  client 透传数据到 server
client 发起请求时，通过增加 option 来设置透传字段，可以增加多个透传字段。

```go
options := []client.Option{
	client.WithMetaData("key1", []byte("val1")),
	client.WithMetaData("key2", []byte("val2")),
	client.WithMetaData("key3", []byte("val3")),
}

rsp, err := proxy.SayHello(ctx, options...) // 注意：框架传过来的 ctx
```

下游 server 通过框架的 ctx 可以获取到 client 的透传字段

```go
trpc.GetMetaData(ctx, "key1") // 注意使用框架的 ctx，上面 client 设置了 key1  的值为 val1，这里将会得到 val1 的返回值，如果 client 没有设置对应的值，则返回空数据。
```

## server 透传数据到 client
server 在回包的时候可以通过 ctx 设置透传字段返回给上游调用方

```go
trpc.SetMetaData(ctx, "key1", []byte("val1")) // 注意使用框架的 ctx，通过这个 api 设置了透传字段 key1 的值为 []byte("val1") 
```

上游 client 可以通过设置各协议的 rsp head 获取

```go
head := &trpc.ResponseProtocol{}
options := []client.Option{
	client.WithRspHead(head),
}

rsp, err := proxy.SayHello(ctx, options...) // 注意：框架传过来的 ctx
head.TransInfo // 框架透传回来的信息 key-value 对（map[string][]byte）
```

