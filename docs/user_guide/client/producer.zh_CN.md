## 1 前言

业务开发中，为了实现服务间解耦、异步处理、消峰等功能，很多场景都会选择消息队列（MQ, Message Queue），tRPC-Go 组件已经很好地支持了这些场景，各个组件代码详情参照 [trpc-database](https://git.woa.com/trpc-go/trpc-database)。

截止目前已经支持的消息队列组件如下（最新组件列表请到 git 仓库中查阅）:

| 名称 | 描述|
| :----: | :----   |
| [kafka](https://git.woa.com/trpc-go/trpc-database/tree/master/kafka) | 开源消息队列 |
| [hippo](https://git.woa.com/trpc-go/trpc-database/tree/master/hippo) | PCG 类 kafka 消息队列 |
| [rabbitmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rabbitmq) | 开源消息队列 |

## 2 原理

在 tRPC-Go 中生产者是通过 Client 实现的，tRPC-Go 中的客户端相关概念在 [tRPC-Go Client](https://git.woa.com/trpc-go/trpc-go/tree/master/client) 有详细的说明
想了解源码的同学可以重点看一下 `ClientTransport` 的 `RoundTrip` 方法，尤其是 tRPC-Go 的这几个消息队列，生产者的逻辑基本上都在这个函数中。
`一句话概括`: 通过配置文件中的参数初始化生产者，调用相应的 api 发送消息。

## 3 实现与示例

以`kafka`消息队列为例：

### 3.1 配置文件

```yaml
client:                                            #客户端调用的后端配置
  service:                                         #针对单个后端的配置
    - name: trpc.app.server.producer               #生产者服务名自己随便定义         
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800                                 #当前这个请求最长处理时间
```

### 3.2 生产者发送消息

- 不同的组件有不同的发送消息方式，以组件 README 为准
- kafka 组件中有以下 3 种方式，其中 SendMessage 和 AsyncSendMessage 方法使用方式类似 sarama 的 SyncProducer 和 AsyncProducer，通过参数指明 topic 等配置信息。而 Produce 方法可以通过配置实现多个 topic、同步、异步等设置，将配置与代码分离，与 trpc-database 其他组件配置方式统一。大家在使用过程中可以根据自己习惯针对性的选择：
- Produce(ctx context.Context, key, value []byte) error
- SendMessage(ctx context.Context, topic string, key, value []byte) (partition int32, offset int64, err error)
- AsyncSendMessage(ctx context.Context, topic string, key, value []byte) (err error)

```go
package main

import (
    "time"
    "context"
    
    "git.code.oa.com/trpc-go/trpc-database/kafka"
    "git.code.oa.com/trpc-go/trpc-go/client"
)

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody)( err error ) {
   
    proxy := kafka.NewClientProxy("trpc.app.server.producer") // service name 自己随便填，主要用于监控上报和寻找配置项
   
    // kafka 命令
    err := proxy.Produce(ctx, key, value) // 消息发送的方式完全依赖 yaml 里面的配置
    // partition, offset, err := SendMessage(ctx, topic, key, value) // 优先使用指定的 topic 传输（同步）
    // err := AsyncSendMessage(ctx, topic, key, value) // 异步发送，不是所有消息队列都支持，支持情况看各个组件的说明
   
    // 业务逻辑
}
```

## 4 FAQ

### Q1: service 的名字怎么取

A: service name 自己随便填，主要用于监控上报和寻找配置项

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
