## 1 背景

目前公司内部有些 grpc-go 存量服务，想逐步往 trpc-go 上迁移。第一个需求是 grpc-go client 使用 grpc 协议调用 trpc-go 现有服务，不需要框架改动。这个在 trpc-codec 中引入的 grpc 来实现。

## 2 原理

`trpc-codec/grpc`设计主要思考点：

- 自定义 codec 直接能解析 grpc(idle, ready...) 各个阶段的协议包，这个比较难，需要深入到 grpc 协议框架中，还需要回包，耦合太重；(不可行)
- **直接在 codec 中创建一个 grpc server，接收到 grpc client 数据包后转发给 trpc-go service handler 框架内处理，最终交给业务逻辑 (采纳该方案)；**

## 3 实现

实现第二种方案需要考虑点：

1. 当 grpc server 接收到 grpc client 的请求后，可以正确的转发给目标 service handler 处理；
2. service handler 需要进行三个步骤：
   1). 输入流解码；
   2). 交给拦截器和上游业务逻辑 Handle 处理；
   3). 输出流编码。

需要关注的三个点是：

1. grpc server 接收到 grpc client 后，会立即进行内部的反序列化成目标 pb 结构体对象；
2. trpc-go service Handle 函数的第二个参数是 trpc-go client 的比特流，所以需要在进入该方法之前在进行一次序列化成比特流；
3. trpc-go server 请求业务逻辑处理完后，首先经过 trpc-go service handle 的序列化操作成比特流，但是 grpc-go server 返回给 grpc client 之前也需要进行一次序列化，所以在 trpc-go service handle 返回后还需要进行一次反序列化操作；

**从上面可以看出，过程确实很复杂，序列化和反序列化成组进行了 3 次。性能肯定是有影响的**

同时可以看出，grpc-go server 把请求转交给 trpc-go server 处理：

1. 需要进行一次序列化和反序列化；请求 pb 和响应 pb 都需要手动注册
2. 两者请求的转交映射关系需要靠注册的路由来匹配实现

代码实现：[trpc-codec](https://git.woa.com/trpc-go/trpc-codec/tree/master/grpc)

## 4 示例

[示例地址](https://git.woa.com/trpc-go/trpc-codec/tree/master/grpc/examples/)

- gRPC calls tRPC service
- tRPC calls gRPC service
- gRPC streaming call tRPC streaming service
- use the same stub code generated from the same pb file to write trpc and grpc services in the same code.

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
