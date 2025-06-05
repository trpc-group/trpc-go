## 1 前言

业务开发中，为了实现服务间解耦、异步处理、消峰等功能，很多场景都会选择消息队列（MQ, Message Queue），tRPC-Go 组件已经很好地支持了这些场景。

## 2 原理

在 tRPC-Go 中消费者是作为一个`Service`运行的，这样做的目的主要是为了能复用框架的服务治理能力，如自动上报监控，模调，调用链等关键信息。

框架会通过配置文件中的参数初始化消费者，起一个`死循环`不断获取最新的消息，接着调用用户注册的处理函数执行业务逻辑，并根据处理函数是否返回 error 来决定是否确认成功消费。

想了解源码的同学可以重点看一下 `ServerTransport` 的 [ListenAndServe](https://git.woa.com/trpc-go/trpc-database/blob/kafka/v0.2.4/kafka/server_transport.go#L37) 方法，消费者的逻辑基本上都在这个函数中。

## 3 实现

以`kafka`消息队列的消费者为例（更多信息参考 kafka 的 [README](https://git.woa.com/trpc-go/trpc-database/blob/master/kafka/README.md)）

### 3.1 消费者配置文件

首先需要配置消费者的远端地址`address`，并且通过`protocol`告诉框架使用哪种消息队列。

```yaml
server:                                                                                   #服务端配置
  service:                                                                                #业务服务提供的 service，可以有多个
    - name: trpc.app.server.consumer                                                      #service 的路由名称 如果使用的是 123 平台，需要使用 trpc.${app}.${server}.consumer  
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&&version=x.x.x.x        #kafka consumer broker address，version 如果不设置则为 1.1.1.0，部分 ckafka 需要指定 0.10.2.0
      protocol: kafka                                                                     #应用层协议 
      timeout: 1000                                                                       #请求最长处理时间 单位 毫秒

```

### 3.2 实现消息处理函数

消息处理函数需要用户自己实现，如下图的 handle 函数，并注册到框架中。
只有当处理函数返回成功 nil，才会确认消费成功，返回 err 不会确认成功，会等待 3s 重新消费，会有重复消息，一定要保证处理函数幂等性。

```go
package main

import (
 "context"
 
 "git.code.oa.com/trpc-go/trpc-database/kafka"
 trpc "git.code.oa.com/trpc-go/trpc-go"
)

func main() {

 s := trpc.NewServer()
    // 启动多个消费者的情况，可以配置多个 service，然后这里任意匹配 kafka.RegisterKafkaConsumerService(s.Service("name"), handle)，没有指定 name 的情况，代表所有 service 共用同一个 handler
    kafka.RegisterKafkaConsumerService(s, handle) 
    s.Serve()
}

// 只有返回成功 nil，才会确认消费成功，返回 err 不会确认成功，会等待 3s 重新消费，会有重复消息，一定要保证处理函数幂等性
func handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
    return nil
}
```

## 4 示例

截止目前已经支持的消息队列组件如下（最新组件列表请到 [git 仓库](https://git.woa.com/trpc-go/trpc-database) 中查阅）:

### 4.1 hippo

[hippo](https://git.woa.com/trpc-go/trpc-database/tree/master/hippo)  PCG 类 kafka 消息队列

### 4.2 kafka

[kafka](https://git.woa.com/trpc-go/trpc-database/tree/master/kafka) 开源消息队列

### 4.3 rabbitmq

[rabbitmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rabbitmq)  开源消息队列

### 4.4 rocketmq

[rocketmq](https://git.woa.com/trpc-go/trpc-database/tree/master/rocketmq) 开源消息队列

### 4.5 tdmq

[tdmq](https://git.woa.com/trpc-go/trpc-database/tree/master/tdmq) 腾讯云的 tdmq 消息队列

### 4.6 tube

[tube](https://git.woa.com/trpc-go/trpc-database/tree/master/tube) tdbank 消息队列

### 4.7 redis

[redis](https://git.woa.com/pcg-csd/trpc-ext/redis/tree/master/trpc/mq) redis 消息队列

## 5 FAQ

请参考服务端开发向导的 [FAQ](https://iwiki.woa.com/p/284289102#11-faq) 部分。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
