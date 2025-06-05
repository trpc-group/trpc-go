## 1 前言

本节展示如何使 tRPC-Go 服务调用 flatbuffers 协议服务

## 2 原理

见 [tRPC-Go 搭建 flatbuffers 协议服务](https://iwiki.woa.com/pages/viewpage.action?pageId=976814310) 中的原理介绍

## 3 示例

在 [tRPC-Go 搭建 flatbuffers 协议服务](https://iwiki.woa.com/pages/viewpage.action?pageId=976814310) 的示例部分已经可以生成客户端代码，整体工程目录结构如下：

```shell
├── cmd/client/main.go # 客户端代码
├── go.mod
├── go.sum
├── greeter_2.go       # 第二个 service 的服务端实现
├── greeter_2_test.go  # 第二个 service 的服务端测试
├── greeter.go         # 第一个 service 的服务端实现
├── greeter_test.go    # 第一个 service 的服务端测试
├── main.go            # 服务启动代码
├── stub/git.woa.com/trpcprotocol/testapp/greeter # 桩代码文件
└── trpc_go.yaml       # 配置文件
```

可以参考 `cmd/client/main.go` 来写客户端代码，如下（只选取单发单收的作为例子）：

```go
// Package main 是由 trpc-go-cmdline v2.8.1 生成的客户端示例代码
// 本文件生成于 project/cmd/client 目录下
// 在 project 目录下执行 go run cmd/client/main.go 来运行本文件
// 注意：本文件并非必须存在，而仅为示例，用户应按需进行修改使用，如不需要，可直接删去
package main

import (
    "flag"
    "io"
        
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
    fb "git.woa.com/trpcprotocol/testapp/greeter"
    flatbuffers "github.com/google/flatbuffers/go"
)

func callGreeterSayHello() {
    proxy := fb.NewGreeterClientProxy(
        client.WithTarget("ip://127.0.0.1:8000"),
        client.WithProtocol("trpc"),
    )
    ctx := trpc.BackgroundContext()
    // 一发一收 client 用法示例
    b := flatbuffers.NewBuilder(clientFBBuilderInitialSize)
    // 添加字段示例
    // 将 CreateString 中的 String 替换为你想要操作的字段类型
    // 将 AddMessage 中的 Message 替换为你想要操作的字段名
    // i := b.CreateString("GreeterSayHello")
    fb.HelloRequestStart(b)
    // fb.HelloRequestAddMessage(b, i)
    b.Finish(fb.HelloRequestEnd(b))
    reply, err := proxy.SayHello(ctx, b)
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    // 将 Message 替换为你需要访问的字段名
    // log.Debugf("simple  rpc   receive: %q", reply.Message())
    log.Debugf("simple  rpc   receive: %v", reply)
}

func callGreeterSayHelloStreamClient() {
    proxy := fb.NewGreeterClientProxy(
     client.WithTarget("ip://127.0.0.1:8000"),
     client.WithProtocol("trpc"),
    )
    ctx := trpc.BackgroundContext()
    // 客户端流式 client 用法示例
    stream, err := proxy.SayHelloStreamClient(ctx)
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    for i := 0; i < 5; i++ {
        b := flatbuffers.NewBuilder(clientFBBuilderInitialSize)
        // 添加字段示例
        // 将 CreateString 中的 String 替换为你想要操作的字段类型
        // 将 AddMessage 中的 Message 替换为你想要操作的字段名
        // idx := b.CreateString(fmt.Sprintf("GreeterSayHelloStreamClient %v", i))
        fb.HelloRequestStart(b)
        // fb.HelloRequestAddMessage(b, idx)
        b.Finish(fb.HelloRequestEnd(b))
        if err := stream.Send(b); err != nil {
            log.Fatalf("err: %v", err)
        }
    }
    rsp, err := stream.CloseAndRecv()
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    // 将 Message 替换为你需要访问的字段名
    // log.Debugf("client stream receive: %q", rsp.Message())
    log.Debugf("client stream receive: %v", rsp)
}

func callGreeterSayHelloStreamServer() {
    proxy := fb.NewGreeterClientProxy(
        client.WithTarget("ip://127.0.0.1:8000"),
        client.WithProtocol("trpc"),
    )
    ctx := trpc.BackgroundContext()
    // 服务端流式 client 用法示例
    b := flatbuffers.NewBuilder(clientFBBuilderInitialSize)
    // 添加字段示例
    // 将 CreateString 中的 String 替换为你想要操作的字段类型
    // 将 AddMessage 中的 Message 替换为你想要操作的字段名
    // i := b.CreateString("GreeterSayHelloStreamServer")
    fb.HelloRequestStart(b)
    // fb.HelloRequestAddMessage(b, i)
    b.Finish(fb.HelloRequestEnd(b))
    stream, err := proxy.SayHelloStreamServer(ctx, b)
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    for {
        reply, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatalf("err: %v", err)
        }
        // 将 Message 替换为你需要访问的字段名
        // log.Debugf("server stream receive: %q", reply.Message())
        log.Debugf("server stream receive: %v", reply)
    }
}

func callGreeterSayHelloStreamBidi() {
    proxy := fb.NewGreeterClientProxy(
        client.WithTarget("ip://127.0.0.1:8000"),
        client.WithProtocol("trpc"),
    )
    ctx := trpc.BackgroundContext()
    // 双向流式 client 用法示例
    stream, err := proxy.SayHelloStreamBidi(ctx)
    if err != nil {
        log.Fatalf("err: %v", err)
    }
    for i := 0; i < 5; i++ {
        b := flatbuffers.NewBuilder(clientFBBuilderInitialSize)
        // 添加字段示例
        // 将 CreateString 中的 String 替换为你想要操作的字段类型
        // 将 AddMessage 中的 Message 替换为你想要操作的字段名
        // idx := b.CreateString(fmt.Sprintf("GreeterSayHelloStreamBidi %v", i))
        fb.HelloRequestStart(b)
        // fb.HelloRequestAddMessage(b, idx)
        b.Finish(fb.HelloRequestEnd(b))
        if err := stream.Send(b); err != nil {
            log.Fatalf("err: %v", err)
        }
    }
    if err := stream.CloseSend(); err != nil {
        log.Fatalf("err: %v", err)
    }
    for {
        rsp, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatalf("err: %v", err)
        }
        // 将 Message 替换为你需要访问的字段名
        // log.Debugf(" bidi  stream receive: %q", rsp.Message())
        log.Debugf(" bidi  stream receive: %v", rsp)
    }
}

// clientFBBuilderInitialSize 为 client 端设置 flatbuffers.NewBuilder 初始化大小
var clientFBBuilderInitialSize int

func init() {
    flag.IntVar(&clientFBBuilderInitialSize, "n", 1024, "set client flatbuffers builder's initial size")
}

func main() {
    flag.Parse()
    callGreeterSayHello()
    callGreeterSayHelloStreamClient()
    callGreeterSayHelloStreamServer()
    callGreeterSayHelloStreamBidi()
}
```

整体结构和 protobuf 相关文件基本一致，其中 `"git.woa.com/trpcprotocol/testapp/greeter"` 是桩代码的模块路径，管理方法可参考 protobuf 的桩代码管理

以上为纯客户端的写法，当在一个服务中写下游的客户端时，需要调用的服务信息可以通过 `trpc_go.yaml` 来进行配置，从而省去以下部分

```go
proxy := fb.NewGreeterClientProxy(
    client.WithTarget("ip://127.0.0.1:8000"),
    client.WithProtocol("trpc"),
)
```

## 4 FAQ

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
