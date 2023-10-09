[English](database.md) | 中文

# 怎么开发一个 database 类型的插件

在日常的开发过程中，大家经常会访问 MySQL、Redis、Kafka 等存储进行数据库的读写。直接使用开源的 SDK 虽然可以满足访问数据库的需求，但是用户需要自己负责路由寻址、监控上报、配置的开发。

考虑到 tRPC-Go 提供了多种多样的路由寻址、监控上报、配置管理的插件，我们可以封装一下开源的 SDK，复用 tRPC-Go 插件的能力，减少重复代码。tRPC-Go 提供了部分开源 SDK 的封装，可以直接复用 tRPC-Go 的路由寻址、监控上报功能，代码仓库：https://github.com/trpc-ecosystem/go-database。

## 生产者

### 定义 `Client` 请求入口

生产者执行具体的指令生产消息，例如希望发起 MySQL `Exec`，我们在外层定义一个 `MysqlCli` 结构体提供 `Exec` 方法，为了复用 tRPC-Go 的生态插件，我们需要进入 tRPC-Go 的 Client，在 Client 层会调用各种拦截器，其中路由寻址和监控上报插件都是使用拦截器实现的，所以，在 Exec 方法的最后，调用 `Client.Invoke` 进入 Client 层执行拦截器：

```golang
type MysqlCli struct {
    serviceName string
    client      client.Client
}

func (c *MysqlCli) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    // 构造请求和响应
    mreq := &Request{
        Exec:   query,
        Args:   args,
    }
    mrsp := &Response{}

    ctx, msg := codec.WithCloneMessage(ctx)
    defer codec.PutBackMessage(msg)
    msg.WithClientRPCName(fmt.Sprintf("/%s/Exec", c.serviceName))
    msg.WithCalleeServiceName(c.serviceName)
    msg.WithSerializationType(codec.SerializationTypeUnsupported)
    msg.WithCompressType(codec.CompressTypeNoop)

    // 将请求和响应放到 context 中
    msg.WithClientReqHead(mreq)
    msg.WithClientRspHead(mrsp)

    // 进入 Client 层，通过 WithProtocol 指定后续使用 mysql 对应的 transport
    err := c.client.Invoke(ctx, mreq, mrsp, client.WithProtocol("mysql"))
    if err != nil {
        return nil, err
    }

    return mrsp.Result, nil
}
```

### 实现 `transport.ClientTransport` 接口

`transport.ClientTransport` 接口有 `RoundTrip` 方法，是 tRPC-Go 框架 `Client` 层结束后，发起网络的请求的起点，database 插件的目标并不是实现数据库的 SDK，而是对已有的数据库 SDK 做一层 tRPC-Go 外壳的封装，使其拥有 tRPC-Go 客户端一样的寻址和监控能力。

所以我们在 `RoundTrip` 方法里，需要调用数据库的 SDK，发起具体的请求，例如希望发起一次 MySQL exec：

```golang
func init() {
    // 注册 mysql transport
    transport.RegisterClientTransport("mysql", &MysqlTransport{})
}

type MysqlTransport struct {}

func (t *MysqlTransport) RoundTrip(ctx context.Context, req []byte, opts ...transport.RoundTripOption) ([]byte, error) {
    // 从 context 取出请求和响应
    msg := codec.Message(ctx)
    req := msg.ClientReqHead().(*Request)
    rsp := msg.ClientRspHead().(*Response)

    opt := &transport.RoundTripOptions{}
    for _, o := range opts {
        o(opt)
    }

    // 保证同一个 address 只会 Open 一次
    db, err := sql.Open(ct.DriverName, opt.Address)
    rsp = db.ExecContext(ctx, req.Exec, req.Args...)
}
```

### 更好的实现方法，使用 Hook

虽然定义请求入口和 transport 层可以实现对数据库请求的拦截，实现路由寻址和监控上报。但是开源库提供的接口各种各样，为了让我们的 database 提供的接口更加完善，和开源 SDK 保持一致，那需要我们对每个接口都定义一个请求入口。但是随着接口的增加，代码会显得啰嗦，维护也会更加困难。

我们可以使用开源 SDK 提供的 Hook 功能，实现拦截器的注入，因为几乎所有的路由寻址，监控上报插件都是拦截器，只要能执行拦截器，那路由寻址和监控上报功能就实现了，不需要进入 Client 层。而且使用 Hook 的方式可以尽可能的使用开源 SDK 原始的请求入口，不需要额外定义。但并不是所有开源的 SDK 都提供了 Hook 的功能，我们以 goredis 为例，看看如何利用 Hook 来执行拦截器。

