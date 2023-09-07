# tRPC-Go 开发拦截器插件

## 前言

本文介绍如何开发 tRPC-Go 框架的拦截器（也称之为过滤器）。tRPC 框架利用拦截器的机制，将接口请求相关的特定逻辑组件化，插件化，从而同具体的业务逻辑解除耦合，达到复用的目的。例如监控拦截器，分布式追踪拦截器，日志拦截器，鉴权拦截器等。

## 原理

理解拦截器的原理关键点在于理解拦截器的`触发时机` 以及 `顺序性`。

触发时机：拦截器可以拦截到接口的请求和响应，并对请求，响应，上下文进行处理（用通俗的语言阐述也就是 可以在`请求接受前`做一些事情，`请求处理后`做一些事情），因此，拦截器从功能上说是分为两个部分的 前置（业务逻辑处理前） 和 后置（业务逻辑处理后）。

顺序性：如下图所示，拦截器是有明确的顺序性，根据拦截器的注册顺序依次执行前置部分逻辑，并逆序执行拦截器的后置部分。

![The Order of Filters](/.resources/developer_guide/develop_plugins/filter/filter.png)

## 示例

下面以一个 rpc 耗时统计上报拦截器进行举例说明如何开发拦截器。

第一步：如下为实现拦截器的函数原型

```golang
// ServerFilter server 耗时统计，从收到请求到返回响应的处理时间
func ServerFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error)
```

```golang
// ClientFilter client 耗时统计，从发起请求到接收响应的调用时间
func ClientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) (err error)
```

第二步：实现

```golang
func ServerFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
    begin := time.Now()        // 业务逻辑处理前打点记录时间戳

    rsp, err := next(ctx, req) // 注意这里必须用户自己调用下一个拦截器，除非有特定目的需要直接返回

    cost := time.Since(begin)  // 业务逻辑处理后计算耗时

    // 上报耗时到具体监控平台

    return rsp, err // 必须返回 next 的 rsp 和 err，要格外注意不要被自己的逻辑的 rsp 和 err 覆盖
}

func ClientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
    begin := time.Now() // 发起请求前打点记录时间戳

    err := next(ctx, req, rsp)

    cost := time.Since(begin) // 接受响应后计算耗时

    // 上报耗时到具体监控平台

    return err
}
```

第三步：将拦截器注册到框架中

```golang
filter.Register("name", ServerFilter, ClientFilter) // 拦截器名字自己随便定义，供后续配置文件使用，必须放在 trpc.NewServer() 之前
```

第四步：配置使用

```yaml
server:
  filter: # 对所有 service 全部生效
    - name1 # 上面第三步注册到框架中的 server 拦截器名字
  service:
    - name: trpc.app.server.service
      filter: # 只对当前 service 生效
        - name2

client:
  filter:
    - name
```

## 流式拦截器

因为流式服务和普通 RPC 调用接口差异较大，例如普通 RPC 的客户端通过 `proxy.SayHello`发起一次 RPC 调用，但是流式客户端通过`proxy.ClientStreamSayHello`创建一个流。流创建后，再调用`SendMsg` `RecvMsg` `CloseSend`来进行流的交互，所以针对流式服务，提供了不一样的拦截器接口。

虽然暴露的接口不同，但是底层的实现方式和普通 RPC 类似，原理参考普通 RPC 拦截器的原理

### 客户端

在客户端配置流式拦截器，需要实现`client.StreamFilter`

```golang
type StreamFilter func(context.Context, *client.ClientStreamDesc, client.Streamer) (client.ClientStream, error)
```

以流式交互过程中的耗时统计上报拦截器进行举例说明如何开发流式拦截器

**第一步**：实现`client.streamFilter`

```golang
func StreamClientFilter(ctx context.Context, desc *client.ClientStreamDesc, streamer client.Streamer) (client.ClientStream, error) {
    begin := time.Now() // 创建流之前，打点记录时间戳

    s, err := streamer(ctx, desc) // 注意这里必须用户自己调用 streamer 执行下一个拦截器，除非有特定目的需要直接返回

    cost := time.Since(begin) // 流创建完成后，计算耗时

    // 上报耗时到具体监控平台

    return &wrappedStream{s}, err // wrappedStream 封装了 client.ClientStream，用于后续拦截 SendMsg、RecvMsg 等方法。注意这里必须返回 streamer 的 err
}
```

**第二步**：封装 `client.ClientStream`，重写对应方法方法

