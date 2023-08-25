# Preface

First, welcome to the tRPC-Go development documentation!

The tRPC-Go framework is the Golang version of tRPC, an RPC framework designed with [high performance](todo), [pluggability](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/overview.md), and [ease of testing](todo) in mind. tRPC-Go follows the overall design principles of tRPC. You can use it to:

- Build multiple ports to support multiple protocols (a port can only correspond to a protocol) services, such as [trpc](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/server/overview.md), [http(s)](todo), [grpc](todo) and so on.

- Build a [message queue consumer service](todo) that provides a [message queue producer client](todo) such as [kafka](https://git.woa.com/trpc-go/trpc-database/tree/master/kafka), [rabbitmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rabbitmq), [rocketmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rocketmq), [hippo](https://git.woa.com/trpc-go/trpc-database/tree/master/hippo)，[tdmq](https://git.woa.com/trpc-go/trpc-database/tree/master/tdmq), [tube](https://git.woa.com/trpc-go/trpc-database/tree/master/tube), etc.

- Build local or distributed [timer services](todo).

- Build [streaming services](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/server/stream.md), implement push, file upload, message delivery and other streaming models.

- Access various private protocol [backend services](https://git.woa.com/trpc-go/trpc-codec), call various storage, such as [redis](https://git.woa.com/trpc-go/trpc-database/tree/master/redis), [mysql](https://git.woa.com/trpc-go/trpc-database/tree/master/mysql), [ckv](https://git.woa.com/trpc-go/trpc-database/tree/master/ckv)，[memcache](https://git.woa.com/trpc-go/trpc-database/tree/master/memcache), [mongodb](https://git.woa.com/trpc-go/trpc-database/tree/master/mongodb), etc., using tRPC-Go encapsulated storage interface, more convenient and easier to use.

- Generate stub code and service templates via [trpc tools](https://git.woa.com/trpc-go/trpc-go-cmdline), debug services via [trpc-cli tools](todo), and send commands to services via [admin functions](https://git.woa.com/trpc-go/trpc-wiki/blob/main/user_guide/admin.md).

Now, let's get started on the tRPC-Go journey!

# Quick Start

Before you get started, you should have basic theoretical knowledge, including but not limited to:

- [Go Language Basics](https://books.studygolang.com/gopl-zh/), the cornerstone of everything, make sure to follow the tRPC-Go development specification.

- [context principles](https://draveness.me/golang/docs/part3-runtime/ch06-concurrency/golang-context/), understanding the principle of context will be very helpful for understanding timeout control.

- [RPC Concepts](https://cloud.tencent.com/developer/article/1343888), calling remote service interfaces is like calling local functions and can make it easier for you to create distributed applications.

- [tRPC Terminology](todo) Introduction, as it is important to understand the core concepts in tRPC design in advance, especially the meaning of Service Name and Proto Name, and their interrelationship.

- [proto3 knowledge](https://developers.google.com/protocol-buffers/docs/proto3), a cross-language protocol describing the service interface, is simple, convenient, and universal.

With the above basic theoretical knowledge in mind, we recommend learning tRPC-Go in the following order:

- Quick start: Initially build an understanding of tRPC-Go through a simple Hello World example, and understand the basic process of developing and putting online a backend service.

- Development specifications: Be sure to follow the tRPC-Go development specifications, especially the code specifications inside, to have strict requirements for the quality of your code, it is recommended to repeatedly read and memorize the specification entries inside.

- FAQ: tRPC is an open source development framework, you should check the FAQ first when you encounter problems.

- User Guide: The above steps have enabled you to develop simple services, but not enough, advanced knowledge needs to continue to read in detail to deal with a variety of complex scenarios.

You can find the source code repository of tRPC-Go from [here](https://git.woa.com/trpc-go/trpc-go), and you can read the source code directly.

# Introduction to Terminology

The following terminology is specific to the tRPC Golang language. tRPC terminology common to all languages can be found in: Introduction to tRPC Terminology.

## context

The first parameter of each rpc method is context, which carries context information, including upstream and downstream environment information, timeout information, call chains and other link information, and must be carried in every network call.

Note: The goroutine you start asynchronously must not use the ctx of the request entry, instead you may use `trpc.Go(ctx, timeout, handler)`.


## message

The message structure of each rpc request, containing detailed data about the current request, such as ipport, package header, app server service method and other fields, can be obtained with `trpc.Message(ctx)`.


## caller/upstream and callee/downstream

There are two services A -> B. If A calls B, then A is `caller, the caller, upstream` and B is `callee, the called, downstream`.