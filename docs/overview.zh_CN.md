## 1 前言

首先，欢迎大家进入 tRPC-Go 的开发文档！

tRPC-Go 框架是 tRPC 的 Golang 版本，主要是以 [高性能](https://iwiki.woa.com/pages/viewpage.action?pageId=99485677)，[可插拔](https://iwiki.woa.com/pages/viewpage.action?pageId=99485612)，[易测试](https://iwiki.woa.com/pages/viewpage.action?pageId=119530324) 为出发点而设计的 RPC 框架。tRPC-Go 完全遵循 tRPC 的整体设计原则。你可以使用它：

- 搭建多个端口支持多个协议（一个端口只能对应一个协议）的服务，如 [trpc](https://iwiki.woa.com/pages/viewpage.action?pageId=284289102)，[http](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)，[http2](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)，[https](https://iwiki.woa.com/pages/viewpage.action?pageId=490796278)，[grpc](https://iwiki.woa.com/pages/viewpage.action?pageId=284289174) 及各种腾讯内部私有协议：[tars](https://iwiki.woa.com/pages/viewpage.action?pageId=410399255)，[oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb)，[wns](https://git.woa.com/trpc-go/trpc-codec/tree/master/wns)，[qzone](https://git.woa.com/trpc-go/trpc-codec/tree/master/qzh)，[sso](https://git.woa.com/trpc-go/trpc-codec/tree/master/sso) 等等。
- 搭建消息队列 [消费者服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289140)，提供消息队列 [生产者客户端](https://iwiki.woa.com/pages/viewpage.action?pageId=284289134)，如 [kafka](https://git.woa.com/trpc-go/trpc-database/tree/master/kafka)，[rabbitmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rabbitmq)，[rocketmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rocketmq)，[hippo](https://git.woa.com/trpc-go/trpc-database/tree/master/hippo)，[tdmq](https://git.woa.com/trpc-go/trpc-database/tree/master/tdmq)，[tube](https://git.woa.com/trpc-go/trpc-database/tree/master/tube) 等等。
- 搭建本地定时器，分布式 [定时器服务](https://iwiki.woa.com/pages/viewpage.action?pageId=284289170)。
- 搭建 [流式服务](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=284289215)，实现 push，文件上传，消息下发等流式模型。
- 访问各种私有协议 [后端服务](https://git.woa.com/trpc-go/trpc-codec)，调用各种 [存储](https://iwiki.woa.com/pages/viewpage.action?pageId=284289130)，如 [redis](https://git.woa.com/trpc-go/trpc-database/tree/master/redis)，[mysql](https://git.woa.com/trpc-go/trpc-database/tree/master/mysql)，[ckv](https://git.woa.com/trpc-go/trpc-database/tree/master/ckv)，[memcache](https://git.woa.com/trpc-go/trpc-database/tree/master/memcache)，[mongodb](https://git.woa.com/trpc-go/trpc-database/tree/master/mongodb) 等等，使用 tRPC-Go 封装的存储接口，使用起来更方便更简单。
- 通过 [trpc 工具](https://git.woa.com/trpc-go/trpc-go-cmdline) 生成桩代码和服务模板，通过 [trpc-cli 工具](https://iwiki.woa.com/pages/viewpage.action?pageId=346696646) 调试服务，通过 [admin 功能](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=99485663) 给服务发送指令。

现在，开始进入 tRPC-Go 之旅吧！

**注：** v0.18.x 为 trpc-go 的 LTS (Long Term Support, 长期维护) 版本

## 2 快速开始

在真正开始之前，首先需要掌握基本理论知识，包括但不限于：

- [Go 语言基础](https://books.studygolang.com/gopl-zh/)，所有一切的基石，务必遵循 [tRPC-Go 研发规范](https://iwiki.woa.com/p/99485634)。
- [context 原理](https://draveness.me/golang/docs/part3-runtime/ch06-concurrency/golang-context/)，必须提前了解，特别是对超时控制的理解会很有帮助。
- [RPC 概念](https://cloud.tencent.com/developer/article/1343888)，调用远程服务接口就像调用本地函数一样，能让你更容易创建分布式应用。
- [tRPC 术语介绍](https://iwiki.woa.com/pages/viewpage.action?pageId=490794774 "tRPC 术语")，必须提前了解 tRPC 设计中的核心概念，尤其是 Service Name 和 Proto Name 的含义，以及相互关系。
- [proto3 知识](https://developers.google.com/protocol-buffers/docs/proto3)，描述服务接口的跨语言协议，简单，方便，通用。

掌握好以上基本理论知识以后，建议按以下推荐顺序开始学习 tRPC-Go：

- 快速上手：通过一个简单的 Hello World 例子初步建立对 tRPC-Go 的认识，了解开发并上线一个后台服务的基本流程。
- 研发规范：务必一定遵守 tRPC-Go 研发规范，特别是里面的代码规范，要对自己的代码质量有严格的要求，推荐反复阅读并熟记里面的规范条目。
- 常见问题：tRPC 是开源共建的开发框架，碰到问题应该首先查看常见问题，没有找到再到 [码客](http://mk.oa.com/coterie/420?offset=3) 上面先搜索再 [提问](https://mk.woa.com/q/new?coterie=420)，码客`圈子`指定`tRPC 微服务框架`。
- 用户指南：通过以上步骤已经能够开发简单服务，但还不够，进阶知识需要继续详细阅读以应对各种各样的复杂场景。

你可以从 [这里](https://git.woa.com/trpc-go/trpc-go) 找到 tRPC-Go 的源码库，可以直接阅读源码。

## 3 术语介绍

以下术语为 tRPC Golang 语言特有的概念。tRPC 所有语言通用术语请参考：[tRPC 术语介绍](https://iwiki.woa.com/pages/viewpage.action?pageId=490794774 "tRPC 术语介绍")

### 3.1 context

请求上下文，在每个 rpc 方法第一个参数都是 context，里面会携带上下文信息，包括上下游环境信息，超时信息，调用链等其他链路信息，在每一次的网络调用都必须携带该 ctx 进行调用。
***`注意：自己异步启动的 goroutine 一定不要使用请求入口的 ctx，可以使用 trpc.Go(ctx, timeout, handler)`。***

### 3.2 message

每一次 rpc 请求的消息结构体，包含了当前请求的详细数据，如 ipport，包头，app server service method 等字段，可以通过`trpc.Message(ctx)`获取。

### 3.3 caller callee 主调 被调 上游 下游

有两个服务 A -> B，A 调用 B，则 A 是`caller，主调方，上游`，B 是`callee，被调方，下游`

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