因为流式服务的交互过程中客户端有`SendMsg`、`RecvMsg`、`CloseSend`这些方法，为了拦截这些交互过程，需要引入一个新的结构体。用户需要为这个结构体重写`client.ClientStream`接口，框架调用`client.ClientStream`接口时，会执行这个结构体的对应方法，这样就实现了拦截。

因为用户可能不需要拦截`client.ClientStream`的所有方法，所以可以将`client.ClientStream`设置为结构体的匿名字段，这样，不需要拦截的方法，会直接走原始的路径。用户需要拦截哪些方法，就在这个结构体中重写那些方法。

例如我只想拦截发送数据的过程，那么只需要重写`SendMsg`方法，至于`client.ClientStream`其他的方法都不需要重新实现。这里是为了演示，所以实现了`client.ClientStream`的所有方法。

```golang
// wrappedStream 封装原始流，需要拦截哪些方法，就重写哪些方法
type wrappedStream struct {
    client.ClientStream // 必须内嵌 client.ClientStream
}

// 重写 RecvMsg，用来拦截流的所有 RecvMsg 调用
func (w *wrappedStream) RecvMsg(m interface{}) error {
    begin := time.Now() // 接收数据之前，打点记录时间戳

    err := w.ClientStream.RecvMsg(m) // 注意这里必须用户自己调用 RecvMsg 让底层流接收数据，除非有特定目的需要直接返回

    cost := time.Since(begin) // 接收到数据后，计算耗时

    // 上报耗时到具体监控平台

    return err // 注意这里必须返回前面产生的 err
}

// 重写 SendMsg，用来拦截流的所有 SendMsg 调用
func (w *wrappedStream) SendMsg(m interface{}) error {
    begin := time.Now() // 发送数据之前，打点记录时间戳

    err := w.ClientStream.SendMsg(m) // 注意这里必须用户自己调用 SendMsg 让底层流接收数据，除非有特定目的需要直接返回

    cost := time.Since(begin) // 发送数据后，计算耗时

    // 上报耗时到具体监控平台

    return err // 注意这里必须返回前面产生的 err
}

// 重写 CloseSend，用来拦截流的所有 CloseSend 调用
func (w *wrappedStream) CloseSend() error {
    begin := time.Now() // 关闭本端之前，打点记录时间戳

    err := w.ClientStream.CloseSend() // 注意这里必须用户自己调用 CloseSend 让底层流关闭本端，除非有特定目的需要直接返回

    cost := time.Since(begin) // 关闭本端后，计算耗时

    // 上报耗时到具体监控平台

    return err // 注意这里必须返回前面产生的 err
}
```

**第三步**：将拦截器配置到 client，可以通过配置文件配置或者在代码中配置

方式 1: 在配置文件配置

先将拦截器注册到框架中

```golang
client.RegisterStreamFilter("name1", StreamClientFilter)    // 拦截器名字自己随便定义，供后续配置文件使用，必须放在 trpc.NewServer() 之前
```

再在配置文件中配置

```yaml
client:
  stream_filter: # 对所有 service 全部生效
    - name1 # 上面注册到框架中 client 流式拦截器的名字
  service:
    - name: trpc.app.server.service
      stream_filter: # 只对当前 service 生效
        - name2
```

方式 2: 在代码中配置

```golang
// 通过 client.WithStreamFilters 将拦截器添加进去
proxy := pb.NewGreeterClientProxy(client.WithStreamFilters(StreamClientFilter))

// 创建流
cstream，err := proxy.ClientStreamSayHello(ctx)

// 流的交互过程
cstream.Send()
cstream.Recv()
```

### 服务端

在服务端配置流式拦截器，需要实现`server.StreamFilter`

```golang
type StreamFilter func(Stream, *StreamServerInfo, StreamHandler) error
```

以流式交互过程中的耗时统计上报拦截器进行举例说明如何开发流式拦截器

**第一步**：实现`server.StreamFilter`

```golang
func StreamServerFilter(ss server.Stream, si *server.StreamServerInfo, handler server.StreamHandler) error {
    begin := time.Now() // 进入流式处理之前，打点记录时间戳

    // wrappedStream 封装了 server.Stream，用于后续拦截 SendMsg、RecvMsg 等方法
    ws := &wrappedStream{ss}

    // 注意这里必须用户自己调用 handler 执行下一个拦截器，除非有特定目的需要直接返回。
    err := handler(ws)

    cost := time.Since(begin) // 处理函数退出后，计算耗时

    // 上报耗时到具体监控平台

    return err // 注意这里必须返回 handler 的 err
}
```

