[TOC]

# 前言

本节展示如何使 tRPC-Go 服务调用 flatbuffers 协议服务

# 原理

见 [tRPC-Go 搭建 flatbuffers 协议服务](user_guide/server/flatbuffers.md) 中的原理介绍

# 示例

在 [tRPC-Go 搭建 flatbuffers 协议服务](user_guide/server/flatbuffers.md) 的示例部分已经可以生成客户端代码，整体工程目录结构如下：

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
package main

import (
	"flag"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"

	trpc "git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/log"
	fb "git.woa.com/trpcprotocol/testapp/greeter"
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

// clientFBBuilderInitialSize 为 client 端设置 flatbuffers.NewBuilder 初始化大小
var clientFBBuilderInitialSize int

func init() {
	flag.IntVar(&clientFBBuilderInitialSize, "n", 1024, "set client flatbuffers builder's initial size")
}

func main() {
	flag.Parse()
	callGreeterSayHello()
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

# FAQ

