English | [中文](database_zh_CN.md)

# How to Develop a Database Plugin

In everyday development, we often need to access various storage systems like MySQL, Redis, Kafka, etc., for database read and write operations. While using open-source SDKs can fulfill the need for accessing databases, users are required to handle naming routing, monitoring, and configuration development on their own.

Considering that tRPC-Go provides various plugins for naming routing, monitoring, and configuration management, we can encapsulate open-source SDKs and leverage the capabilities of tRPC-Go plugins to reduce repetitive code. tRPC-Go provides wrappers for some open-source SDKs, allowing you to directly reuse tRPC-Go's naming routing and monitoring features. The code repository can be found at: https://github.com/trpc-ecosystem/go-database.

## Producer

### Define the `Client` Request Entry

Producers perform specific actions to produce messages. For example, if you want to execute a MySQL execution, you can define a `MysqlCli` struct that provides an `Exec` method. To reuse tRPC-Go plugins' capabilities, you need to enter the tRPC-Go Client layer, where various filters are called. Routing and monitoring plugins are implemented using filters. Therefore, in the `Exec` method, you call `Client.Invoke` to enter the Client layer:

```golang
type MysqlCli struct {
    serviceName string
    client      client.Client
}

func (c *MysqlCli) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    // Construct the request and response
    mreq := &Request{
        Exec: query,
        Args: args,
    }
    mrsp := &Response{}

    ctx, msg := codec.WithCloneMessage(ctx)
    defer codec.PutBackMessage(msg)
    msg.WithClientRPCName(fmt.Sprintf("/%s/Exec", c.serviceName))
    msg.WithCalleeServiceName(c.serviceName)
    msg.WithSerializationType(codec.SerializationTypeUnsupported)
    msg.WithCompressType(codec.CompressTypeNoop)

    // Put the request and response into the context
    msg.WithClientReqHead(mreq)
    msg.WithClientRspHead(mrsp)

    // Enter the Client layer and specify the transport for MySQL using WithProtocol
    err := c.client.Invoke(ctx, mreq, mrsp, client.WithProtocol("mysql"))
    if err != nil {
        return nil, err
    }

    return mrsp.Result, nil
}
```

### Implement the `transport.ClientTransport` Interface

The `transport.ClientTransport` interface has a `RoundTrip` method, which is the starting point for making network requests after the tRPC-Go Client layer ends. The goal of the database plugin is not to create a database SDK but to wrap existing database SDKs with a tRPC-Go plugin, giving them the ability to route and monitor like a tRPC-Go client.

In the `RoundTrip` method, you need to call the database's SDK to initiate specific requests. For example, if you want to execute a MySQL execution:

```golang
func init() {
    // Register the MySQL transport
    transport.RegisterClientTransport("mysql", &MysqlTransport{})
}

type MysqlTransport struct{}

func (t *MysqlTransport) RoundTrip(ctx context.Context, req []byte, opts ...transport.RoundTripOption) ([]byte, error) {
    // Retrieve the request and response from the context
    msg := codec.Message(ctx)
    req := msg.ClientReqHead().(*Request)
    rsp := msg.ClientRspHead().(*Response)

    opt := &transport.RoundTripOptions{}
    for _, o := range opts {
        o(opt)
    }

    // Ensure that the same address is only opened once
    db, err := sql.Open(ct.DriverName, opt.Address)
    rsp = db.ExecContext(ctx, req.Exec, req.Args...)
}
```

### A Better Implementation Using Hooks

While defining request entry points and transport layers can achieve request interception for databases and implement routing and monitoring, it can become verbose and challenging to maintain as the number of interfaces increases. A more efficient approach is to use Hooks provided by open-source SDKs to inject filters. Almost all routing and monitoring plugins are implemented as filters, so by executing filters, routing and monitoring functionality is achieved without entering the Client layer. Additionally, using Hooks allows you to use the original request entry points provided by the open-source SDKs, avoiding the need for additional definitions. However, not all open-source SDKs provide Hook functionality. Let's take the example of the go-redis SDK to see how to leverage Hooks for filters.

The [go-redis](github.com/redis/go-redis/v9) UniversalClient provides an `AddHook(hook)` method, which allows you to execute custom logic before and after creating connections and making requests:

```golang
type Hook interface {
    DialHook(next DialHook) DialHook
    ProcessHook(next ProcessHook) ProcessHook
    ProcessPipelineHook(next ProcessPipelineHook) ProcessPipelineHook
}
```

For example, if you want to perform routing during connection creation:

```golang
type hook struct {
    filters  *joinfilters.Filters // Filters
    selector selector.Selector    // Routing selector
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

For example, if you want to execute filters before and after making requests:

```golang
func (h *hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
    return func(ctx context.Context, cmds []redis.Cmder) error {
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

Complete code can be found here: https://github.com/trpc-ecosystem/go-database/blob/main/goredis/client.go

## Consumer

### Define Stubs

Consumer services are typically callback functions similar to regular tRPC-Go servers. You can use stubs to register user processing logic with the framework. For example, in a Kafka consumer service that uses the [sarama](https://github.com/IBM/sarama) SDK, you can define the signature of the business processing interface as follows:

```golang
type KafkaConsumer interface {
    Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}
```

Define stubs and provide a method for users to register their processing logic:

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

To ensure that the tRPC-Go framework executes the business processing logic correctly, you need to implement `server.Method.Func` to extract the `sarama.ConsumerMessage` from the context and pass it to the user's processing function:

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

### Implement the `transport.ServerTransport`

The `transport.ServerTransport` interface's `ListenAndServe` method is the entry point for network requests. In this function, you need to use the sarama consumer interface to fetch messages and pass them to users:

Complete code: https://github.com/trpc-ecosystem/go-database/blob/main/kafka/server_transport.go