[go-redis](github.com/redis/go-redis/v9) 的 UniversalClient 提供了 AddHook(hook) 的方法，能够在创建连接和发起请求的前后执行自定义逻辑。

```golang
type Hook interface {
    DialHook(next DialHook) DialHook
    ProcessHook(next ProcessHook) ProcessHook
    ProcessPipelineHook(next ProcessPipelineHook) ProcessPipelineHook
}
```

例如我们希望在创建连接的时候，进行路由寻址：

```golang
type hook struct {
    filters    *joinfilters.Filters // 拦截器
    selector   selector.Selector    // 路由寻址选择器
}

func (h *hook) DialHook(next redis.DialHook) redis.DialHook {
    return func(ctx context.Context, network, addr string) (net.Conn, error) {
        start := time.Now()
        node, err := h.selector.Select(h.options.Service)
        if err != nil {
            return nil, err
        }
        conn, err := next(ctx, network, node.Address)
        return conn, err
    }
}
```

例如我们希望在发起请求的前后执行拦截器：

```golang
func (h *hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook { return func(ctx context.Context, cmds []redis.Cmder) error {
        ctx, redisMsg := WithMessage(ctx)
        call := func(context.Context) (string, error) {
            nextErr := next(ctx, cmds)
            rspBody := pipelineRspBody(cmds)
            return rspBody, nextErr
        }
        req := &invokeReq{
            cmds:    cmds,
            rpcName: "pipeline",
            call:    call,
        }
        req.calleeMethod, req.reqBody = pipelineReqBody(cmds)
        trpcErr := h.filters.Invoke(ctx, rReq, rRsp, func(ctx context.Context, _, _ interface{}) error {
            rRsp.Cmd, callErr = req.call(ctx)
            return TRPCErr(callErr)
        }, opts...)
    }
}
```

完整代码见：https://github.com/trpc-ecosystem/go-database/blob/main/goredis/client.go

## 消费者

### 定义桩代码

消费者服务通常是回调函数，类似于普通 tRPC-Go 服务端，通过桩代码将用户处理逻辑注册到框架中。例如 Kafka 消费服务，底层 Kafka SDK 使用 [sarama](https://github.com/IBM/sarama) 我们定义业务处理接口签名如下：

```golang
type KafkaConsumer interface {
    Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}
```

定义桩代码，提供用户处理逻辑注册方法：

```golang
func RegisterKafkaConsumerService(s server.Service, svr KafkaConsumer) {
    _ = s.Register(&KafkaConsumerServiceDesc, svr)
}

var KafkaConsumerServiceDesc = server.ServiceDesc{
    ServiceName: "trpc.kafka.consumer.service",
    HandlerType: ((*KafkaConsumer)(nil)),
    Methods: []server.Method{{
        Name: "/trpc.kafka.consumer.service/handle",
        Func: KafkaConsumerHandle,
    }},
}
```

为了能让 tRPC-Go 框架正确执行业务处理逻辑，我们需要实现 `server.Method.Func`，从 context 中拿到 `sarama.ConsumerMessage` 交给业务处理函数：

```golang
func(svr interface{}, ctx context.Context, f FilterFunc) (rspbody interface{}, err error)

func KafkaConsumerHandle(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
    filters, err := f(nil)
    if err != nil {
        return nil, err
    }
    handleFunc := func(ctx context.Context, _ interface{}, _ interface{}) error {
        msg := codec.Message(ctx)
        m, ok := msg.ServerReqHead().(*sarama.ConsumerMessage)
        if !ok {
            return errs.NewFrameError(errs.RetServerDecodeFail, "kafka consumer handler: message type invalid")
        }
        return svr.(KafkaConsumer).Handle(ctx, m)
    }
    if err := filters.Handle(ctx, nil, nil, handleFunc); err != nil {
        return nil, err
    }
    return nil, nil
}
```

### 实现 `transport.ServerTransport`

`transport.ServerTransport` 的 `ListenAndServe` 方法是网络请求的入口，在这个函数里，我们需要调于 `sarama` 的消费接口，获取到消息后给到用户：

完整代码：https://github.com/trpc-ecosystem/go-database/blob/main/kafka/server_transport.go