**第二步**：封装 `server.Stream`，重写对应方法

因为流式服务的交互过程中服务端端有`SendMsg`、`RecvMsg`这些方法，为了拦截这些交互过程，需要引入一个新结构体。用户需要为这个结构体重写`server.Stream`接口，框架调用`server.Stream`接口时，会执行这个结构体的对应方法，这样就实现了拦截。

因为用户可能不需要拦截`server.Stream`的所有方法，所以可以将`server.Stream`设置为结构体的匿名字段，这样，不需要拦截的方法，会直接走原始的路径。用户需要拦截哪些方法，就在这个结构体中重写那些方法。

例如我只想拦截发送数据的过程，那么只需要重写`SendMsg`方法，至于`server.Stream`其他的方法都不需要实现。这里是为了演示，所以实现了`server.Stream`的所有方法。

```golang
// wrappedStream 封装原始流，需要拦截哪些方法，就重写哪些方法
type wrappedStream struct {
    server.Stream // 必须内嵌 server.Stream
}

// 重写 RecvMsg，用来拦截流的所有 RecvMsg 调用
func (w *wrappedStream) RecvMsg(m interface{}) error {
    begin := time.Now() // 接收数据之前，打点记录时间戳

    err := w.Stream.RecvMsg(m) // 注意这里必须用户自己调用 RecvMsg 让底层流接收数据，除非有特定目的需要直接返回

    cost := time.Since(begin) // 接收到数据后，计算耗时

    // 上报耗时到具体监控平台

    return err // 注意这里必须返回前面产生的 err
}

// 重写 SendMsg，用来拦截流的所有 SendMsg 调用
func (w *wrappedStream) SendMsg(m interface{}) error {
    begin := time.Now() // 发送数据之前，打点记录时间戳

    err := w.Stream.SendMsg(m) // 注意这里必须用户自己调用 SendMsg 让底层流接收数据，除非有特定目的需要直接返回

    cost := time.Since(begin) // 发送数据后，计算耗时

    // 上报耗时到具体监控平台

    return err // 注意这里必须返回前面产生的 err
}
```

**第三步**：将拦截器配置到 server，可以通过配置文件配置或者在代码中配置

方式 1: 在配置文件配置

先将拦截器注册到框架中

```golang
server.RegisterStreamFilter("name1", StreamServerFilter)    // 拦截器名字自己随便定义，供后续配置文件使用，必须放在 trpc.NewServer() 之前
```

再在配置文件中配置

```yaml
server:
  stream_filter: # 对所有 service 全部生效
    - name1 # 上面注册到框架中的 server 流式拦截器名字
  service:
    - name: trpc.app.server.service
      stream_filter: # 只对当前 service 生效
        - name2
```

方式 2: 在代码中配置

```golang
// 通过 server.WithStreamFilters 将拦截器添加进去
s := trpc.NewServer(server.WithStreamFilters(StreamServerFilter))

pb.RegisterGreeterService(s, &greeterServiceImpl{})
if err := s.Serve(); err != nil {
    log.Fatal(err)
}
```

## FAQ

### Q：拦截器入口这里能否拿到二进制数据

不可以，拦截器入口这里的 req rsp 都是已经经过序列化过的结构体了，可以直接使用数据，没有二进制。

### Q：多个拦截器执行顺序如何

多个拦截器的执行顺序按配置文件中的数组顺序执行，如

```yaml
server:
  filter:
    - filter1
    - filter2
  service:
    - name: trpc.app.server.service
      filter:
        - filter3
```

则执行顺序如下：

```
接收到请求 -> filter1 前置逻辑 -> filter2 前置逻辑 -> filter3 前置逻辑 -> 用户的业务处理逻辑 -> filter3 后置逻辑 -> filter2 后置逻辑 -> filter1 后置逻辑 -> 回包
```

### Q：一个拦截器必须同时设置 server 和 client 吗

不需要，只需要 server 时，client 传入 nil，同理只需要 client 时，server 传入 nil，如

```golang
filter.Register("name1", serverFilter, nil)  // 注意，此时的 name1 拦截器只能配置在 server 的 filter 列表里面，配置到 client 里面，rpc 请求会报错
filter.Register("name2", nil, clientFilter)  // 注意，此时的 name2 拦截器只能配置在 client 的 filter 列表里面，配置到 server 里面会启动失败
```
