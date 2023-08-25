[TOC]

# Introduction

This section shows how to call flatbuffers services in tRPC-Go.

# Principles

See [tRPC-Go builds flatbuffers service](user_guide/server/flatbuffers.md).

# 示例

In examples from [tRPC-Go builds flatbuffers service](user_guide/server/flatbuffers.md), we can generate the client. The project structure is as follows:

```shell
├── cmd/client/main.go # client
├── go.mod
├── go.sum
├── greeter_2.go       # second service implementation
├── greeter_2_test.go  # test for second service implementation
├── greeter.go         # first service implementation
├── greeter_test.go    # test for first service implementation
├── main.go            # service startup
├── stub/git.woa.com/trpcprotocol/testapp/greeter # stub code
└── trpc_go.yaml       # configuration file
```

Refer to `cmd/client/main.go` for client code. Here is an example for single-send and single-receive:

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
	// Single-send single-receive client usage
	b := flatbuffers.NewBuilder(clientFBBuilderInitialSize)
	// Add field
	// Replace "String" in "CreateString" with the type of the field
	// Replace "Message" in "AddMessage" with the name of the field
	// i := b.CreateString("GreeterSayHello")
	fb.HelloRequestStart(b)
	// fb.HelloRequestAddMessage(b, i)
	b.Finish(fb.HelloRequestEnd(b))
	reply, err := proxy.SayHello(ctx, b)
	if err != nil {
		log.Fatalf("err: %v", err)
	}
	// Replace "Message" wtih the name of the field
	// log.Debugf("simple  rpc   receive: %q", reply.Message())
	log.Debugf("simple  rpc   receive: %v", reply)
}

// clientFBBuilderInitialSize set initial client-side size for flatbuffers.NewBuilder
var clientFBBuilderInitialSize int

func init() {
	flag.IntVar(&clientFBBuilderInitialSize, "n", 1024, "set client flatbuffers builder's initial size")
}

func main() {
	flag.Parse()
	callGreeterSayHello()
}
```

The structure is similar to protobuf, where `"git.woa.com/trpcprotocol/testapp/greeter"` is the importpath for stubs. Refer to protobuf for managing stubs.

Above is pure client-side implementation. When using client in a service, downstream services can be set in `trpc_go.yaml`, and no longer needing the code below:

```go
proxy := fb.NewGreeterClientProxy(
	client.WithTarget("ip://127.0.0.1:8000"),
	client.WithProtocol("trpc"),
)
```

# FAQ

